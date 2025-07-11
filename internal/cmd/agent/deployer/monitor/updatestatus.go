// Package monitor provides functionality for monitoring and updating the status of a bundle deployment.
// It includes functions for determining whether the agent should be redeployed, whether the status should be updated,
// and for updating the status based on the resources and helm release history.
package monitor

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/rancher/fleet/internal/cmd/agent/deployer/desiredset"
	"github.com/rancher/fleet/internal/cmd/agent/deployer/objectset"
	"github.com/rancher/fleet/internal/cmd/agent/deployer/summary"
	"github.com/rancher/fleet/internal/helmdeployer"
	fleet "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
)

// limit the length of nonReady and modified resources
const resourcesDetailsMaxLength = 10

type Monitor struct {
	client     client.Client
	desiredset *desiredset.Client

	deployer *helmdeployer.Helm

	defaultNamespace string
	labelPrefix      string
	labelSuffix      string
}

func New(client client.Client, ds *desiredset.Client, deployer *helmdeployer.Helm, defaultNamespace string, labelSuffix string) *Monitor {
	return &Monitor{
		client:           client,
		desiredset:       ds,
		deployer:         deployer,
		defaultNamespace: defaultNamespace,
		labelPrefix:      defaultNamespace,
		labelSuffix:      labelSuffix,
	}
}

func ShouldRedeployAgent(bd *fleet.BundleDeployment) bool {
	if isAgent(bd) {
		return true
	}
	if bd.Spec.Options.ForceSyncGeneration <= 0 {
		return false
	}
	if bd.Status.SyncGeneration == nil {
		return true
	}
	return *bd.Status.SyncGeneration != bd.Spec.Options.ForceSyncGeneration
}

func isAgent(bd *fleet.BundleDeployment) bool {
	return strings.HasPrefix(bd.Name, "fleet-agent")
}

// ShouldUpdateStatus skips resource and ready status updates if the bundle
// deployment is unchanged or not installed yet.
func ShouldUpdateStatus(bd *fleet.BundleDeployment) bool {
	if bd.Spec.DeploymentID != bd.Status.AppliedDeploymentID {
		return false
	}

	// If the bundle failed to install the status should not be updated. Updating
	// here would remove the condition message that was previously set on it.
	if Cond(fleet.BundleDeploymentConditionInstalled).IsFalse(bd) {
		return false
	}

	return true
}

// UpdateStatus sets the status of the bundledeployment based on the resources from the helm release history and the live state.
// In the status it updates: Ready, NonReadyStatus, IncompleteState, NonReadyStatus, NonModified, ModifiedStatus, Resources and ResourceCounts fields.
// Additionally it sets the Ready condition either from the NonReadyStatus or the NonModified status field.
func (m *Monitor) UpdateStatus(ctx context.Context, bd *fleet.BundleDeployment, resources *helmdeployer.Resources) (fleet.BundleDeploymentStatus, error) {
	logger := log.FromContext(ctx).WithName("update-status")
	ctx = log.IntoContext(ctx, logger)

	// updateFromPreviousDeployment mutates bd.Status, so copy it first
	origStatus := *bd.Status.DeepCopy()
	bd = bd.DeepCopy()
	err := m.updateFromPreviousDeployment(ctx, bd, resources)
	if err != nil {

		// Returning an error will cause UpdateStatus to requeue in a loop.
		// When there is no resourceID the error should be on the status. Without
		// the ID we do not have the information to lookup the resources to
		// compute the plan and discover the state of resources.
		if err == helmdeployer.ErrNoResourceID {
			return origStatus, nil
		}

		return origStatus, err
	}

	status := bd.Status
	status.SyncGeneration = &bd.Spec.Options.ForceSyncGeneration

	readyError := readyError(status)
	Cond(fleet.BundleDeploymentConditionReady).SetError(&status, "", readyError)
	if readyError != nil {
		logger.Info("Status not ready according to nonModified and nonReady", "nonModified", status.NonModified, "nonReady", status.NonReadyStatus)
	} else {
		logger.V(1).Info("Status ready, Ready condition set to true")
	}

	removePrivateFields(&status)
	return status, nil
}

// removePrivateFields removes fields from the status, which won't be marshalled to JSON.
// They would however trigger a status update in apply
func removePrivateFields(s1 *fleet.BundleDeploymentStatus) {
	for id := range s1.NonReadyStatus {
		s1.NonReadyStatus[id].Summary.Relationships = nil
		s1.NonReadyStatus[id].Summary.Attributes = nil
	}
}

