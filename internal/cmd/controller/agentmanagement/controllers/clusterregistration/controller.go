// Package clusterregistration implements manager-initiated and agent-initiated registration.
//
// Add or import downstream clusters / agents to Fleet and keep information
// from their registration (e.g. local cluster kubeconfig) up-to-date.
package clusterregistration

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/rancher/fleet/internal/cmd/controller/agentmanagement/controllers/resources"
	secretutil "github.com/rancher/fleet/internal/cmd/controller/agentmanagement/secret"
	"github.com/rancher/fleet/internal/config"
	"github.com/rancher/fleet/internal/names"
	"github.com/rancher/fleet/internal/registration"
	fleet "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	"github.com/rancher/fleet/pkg/durations"
	fleetcontrollers "github.com/rancher/fleet/pkg/generated/controllers/fleet.cattle.io/v1alpha1"

	"github.com/rancher/wrangler/v3/pkg/apply"
	corecontrollers "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	rbaccontrollers "github.com/rancher/wrangler/v3/pkg/generated/controllers/rbac/v1"
	"github.com/rancher/wrangler/v3/pkg/generic"
	"github.com/rancher/wrangler/v3/pkg/relatedresource"

	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	AgentCredentialSecretType     = "fleet.cattle.io/agent-credential" // nolint:gosec // not a credential
	clusterByClientID             = "clusterByClientID"
	clusterRegistrationByClientID = "clusterRegistrationByClientID"
	deleteSecretAfter             = durations.ClusterRegistrationDeleteDelay
)

type Handler struct {
	SystemNamespace             string
	SystemRegistrationNamespace string
	ClusterRegistration         fleetcontrollers.ClusterRegistrationController
	ClusterCache                fleetcontrollers.ClusterCache
	Clusters                    fleetcontrollers.ClusterClient
	ServiceAccountCache         corecontrollers.ServiceAccountCache
	SecretsCache                corecontrollers.SecretCache
	Secrets                     corecontrollers.SecretController
}

func Register(ctx context.Context,
	apply apply.Apply,
	systemNamespace string,
	systemRegistrationNamespace string,
	serviceAccount corecontrollers.ServiceAccountController,
	secret corecontrollers.SecretController,
	role rbaccontrollers.RoleController,
	roleBinding rbaccontrollers.RoleBindingController,
	clusterRegistration fleetcontrollers.ClusterRegistrationController,
	clusters fleetcontrollers.ClusterController) {
	h := &Handler{
		SystemNamespace:             systemNamespace,
		SystemRegistrationNamespace: systemRegistrationNamespace,
		ClusterRegistration:         clusterRegistration,
		ClusterCache:                clusters.Cache(),
		Clusters:                    clusters,
		ServiceAccountCache:         serviceAccount.Cache(),
		Secrets:                     secret,
		SecretsCache:                secret.Cache(),
	}

	fleetcontrollers.RegisterClusterRegistrationGeneratingHandler(ctx,
		clusterRegistration,
		apply.WithCacheTypes(serviceAccount,
			secret,
			role,
			roleBinding,
		),
		"",
		"cluster-registration",
		h.OnChange,
		&generic.GeneratingHandlerOptions{
			AllowClusterScoped: true,
		})

	secret.OnChange(ctx, "registration-expire", h.OnSecretChange)
	clusters.OnChange(ctx, "cluster-to-clusterregistration", h.OnCluster)
	clusters.Cache().AddIndexer(clusterByClientID, func(obj *fleet.Cluster) ([]string, error) {
		return []string{
			fmt.Sprintf("%s/%s", obj.Namespace, obj.Spec.ClientID),
		}, nil
	})
	clusterRegistration.Cache().AddIndexer(clusterRegistrationByClientID, func(obj *fleet.ClusterRegistration) ([]string, error) {
		return []string{
			fmt.Sprintf("%s/%s", obj.Namespace, obj.Spec.ClientID),
		}, nil
	})
	relatedresource.Watch(ctx, "sa-to-cluster-registration", saToClusterRegistration, clusterRegistration, serviceAccount)
}

