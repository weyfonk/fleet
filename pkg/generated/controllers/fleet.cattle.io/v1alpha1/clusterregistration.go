/*
Copyright (c) 2020 - 2023 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by main. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"sync"
	"time"

	v1alpha1 "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	"github.com/rancher/lasso/pkg/client"
	"github.com/rancher/lasso/pkg/controller"
	"github.com/rancher/wrangler/pkg/apply"
	"github.com/rancher/wrangler/pkg/condition"
	"github.com/rancher/wrangler/pkg/generic"
	"github.com/rancher/wrangler/pkg/kv"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

type ClusterRegistrationHandler func(string, *v1alpha1.ClusterRegistration) (*v1alpha1.ClusterRegistration, error)

type ClusterRegistrationController interface {
	generic.ControllerMeta
	ClusterRegistrationClient

	OnChange(ctx context.Context, name string, sync ClusterRegistrationHandler)
	OnRemove(ctx context.Context, name string, sync ClusterRegistrationHandler)
	Enqueue(namespace, name string)
	EnqueueAfter(namespace, name string, duration time.Duration)

	Cache() ClusterRegistrationCache
}

type ClusterRegistrationClient interface {
	Create(*v1alpha1.ClusterRegistration) (*v1alpha1.ClusterRegistration, error)
	Update(*v1alpha1.ClusterRegistration) (*v1alpha1.ClusterRegistration, error)
	UpdateStatus(*v1alpha1.ClusterRegistration) (*v1alpha1.ClusterRegistration, error)
	Delete(namespace, name string, options *metav1.DeleteOptions) error
	Get(namespace, name string, options metav1.GetOptions) (*v1alpha1.ClusterRegistration, error)
	List(namespace string, opts metav1.ListOptions) (*v1alpha1.ClusterRegistrationList, error)
	Watch(namespace string, opts metav1.ListOptions) (watch.Interface, error)
	Patch(namespace, name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.ClusterRegistration, err error)
}

type ClusterRegistrationCache interface {
	Get(namespace, name string) (*v1alpha1.ClusterRegistration, error)
	List(namespace string, selector labels.Selector) ([]*v1alpha1.ClusterRegistration, error)

	AddIndexer(indexName string, indexer ClusterRegistrationIndexer)
	GetByIndex(indexName, key string) ([]*v1alpha1.ClusterRegistration, error)
}

type ClusterRegistrationIndexer func(obj *v1alpha1.ClusterRegistration) ([]string, error)

type clusterRegistrationController struct {
	controller    controller.SharedController
	client        *client.Client
	gvk           schema.GroupVersionKind
	groupResource schema.GroupResource
}

func NewClusterRegistrationController(gvk schema.GroupVersionKind, resource string, namespaced bool, controller controller.SharedControllerFactory) ClusterRegistrationController {
	c := controller.ForResourceKind(gvk.GroupVersion().WithResource(resource), gvk.Kind, namespaced)
	return &clusterRegistrationController{
		controller: c,
		client:     c.Client(),
		gvk:        gvk,
		groupResource: schema.GroupResource{
			Group:    gvk.Group,
			Resource: resource,
		},
	}
}

func FromClusterRegistrationHandlerToHandler(sync ClusterRegistrationHandler) generic.Handler {
	return func(key string, obj runtime.Object) (ret runtime.Object, err error) {
		var v *v1alpha1.ClusterRegistration
		if obj == nil {
			v, err = sync(key, nil)
		} else {
			v, err = sync(key, obj.(*v1alpha1.ClusterRegistration))
		}
		if v == nil {
			return nil, err
		}
		return v, err
	}
}

func (c *clusterRegistrationController) Updater() generic.Updater {
	return func(obj runtime.Object) (runtime.Object, error) {
		newObj, err := c.Update(obj.(*v1alpha1.ClusterRegistration))
		if newObj == nil {
			return nil, err
		}
		return newObj, err
	}
}

func UpdateClusterRegistrationDeepCopyOnChange(client ClusterRegistrationClient, obj *v1alpha1.ClusterRegistration, handler func(obj *v1alpha1.ClusterRegistration) (*v1alpha1.ClusterRegistration, error)) (*v1alpha1.ClusterRegistration, error) {
	if obj == nil {
		return obj, nil
	}

	copyObj := obj.DeepCopy()
	newObj, err := handler(copyObj)
	if newObj != nil {
		copyObj = newObj
	}
	if obj.ResourceVersion == copyObj.ResourceVersion && !equality.Semantic.DeepEqual(obj, copyObj) {
		return client.Update(copyObj)
	}

	return copyObj, err
}

func (c *clusterRegistrationController) AddGenericHandler(ctx context.Context, name string, handler generic.Handler) {
	c.controller.RegisterHandler(ctx, name, controller.SharedControllerHandlerFunc(handler))
}

func (c *clusterRegistrationController) AddGenericRemoveHandler(ctx context.Context, name string, handler generic.Handler) {
	c.AddGenericHandler(ctx, name, generic.NewRemoveHandler(name, c.Updater(), handler))
}

func (c *clusterRegistrationController) OnChange(ctx context.Context, name string, sync ClusterRegistrationHandler) {
	c.AddGenericHandler(ctx, name, FromClusterRegistrationHandlerToHandler(sync))
}

func (c *clusterRegistrationController) OnRemove(ctx context.Context, name string, sync ClusterRegistrationHandler) {
	c.AddGenericHandler(ctx, name, generic.NewRemoveHandler(name, c.Updater(), FromClusterRegistrationHandlerToHandler(sync)))
}

func (c *clusterRegistrationController) Enqueue(namespace, name string) {
	c.controller.Enqueue(namespace, name)
}

func (c *clusterRegistrationController) EnqueueAfter(namespace, name string, duration time.Duration) {
	c.controller.EnqueueAfter(namespace, name, duration)
}

func (c *clusterRegistrationController) Informer() cache.SharedIndexInformer {
	return c.controller.Informer()
}

func (c *clusterRegistrationController) GroupVersionKind() schema.GroupVersionKind {
	return c.gvk
}

func (c *clusterRegistrationController) Cache() ClusterRegistrationCache {
	return &clusterRegistrationCache{
		indexer:  c.Informer().GetIndexer(),
		resource: c.groupResource,
	}
}

func (c *clusterRegistrationController) Create(obj *v1alpha1.ClusterRegistration) (*v1alpha1.ClusterRegistration, error) {
	result := &v1alpha1.ClusterRegistration{}
	return result, c.client.Create(context.TODO(), obj.Namespace, obj, result, metav1.CreateOptions{})
}

func (c *clusterRegistrationController) Update(obj *v1alpha1.ClusterRegistration) (*v1alpha1.ClusterRegistration, error) {
	result := &v1alpha1.ClusterRegistration{}
	return result, c.client.Update(context.TODO(), obj.Namespace, obj, result, metav1.UpdateOptions{})
}

func (c *clusterRegistrationController) UpdateStatus(obj *v1alpha1.ClusterRegistration) (*v1alpha1.ClusterRegistration, error) {
	result := &v1alpha1.ClusterRegistration{}
	return result, c.client.UpdateStatus(context.TODO(), obj.Namespace, obj, result, metav1.UpdateOptions{})
}

func (c *clusterRegistrationController) Delete(namespace, name string, options *metav1.DeleteOptions) error {
	if options == nil {
		options = &metav1.DeleteOptions{}
	}
	return c.client.Delete(context.TODO(), namespace, name, *options)
}

func (c *clusterRegistrationController) Get(namespace, name string, options metav1.GetOptions) (*v1alpha1.ClusterRegistration, error) {
	result := &v1alpha1.ClusterRegistration{}
	return result, c.client.Get(context.TODO(), namespace, name, result, options)
}

func (c *clusterRegistrationController) List(namespace string, opts metav1.ListOptions) (*v1alpha1.ClusterRegistrationList, error) {
	result := &v1alpha1.ClusterRegistrationList{}
	return result, c.client.List(context.TODO(), namespace, result, opts)
}

func (c *clusterRegistrationController) Watch(namespace string, opts metav1.ListOptions) (watch.Interface, error) {
	return c.client.Watch(context.TODO(), namespace, opts)
}

func (c *clusterRegistrationController) Patch(namespace, name string, pt types.PatchType, data []byte, subresources ...string) (*v1alpha1.ClusterRegistration, error) {
	result := &v1alpha1.ClusterRegistration{}
	return result, c.client.Patch(context.TODO(), namespace, name, pt, data, result, metav1.PatchOptions{}, subresources...)
}

type clusterRegistrationCache struct {
	indexer  cache.Indexer
	resource schema.GroupResource
}

func (c *clusterRegistrationCache) Get(namespace, name string) (*v1alpha1.ClusterRegistration, error) {
	obj, exists, err := c.indexer.GetByKey(namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(c.resource, name)
	}
	return obj.(*v1alpha1.ClusterRegistration), nil
}

func (c *clusterRegistrationCache) List(namespace string, selector labels.Selector) (ret []*v1alpha1.ClusterRegistration, err error) {

	err = cache.ListAllByNamespace(c.indexer, namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.ClusterRegistration))
	})

	return ret, err
}

func (c *clusterRegistrationCache) AddIndexer(indexName string, indexer ClusterRegistrationIndexer) {
	utilruntime.Must(c.indexer.AddIndexers(map[string]cache.IndexFunc{
		indexName: func(obj interface{}) (strings []string, e error) {
			return indexer(obj.(*v1alpha1.ClusterRegistration))
		},
	}))
}

func (c *clusterRegistrationCache) GetByIndex(indexName, key string) (result []*v1alpha1.ClusterRegistration, err error) {
	objs, err := c.indexer.ByIndex(indexName, key)
	if err != nil {
		return nil, err
	}
	result = make([]*v1alpha1.ClusterRegistration, 0, len(objs))
	for _, obj := range objs {
		result = append(result, obj.(*v1alpha1.ClusterRegistration))
	}
	return result, nil
}

type ClusterRegistrationStatusHandler func(obj *v1alpha1.ClusterRegistration, status v1alpha1.ClusterRegistrationStatus) (v1alpha1.ClusterRegistrationStatus, error)

type ClusterRegistrationGeneratingHandler func(obj *v1alpha1.ClusterRegistration, status v1alpha1.ClusterRegistrationStatus) ([]runtime.Object, v1alpha1.ClusterRegistrationStatus, error)

func RegisterClusterRegistrationStatusHandler(ctx context.Context, controller ClusterRegistrationController, condition condition.Cond, name string, handler ClusterRegistrationStatusHandler) {
	statusHandler := &clusterRegistrationStatusHandler{
		client:    controller,
		condition: condition,
		handler:   handler,
	}
	controller.AddGenericHandler(ctx, name, FromClusterRegistrationHandlerToHandler(statusHandler.sync))
}

func RegisterClusterRegistrationGeneratingHandler(ctx context.Context, controller ClusterRegistrationController, apply apply.Apply,
	condition condition.Cond, name string, handler ClusterRegistrationGeneratingHandler, opts *generic.GeneratingHandlerOptions) {
	statusHandler := &clusterRegistrationGeneratingHandler{
		ClusterRegistrationGeneratingHandler: handler,
		apply:                                apply,
		name:                                 name,
		gvk:                                  controller.GroupVersionKind(),
	}
	if opts != nil {
		statusHandler.opts = *opts
	}
	controller.OnChange(ctx, name, statusHandler.Remove)
	RegisterClusterRegistrationStatusHandler(ctx, controller, condition, name, statusHandler.Handle)
}

type clusterRegistrationStatusHandler struct {
	client    ClusterRegistrationClient
	condition condition.Cond
	handler   ClusterRegistrationStatusHandler
}

func (a *clusterRegistrationStatusHandler) sync(key string, obj *v1alpha1.ClusterRegistration) (*v1alpha1.ClusterRegistration, error) {
	if obj == nil {
		return obj, nil
	}

	origStatus := obj.Status.DeepCopy()
	obj = obj.DeepCopy()
	newStatus, err := a.handler(obj, obj.Status)
	if err != nil {
		// Revert to old status on error
		newStatus = *origStatus.DeepCopy()
	}

	if a.condition != "" {
		if errors.IsConflict(err) {
			a.condition.SetError(&newStatus, "", nil)
		} else {
			a.condition.SetError(&newStatus, "", err)
		}
	}
	if !equality.Semantic.DeepEqual(origStatus, &newStatus) {
		if a.condition != "" {
			// Since status has changed, update the lastUpdatedTime
			a.condition.LastUpdated(&newStatus, time.Now().UTC().Format(time.RFC3339))
		}

		var newErr error
		obj.Status = newStatus
		newObj, newErr := a.client.UpdateStatus(obj)
		if err == nil {
			err = newErr
		}
		if newErr == nil {
			obj = newObj
		}
	}
	return obj, err
}

type clusterRegistrationGeneratingHandler struct {
	ClusterRegistrationGeneratingHandler
	apply apply.Apply
	opts  generic.GeneratingHandlerOptions
	gvk   schema.GroupVersionKind
	name  string
	seen  sync.Map
}

func (a *clusterRegistrationGeneratingHandler) Remove(key string, obj *v1alpha1.ClusterRegistration) (*v1alpha1.ClusterRegistration, error) {
	if obj != nil {
		return obj, nil
	}

	obj = &v1alpha1.ClusterRegistration{}
	obj.Namespace, obj.Name = kv.RSplit(key, "/")
	obj.SetGroupVersionKind(a.gvk)

	if a.opts.UniqueApplyForResourceVersion {
		a.seen.Delete(key)
	}

	return nil, generic.ConfigureApplyForObject(a.apply, obj, &a.opts).
		WithOwner(obj).
		WithSetID(a.name).
		ApplyObjects()
}

func (a *clusterRegistrationGeneratingHandler) Handle(obj *v1alpha1.ClusterRegistration, status v1alpha1.ClusterRegistrationStatus) (v1alpha1.ClusterRegistrationStatus, error) {
	if !obj.DeletionTimestamp.IsZero() {
		return status, nil
	}

	objs, newStatus, err := a.ClusterRegistrationGeneratingHandler(obj, status)
	if err != nil || !a.isNewResourceVersion(obj) {
		return newStatus, err
	}

	err = generic.ConfigureApplyForObject(a.apply, obj, &a.opts).
		WithOwner(obj).
		WithSetID(a.name).
		ApplyObjects(objs...)
	if err == nil {
		a.seenResourceVersion(obj)
	}
	return newStatus, err
}

func (a *clusterRegistrationGeneratingHandler) isNewResourceVersion(obj *v1alpha1.ClusterRegistration) bool {
	if !a.opts.UniqueApplyForResourceVersion {
		return true
	}

	// Apply once per resource version
	key := obj.Namespace + "/" + obj.Name
	previous, ok := a.seen.Load(key)
	return !ok || previous != obj.ResourceVersion
}

func (a *clusterRegistrationGeneratingHandler) seenResourceVersion(obj *v1alpha1.ClusterRegistration) {
	if !a.opts.UniqueApplyForResourceVersion {
		return
	}

	key := obj.Namespace + "/" + obj.Name
	a.seen.Store(key, obj.ResourceVersion)
}