// readyError returns an error based on the provided status.
// That error is non-nil if the status corresponds to a non-ready or modified state of the bundle deployment.
func readyError(status fleet.BundleDeploymentStatus) error {
	if status.Ready && status.NonModified {
		return nil
	}

	var msg string
	if !status.Ready {
		msg = "not ready"
		if len(status.NonReadyStatus) > 0 {
			msg = status.NonReadyStatus[0].String()
		}
	} else if !status.NonModified {
		msg = "out of sync"
		if len(status.ModifiedStatus) > 0 {
			msg = status.ModifiedStatus[0].String()
		}
	}

	return errors.New(msg)
}

// updateFromPreviousDeployment updates the status with information from the
// helm release history and an apply dry run.
// Modified resources are resources that have changed from the previous helm release.
func (m *Monitor) updateFromPreviousDeployment(ctx context.Context, bd *fleet.BundleDeployment, resources *helmdeployer.Resources) error {
	resourcesPreviousRelease, err := m.deployer.ResourcesFromPreviousReleaseVersion(bd.Name, bd.Status.Release)
	if err != nil {
		return err
	}

	ns := resources.DefaultNamespace
	if ns == "" {
		ns = m.defaultNamespace
	}

	// resources.Objects contains the desired state of the resources from helm history
	plan, err := m.desiredset.Plan(ctx, ns, desiredset.GetSetID(bd.Name, m.labelPrefix, m.labelSuffix), resources.Objects...)
	if err != nil {
		return err
	}

	// dryrun.Diff only takes plan.Update into account. plan.Update
	// contains objects which have changes to existing values. Adding a new
	// key to a map is not considered an update.
	plan, err = desiredset.Diff(plan, bd, resources.DefaultNamespace, resources.Objects...)
	if err != nil {
		return err
	}

	nonReadyResources := nonReady(ctx, plan, bd.Spec.Options.IgnoreOptions)
	modifiedResources := modified(ctx, m.client, plan, resourcesPreviousRelease)
	allResources, err := toBundleDeploymentResources(m.client, plan.Objects, resources.DefaultNamespace)
	if err != nil {
		return err
	}

	updateFromResources(&bd.Status, allResources, nonReadyResources, modifiedResources)
	return nil
}

func toBundleDeploymentResources(client client.Client, objs []runtime.Object, defaultNamespace string) ([]fleet.BundleDeploymentResource, error) {
	res := make([]fleet.BundleDeploymentResource, 0, len(objs))
	for _, obj := range objs {
		ma, err := meta.Accessor(obj)
		if err != nil {
			return nil, err
		}

		ns := ma.GetNamespace()
		gvk := obj.GetObjectKind().GroupVersionKind()
		if ns == "" && isNamespaced(client.RESTMapper(), gvk) {
			ns = defaultNamespace
		}

		version, kind := gvk.ToAPIVersionAndKind()
		res = append(res, fleet.BundleDeploymentResource{
			Kind:       kind,
			APIVersion: version,
			Namespace:  ns,
			Name:       ma.GetName(),
			CreatedAt:  ma.GetCreationTimestamp(),
		})
	}
	return res, nil
}

func updateFromResources(bdStatus *fleet.BundleDeploymentStatus, resources []fleet.BundleDeploymentResource, nonReadyResources []fleet.NonReadyStatus, modifiedResources []fleet.ModifiedStatus) {
	bdStatus.Ready = len(nonReadyResources) == 0
	bdStatus.NonReadyStatus = nonReadyResources
	if len(bdStatus.NonReadyStatus) > resourcesDetailsMaxLength {
		bdStatus.IncompleteState = true
		bdStatus.NonReadyStatus = nonReadyResources[:resourcesDetailsMaxLength]
	}

	bdStatus.NonModified = len(modifiedResources) == 0
	bdStatus.ModifiedStatus = modifiedResources
	if len(bdStatus.ModifiedStatus) > resourcesDetailsMaxLength {
		bdStatus.IncompleteState = true
		bdStatus.ModifiedStatus = modifiedResources[:resourcesDetailsMaxLength]
	}

	bdStatus.Resources = resources
	bdStatus.ResourceCounts = calculateResourceCounts(resources, nonReadyResources, modifiedResources)
}