func saToClusterRegistration(namespace, name string, obj runtime.Object) ([]relatedresource.Key, error) {
	if sa, ok := obj.(*v1.ServiceAccount); ok {
		ns := sa.Annotations[fleet.ClusterRegistrationNamespaceAnnotation]
		name := sa.Annotations[fleet.ClusterRegistrationAnnotation]
		if ns != "" && name != "" {
			return []relatedresource.Key{{
				Namespace: ns,
				Name:      name,
			}}, nil
		}
	}
	return nil, nil
}

func (h *Handler) OnCluster(key string, cluster *fleet.Cluster) (*fleet.Cluster, error) {
	if cluster == nil || cluster.Status.Namespace == "" {
		return cluster, nil
	}

	crs, err := h.ClusterRegistration.Cache().GetByIndex(clusterRegistrationByClientID,
		fmt.Sprintf("%s/%s", cluster.Namespace, cluster.Spec.ClientID))
	if err != nil {
		return nil, err
	}
	for _, cr := range crs {
		if !cr.Status.Granted {
			logrus.Infof("Namespace assigned to cluster '%s/%s' enqueues cluster registration '%s/%s'", cluster.Namespace, cluster.Name,
				cr.Namespace, cr.Name)
			h.ClusterRegistration.Enqueue(cr.Namespace, cr.Name)
		}
	}

	return cluster, nil
}

func (h *Handler) OnSecretChange(key string, secret *v1.Secret) (*v1.Secret, error) {
	if secret == nil || secret.Namespace != h.SystemRegistrationNamespace ||
		secret.Labels[fleet.ClusterAnnotation] == "" {
		return secret, nil
	}

	if time.Since(secret.CreationTimestamp.Time) > deleteSecretAfter {
		logrus.Infof("Deleting expired registration secret %s/%s", secret.Namespace, secret.Name)
		return secret, h.Secrets.Delete(secret.Namespace, secret.Name, nil)
	}

	h.Secrets.EnqueueAfter(secret.Namespace, secret.Name, deleteSecretAfter/2)
	return secret, nil
}