func calculateResourceCounts(all []fleet.BundleDeploymentResource, nonReady []fleet.NonReadyStatus, modified []fleet.ModifiedStatus) fleet.ResourceCounts {
	// Create a map with all different resource keys, then remove modified or non-ready keys
	resourceKeys := make(map[fleet.ResourceKey]struct{}, len(all))
	for _, r := range all {
		resourceKeys[fleet.ResourceKey{
			Kind:       r.Kind,
			APIVersion: r.APIVersion,
			Namespace:  r.Namespace,
			Name:       r.Name,
		}] = struct{}{}
	}

	// The agent must have enough visibility to determine the exact state of every resource.
	// e.g. "WaitApplied" or "Unknown" states do not make sense in this context
	counts := fleet.ResourceCounts{
		DesiredReady: calculateDesiredReady(resourceKeys, modified),
	}
	for _, r := range modified {
		if r.Create {
			counts.Missing++
		} else if r.Delete {
			counts.Orphaned++
		} else {
			counts.Modified++
		}
		delete(resourceKeys, fleet.ResourceKey{
			Kind:       r.Kind,
			APIVersion: r.APIVersion,
			Namespace:  r.Namespace,
			Name:       r.Name,
		})
	}
	for _, r := range nonReady {
		key := fleet.ResourceKey{
			Kind:       r.Kind,
			APIVersion: r.APIVersion,
			Namespace:  r.Namespace,
			Name:       r.Name,
		}
		// If not present, it was already accounted for as "modified"
		if _, ok := resourceKeys[key]; ok {
			counts.NotReady++
			delete(resourceKeys, key)
		}
	}

	// Remaining keys are considered ready
	counts.Ready = len(resourceKeys)

	return counts
}

// calculateDesiredReady retrieves the number of total resources to be deployed.
// A ResourceKey set is obtained from plan.Objects, which only includes living objects in the cluster, so it needs to be extended with  resources in "Missing" state.
func calculateDesiredReady(liveResourceKeys map[fleet.ResourceKey]struct{}, modified []fleet.ModifiedStatus) int {
	desired := len(liveResourceKeys)
	for _, r := range modified {
		if !r.Create {
			continue
		}
		// Missing resource state
		// Increase desired count if not already present in the resource keys set
		if _, ok := liveResourceKeys[fleet.ResourceKey{
			Kind:       r.Kind,
			APIVersion: r.APIVersion,
			Namespace:  r.Namespace,
			Name:       r.Name,
		}]; !ok {
			desired++
		}
	}
	return desired
}

func nonReady(ctx context.Context, plan desiredset.Plan, ignoreOptions *fleet.IgnoreOptions) (result []fleet.NonReadyStatus) {
	logger := log.FromContext(ctx)
	defer func() {
		sort.Slice(result, func(i, j int) bool {
			return result[i].UID < result[j].UID
		})
	}()

	for _, obj := range plan.Objects {
		if u, ok := obj.(*unstructured.Unstructured); ok {
			if ignoreOptions != nil && ignoreOptions.Conditions != nil {
				if err := excludeIgnoredConditions(u, ignoreOptions); err != nil {
					logger.Error(err, "failed to ignore conditions")
				}
			}

			sum := summary.Summarize(u)
			if !sum.IsReady() {
				result = append(result, fleet.NonReadyStatus{
					UID:        u.GetUID(),
					Kind:       u.GetKind(),
					APIVersion: u.GetAPIVersion(),
					Namespace:  u.GetNamespace(),
					Name:       u.GetName(),
					Summary:    sum,
				})
			}
		}
	}

	return result
}

// modified returns a list of modified statuses based on the provided plan and previous release resources.
// The function iterates through the plan's create, delete, and update actions and constructs a modified status
// for each resource.
// If the number of modified statuses exceeds 10, the function stops and returns the current result.
func modified(ctx context.Context, c client.Client, plan desiredset.Plan, resourcesPreviousRelease *helmdeployer.Resources) (result []fleet.ModifiedStatus) {
	logger := log.FromContext(ctx)
	defer func() {
		sort.Slice(result, func(i, j int) bool {
			return sortKey(result[i]) < sortKey(result[j])
		})
	}()
	for gvk, keys := range plan.Create {
		apiVersion, kind := gvk.ToAPIVersionAndKind()
		for _, key := range keys {
			obj := &unstructured.Unstructured{}
			obj.SetGroupVersionKind(gvk)
			key := client.ObjectKey{
				Namespace: key.Namespace,
				Name:      key.Name,
			}
			err := c.Get(ctx, key, obj)

			exists := !apierrors.IsNotFound(err)

			if exists {
				logger.Info("Resource of BundleDeployment not owned by us",
					"resourceName", key.Name,
					"resourceKind", kind,
					"resourceApiVersion", apiVersion,
					"resourceNamespace", key.Namespace,
					"resourceLabels", obj.GetLabels(),
					"resourceAnnotations", obj.GetAnnotations(),
				)
			}

			result = append(result, fleet.ModifiedStatus{
				Kind:       kind,
				APIVersion: apiVersion,
				Namespace:  key.Namespace,
				Name:       key.Name,
				Create:     true,
				Exist:      exists,
			})
		}
	}

	for gvk, keys := range plan.Delete {
		apiVersion, kind := gvk.ToAPIVersionAndKind()
		for _, key := range keys {
			// Check if resource was in a previous release. It is possible that some operators copy the
			// objectset.rio.cattle.io/hash label into a dynamically created objects. We need to skip these resources
			// because they are not part of the release, and they would appear as orphaned.
			// https://github.com/rancher/fleet/issues/1141
			if isResourceInPreviousRelease(key, kind, resourcesPreviousRelease.Objects) {
				result = append(result, fleet.ModifiedStatus{
					Kind:       kind,
					APIVersion: apiVersion,
					Namespace:  key.Namespace,
					Name:       key.Name,
					Delete:     true,
				})
			}
		}
	}

	for gvk, patches := range plan.Update {
		apiVersion, kind := gvk.ToAPIVersionAndKind()
		for key, patch := range patches {
			result = append(result, fleet.ModifiedStatus{
				Kind:       kind,
				APIVersion: apiVersion,
				Namespace:  key.Namespace,
				Name:       key.Name,
				Patch:      patch,
			})
		}
	}

	return result
}

func isResourceInPreviousRelease(key objectset.ObjectKey, kind string, objsPreviousRelease []runtime.Object) bool {
	for _, obj := range objsPreviousRelease {
		metadata, _ := meta.Accessor(obj)
		if obj.GetObjectKind().GroupVersionKind().Kind == kind && metadata.GetName() == key.Name {
			return true
		}
	}

	return false
}

// excludeIgnoredConditions removes the conditions that are included in ignoreOptions from the object passed as a parameter
func excludeIgnoredConditions(obj *unstructured.Unstructured, ignoreOptions *fleet.IgnoreOptions) error {
	if ignoreOptions == nil {
		return nil
	}

	conditions, _, err := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if err != nil {
		return err
	}
	conditionsWithoutIgnored := make([]interface{}, 0)

	for _, condition := range conditions {
		condition, ok := condition.(map[string]interface{})
		if !ok {
			return fmt.Errorf("condition: %#v can't be converted to map[string]interface{}", condition)
		}
		excludeCondition := false
		for _, ignoredCondition := range ignoreOptions.Conditions {
			if shouldExcludeCondition(condition, ignoredCondition) {
				excludeCondition = true
				break
			}
		}
		if !excludeCondition {
			conditionsWithoutIgnored = append(conditionsWithoutIgnored, condition)
		}
	}

	err = unstructured.SetNestedSlice(obj.Object, conditionsWithoutIgnored, "status", "conditions")
	if err != nil {
		return err
	}

	return nil
}

// shouldExcludeCondition returns true if all the elements of ignoredConditions are inside conditions
func shouldExcludeCondition(conditions map[string]interface{}, ignoredConditions map[string]string) bool {
	if len(ignoredConditions) > len(conditions) {
		return false
	}

	for k, v := range ignoredConditions {
		if vc, found := conditions[k]; !found || vc != v {
			return false
		}
	}

	return true
}

func isNamespaced(mapper meta.RESTMapper, gvk schema.GroupVersionKind) bool {
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return true
	}
	return mapping.Scope.Name() == meta.RESTScopeNameNamespace
}

func sortKey(f fleet.ModifiedStatus) string {
	return f.APIVersion + "/" + f.Kind + "/" + f.Namespace + "/" + f.Name
}