// OnChange creates the service account and roles for a cluster registration.
// The service account's token is deployed to the downstream cluster, via the
// fleet-secret. It allows the downstream fleet-agent to list
// bundledeployments and update their status in its own cluster namespace on upstream.
// It can also get content resources, but not list them. The name of content
// resources is random.
func (h *Handler) OnChange(request *fleet.ClusterRegistration, status fleet.ClusterRegistrationStatus) ([]runtime.Object, fleet.ClusterRegistrationStatus, error) {
	if status.Granted {
		// only create the cluster for the request once
		return nil, status, generic.ErrSkip
	}

	cluster, err := h.createOrGetCluster(request)
	if err != nil || cluster == nil {
		return nil, status, err
	}

	if cluster.Status.Namespace == "" {
		status.ClusterName = cluster.Name
		return nil, status, nil
	}

	// set the Cluster as owner of the cluster registration request
	// ownerFound is used to avoid calling update on request whenever OnChange is called
	ownerFound := false
	for _, owner := range request.OwnerReferences {
		if owner.Kind == "Cluster" && owner.Name == cluster.Name && owner.UID == cluster.UID {
			ownerFound = true
			break
		}
	}
	if !ownerFound {
		request.SetOwnerReferences([]metav1.OwnerReference{
			{
				APIVersion: fleet.SchemeGroupVersion.String(),
				Kind:       "Cluster",
				Name:       cluster.Name,
				UID:        cluster.UID,
			},
		})
		request, err = h.ClusterRegistration.Update(request)
		if err != nil {
			return nil, status, err
		}
	}

	saName := names.SafeConcatName(request.Name, string(request.UID))
	sa, err := h.ServiceAccountCache.Get(cluster.Status.Namespace, saName)
	if err != nil && apierrors.IsNotFound(err) {
		// create request service account if missing
		status.ClusterName = cluster.Name
		return []runtime.Object{requestSA(saName, cluster, request)}, status, nil
	} else if err != nil {
		return nil, status, fmt.Errorf("failed to retrieve service account from cache: %w", err)
	}

	// try to get request service account's token
	var secret *v1.Secret
	if secret, err = h.authorizeCluster(sa, cluster, request); err != nil {
		return nil, status, fmt.Errorf("failed to authorize cluster, cannot get service account token: %w", err)
	} else if secret == nil {
		status.ClusterName = cluster.Name
		logrus.Infof("Cluster registration request '%s/%s', cluster '%s/%s' not granted, waiting for service account token",
			request.Namespace, request.Name, cluster.Namespace, cluster.Name)
		return nil, status, nil
	}

	// delete old cluster registrations
	crlist, _ := h.ClusterRegistration.List(request.Namespace, metav1.ListOptions{})
	for _, creg := range crlist.Items {
		if shouldDelete(creg, *request) {
			logrus.Debugf("Deleting old clusterregistration '%s/%s', now at '%s'", creg.Namespace, creg.Name, request.Name)
			if err := h.ClusterRegistration.Delete(creg.Namespace, creg.Name, nil); err != nil && !apierrors.IsNotFound(err) {
				return nil, status, err
			}
		}
	}

	// request is granted, create the registration secret and roles
	status.ClusterName = cluster.Name
	status.Granted = true

	logrus.Infof("Cluster registration request '%s/%s' granted, creating cluster, request service account, registration secret", request.Namespace, request.Name)

	return []runtime.Object{
		// the registration secret c-clientID-clientRandom
		secret,
		// Update the existing service account 'request-UID' in the
		// cluster namespace, e.g. 'cluster-fleet-default-NAME-ID'
		requestSA(saName, cluster, request),
		// Add role bindings to manage bundledeployments and contents,
		// the agent could previously only access secrets in
		// 'cattle-fleet-clusters-system' and clusterregistrations in
		// the cluster registration namespace (e.g. 'fleet-default'). See
		// clusterregistrationtoken controller for details.
		&rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      request.Name,
				Namespace: request.Namespace,
				Labels: map[string]string{
					fleet.ManagedLabel: "true",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					Verbs:         []string{"patch"},
					APIGroups:     []string{fleet.SchemeGroupVersion.Group},
					Resources:     []string{fleet.ClusterResourceNamePlural + "/status"},
					ResourceNames: []string{cluster.Name},
				},
			},
		},
		&rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      request.Name,
				Namespace: request.Namespace,
				Labels: map[string]string{
					fleet.ManagedLabel: "true",
				},
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      saName,
					Namespace: cluster.Status.Namespace,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "Role",
				Name:     request.Name,
			},
		},
		// cluster role "fleet-bundle-deployment" created when
		// fleet-controller starts
		&rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      request.Name,
				Namespace: cluster.Status.Namespace,
				Labels: map[string]string{
					fleet.ManagedLabel: "true",
				},
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      saName,
					Namespace: cluster.Status.Namespace,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     resources.BundleDeploymentClusterRole,
			},
		},
		// cluster role "fleet-content" created when fleet-controller
		// starts
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: names.SafeConcatName(request.Name, "content"),
				Labels: map[string]string{
					fleet.ManagedLabel: "true",
				},
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      saName,
					Namespace: cluster.Status.Namespace,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     resources.ContentClusterRole,
			},
		},
	}, status, nil
}

// shouldDelete returns true for any other cluster registration with the same clientID, but different random and older creation timestamp
func shouldDelete(creg fleet.ClusterRegistration, request fleet.ClusterRegistration) bool {
	return creg.Spec.ClientID == request.Spec.ClientID &&
		creg.Spec.ClientRandom != request.Spec.ClientRandom &&
		creg.Name != request.Name &&
		creg.CreationTimestamp.Time.Before(request.CreationTimestamp.Time)
}

func (h *Handler) createOrGetCluster(request *fleet.ClusterRegistration) (*fleet.Cluster, error) {
	clusters, err := h.ClusterCache.GetByIndex(clusterByClientID, fmt.Sprintf("%s/%s", request.Namespace, request.Spec.ClientID))
	if err == nil && len(clusters) > 0 {
		return clusters[0], nil
	} else if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}

	clusterName := names.SafeConcatName("cluster", names.KeyHash(request.Spec.ClientID))
	if cluster, err := h.ClusterCache.Get(request.Namespace, clusterName); !apierrors.IsNotFound(err) {
		if cluster.Spec.ClientID != request.Spec.ClientID {
			// This would happen with a hash collision
			return nil, fmt.Errorf("non-matching ClientID on cluster %s/%s got %s expected %s",
				request.Namespace, clusterName, cluster.Spec.ClientID, request.Spec.ClientID)
		}
		return cluster, err
	}

	// need to create the cluster for agent initiated registration, local
	// and managed clusters would already exist
	labels := map[string]string{}
	if !config.Get().IgnoreClusterRegistrationLabels {
		for k, v := range request.Spec.ClusterLabels {
			labels[k] = v
		}
	}
	labels[fleet.ClusterAnnotation] = clusterName

	cluster, err := h.Clusters.Create(&fleet.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: request.Namespace,
			Labels:    labels,
		},
		Spec: fleet.ClusterSpec{
			ClientID: request.Spec.ClientID,
		},
	})
	if apierrors.IsAlreadyExists(err) {
		return h.Clusters.Get(request.Namespace, clusterName, metav1.GetOptions{})
	}
	if err == nil {
		logrus.Infof("Created cluster %s/%s", request.Namespace, clusterName)
	}
	return cluster, err
}

func (h *Handler) authorizeCluster(sa *v1.ServiceAccount, cluster *fleet.Cluster, req *fleet.ClusterRegistration) (*v1.Secret, error) {
	var secret *v1.Secret
	var err error
	if len(sa.Secrets) != 0 {
		secret, err = h.SecretsCache.Get(sa.Namespace, sa.Secrets[0].Name)
		if apierrors.IsNotFound(err) {
			// secrets can be slow to propagate to the cache
			secret, err = h.Secrets.Get(sa.Namespace, sa.Secrets[0].Name, metav1.GetOptions{})
		}
	} else {
		secret, err = secretutil.GetServiceAccountTokenSecret(sa, h.Secrets)
	}
	if err != nil || secret == nil {
		return nil, err
	}
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      registration.SecretName(req.Spec.ClientID, req.Spec.ClientRandom),
			Namespace: h.SystemRegistrationNamespace,
			Labels: map[string]string{
				fleet.ClusterAnnotation: cluster.Name,
				fleet.ManagedLabel:      "true",
			},
		},
		Type: AgentCredentialSecretType,
		Data: map[string][]byte{
			"token":               secret.Data["token"],
			"deploymentNamespace": []byte(cluster.Status.Namespace),
			"clusterNamespace":    []byte(cluster.Namespace),
			"clusterName":         []byte(cluster.Name),
			"systemNamespace":     []byte(h.SystemNamespace),
		},
	}, nil
}

func requestSA(saName string, cluster *fleet.Cluster, request *fleet.ClusterRegistration) *v1.ServiceAccount {
	return &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: cluster.Status.Namespace,
			Labels: map[string]string{
				fleet.ManagedLabel: "true",
			},
			Annotations: map[string]string{
				fleet.ClusterAnnotation:                      cluster.Name,
				fleet.ClusterRegistrationAnnotation:          request.Name,
				fleet.ClusterRegistrationNamespaceAnnotation: request.Namespace,
			},
		},
	}
}
