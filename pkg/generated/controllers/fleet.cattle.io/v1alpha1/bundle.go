/*
Copyright (c) 2020 - 2024 SUSE LLC

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

type BundleHandler func(string, *v1alpha1.Bundle) (*v1alpha1.Bundle, error)

type BundleController interface {
	generic.ControllerMeta
	BundleClient

	OnChange(ctx context.Context, name string, sync BundleHandler)
	OnRemove(ctx context.Context, name string, sync BundleHandler)
	Enqueue(namespace, name string)
	EnqueueAfter(namespace, name string, duration time.Duration)

	Cache() BundleCache
}

type BundleClient interface {
	Create(*v1alpha1.Bundle) (*v1alpha1.Bundle, error)
	Update(*v1alpha1.Bundle) (*v1alpha1.Bundle, error)
	UpdateStatus(*v1alpha1.Bundle) (*v1alpha1.Bundle, error)
	Delete(namespace, name string, options *metav1.DeleteOptions) error
	Get(namespace, name string, options metav1.GetOptions) (*v1alpha1.Bundle, error)
	List(namespace string, opts metav1.ListOptions) (*v1alpha1.BundleList, error)
	Watch(namespace string, opts metav1.ListOptions) (watch.Interface, error)
	Patch(namespace, name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Bundle, err error)
}

type BundleCache interface {
	Get(namespace, name string) (*v1alpha1.Bundle, error)
	List(namespace string, selector labels.Selector) ([]*v1alpha1.Bundle, error)

	AddIndexer(indexName string, indexer BundleIndexer)
	GetByIndex(indexName, key string) ([]*v1alpha1.Bundle, error)
}

type BundleIndexer func(obj *v1alpha1.Bundle) ([]string, error)

type bundleController struct {
	controller    controller.SharedController
	client        *client.Client
	gvk           schema.GroupVersionKind
	groupResource schema.GroupResource
}

func NewBundleController(gvk schema.GroupVersionKind, resource string, namespaced bool, controller controller.SharedControllerFactory) BundleController {
	c := controller.ForResourceKind(gvk.GroupVersion().WithResource(resource), gvk.Kind, namespaced)
	return &bundleController{
		controller: c,
		client:     c.Client(),
		gvk:        gvk,
		groupResource: schema.GroupResource{
			Group:    gvk.Group,
			Resource: resource,
		},
	}
}

func FromBundleHandlerToHandler(sync BundleHandler) generic.Handler {
	return func(key string, obj runtime.Object) (ret runtime.Object, err error) {
		var v *v1alpha1.Bundle
		if obj == nil {
			v, err = sync(key, nil)
		} else {
			v, err = sync(key, obj.(*v1alpha1.Bundle))
		}
		if v == nil {
			return nil, err
		}
		return v, err
	}
}

func (c *bundleController) Updater() generic.Updater {
	return func(obj runtime.Object) (runtime.Object, error) {
		newObj, err := c.Update(obj.(*v1alpha1.Bundle))
		if newObj == nil {
			return nil, err
		}
		return newObj, err
	}
}

func UpdateBundleDeepCopyOnChange(client BundleClient, obj *v1alpha1.Bundle, handler func(obj *v1alpha1.Bundle) (*v1alpha1.Bundle, error)) (*v1alpha1.Bundle, error) {
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

func (c *bundleController) AddGenericHandler(ctx context.Context, name string, handler generic.Handler) {
	c.controller.RegisterHandler(ctx, name, controller.SharedControllerHandlerFunc(handler))
}

func (c *bundleController) AddGenericRemoveHandler(ctx context.Context, name string, handler generic.Handler) {
	c.AddGenericHandler(ctx, name, generic.NewRemoveHandler(name, c.Updater(), handler))
}

func (c *bundleController) OnChange(ctx context.Context, name string, sync BundleHandler) {
	c.AddGenericHandler(ctx, name, FromBundleHandlerToHandler(sync))
}

func (c *bundleController) OnRemove(ctx context.Context, name string, sync BundleHandler) {
	c.AddGenericHandler(ctx, name, generic.NewRemoveHandler(name, c.Updater(), FromBundleHandlerToHandler(sync)))
}

func (c *bundleController) Enqueue(namespace, name string) {
	c.controller.Enqueue(namespace, name)
}

func (c *bundleController) EnqueueAfter(namespace, name string, duration time.Duration) {
	c.controller.EnqueueAfter(namespace, name, duration)
}

func (c *bundleController) Informer() cache.SharedIndexInformer {
	return c.controller.Informer()
}

func (c *bundleController) GroupVersionKind() schema.GroupVersionKind {
	return c.gvk
}

func (c *bundleController) Cache() BundleCache {
	return &bundleCache{
		indexer:  c.Informer().GetIndexer(),
		resource: c.groupResource,
	}
}

func (c *bundleController) Create(obj *v1alpha1.Bundle) (*v1alpha1.Bundle, error) {
	result := &v1alpha1.Bundle{}
	return result, c.client.Create(context.TODO(), obj.Namespace, obj, result, metav1.CreateOptions{})
}

func (c *bundleController) Update(obj *v1alpha1.Bundle) (*v1alpha1.Bundle, error) {
	result := &v1alpha1.Bundle{}
	return result, c.client.Update(context.TODO(), obj.Namespace, obj, result, metav1.UpdateOptions{})
}

func (c *bundleController) UpdateStatus(obj *v1alpha1.Bundle) (*v1alpha1.Bundle, error) {
	result := &v1alpha1.Bundle{}
	return result, c.client.UpdateStatus(context.TODO(), obj.Namespace, obj, result, metav1.UpdateOptions{})
}

func (c *bundleController) Delete(namespace, name string, options *metav1.DeleteOptions) error {
	if options == nil {
		options = &metav1.DeleteOptions{}
	}
	return c.client.Delete(context.TODO(), namespace, name, *options)
}

func (c *bundleController) Get(namespace, name string, options metav1.GetOptions) (*v1alpha1.Bundle, error) {
	result := &v1alpha1.Bundle{}
	return result, c.client.Get(context.TODO(), namespace, name, result, options)
}

func (c *bundleController) List(namespace string, opts metav1.ListOptions) (*v1alpha1.BundleList, error) {
	result := &v1alpha1.BundleList{}
	return result, c.client.List(context.TODO(), namespace, result, opts)
}

func (c *bundleController) Watch(namespace string, opts metav1.ListOptions) (watch.Interface, error) {
	return c.client.Watch(context.TODO(), namespace, opts)
}

func (c *bundleController) Patch(namespace, name string, pt types.PatchType, data []byte, subresources ...string) (*v1alpha1.Bundle, error) {
	result := &v1alpha1.Bundle{}
	return result, c.client.Patch(context.TODO(), namespace, name, pt, data, result, metav1.PatchOptions{}, subresources...)
}

type bundleCache struct {
	indexer  cache.Indexer
	resource schema.GroupResource
}

func (c *bundleCache) Get(namespace, name string) (*v1alpha1.Bundle, error) {
	obj, exists, err := c.indexer.GetByKey(namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(c.resource, name)
	}
	return obj.(*v1alpha1.Bundle), nil
}

func (c *bundleCache) List(namespace string, selector labels.Selector) (ret []*v1alpha1.Bundle, err error) {

	err = cache.ListAllByNamespace(c.indexer, namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Bundle))
	})

	return ret, err
}

func (c *bundleCache) AddIndexer(indexName string, indexer BundleIndexer) {
	utilruntime.Must(c.indexer.AddIndexers(map[string]cache.IndexFunc{
		indexName: func(obj interface{}) (strings []string, e error) {
			return indexer(obj.(*v1alpha1.Bundle))
		},
	}))
}

func (c *bundleCache) GetByIndex(indexName, key string) (result []*v1alpha1.Bundle, err error) {
	objs, err := c.indexer.ByIndex(indexName, key)
	if err != nil {
		return nil, err
	}
	result = make([]*v1alpha1.Bundle, 0, len(objs))
	for _, obj := range objs {
		result = append(result, obj.(*v1alpha1.Bundle))
	}
	return result, nil
}

// BundleStatusHandler is executed for every added or modified Bundle. Should return the new status to be updated
type BundleStatusHandler func(obj *v1alpha1.Bundle, status v1alpha1.BundleStatus) (v1alpha1.BundleStatus, error)

// BundleGeneratingHandler is the top-level handler that is executed for every Bundle event. It extends BundleStatusHandler by a returning a slice of child objects to be passed to apply.Apply
type BundleGeneratingHandler func(obj *v1alpha1.Bundle, status v1alpha1.BundleStatus) ([]runtime.Object, v1alpha1.BundleStatus, error)

// RegisterBundleStatusHandler configures a BundleController to execute a BundleStatusHandler for every events observed.
// If a non-empty condition is provided, it will be updated in the status conditions for every handler execution
func RegisterBundleStatusHandler(ctx context.Context, controller BundleController, condition condition.Cond, name string, handler BundleStatusHandler) {
	statusHandler := &bundleStatusHandler{
		client:    controller,
		condition: condition,
		handler:   handler,
	}
	controller.AddGenericHandler(ctx, name, FromBundleHandlerToHandler(statusHandler.sync))
}

// RegisterBundleGeneratingHandler configures a BundleController to execute a BundleGeneratingHandler for every events observed, passing the returned objects to the provided apply.Apply.
// If a non-empty condition is provided, it will be updated in the status conditions for every handler execution
func RegisterBundleGeneratingHandler(ctx context.Context, controller BundleController, apply apply.Apply,
	condition condition.Cond, name string, handler BundleGeneratingHandler, opts *generic.GeneratingHandlerOptions) {
	statusHandler := &bundleGeneratingHandler{
		BundleGeneratingHandler: handler,
		apply:                   apply,
		name:                    name,
		gvk:                     controller.GroupVersionKind(),
	}
	if opts != nil {
		statusHandler.opts = *opts
	}
	controller.OnChange(ctx, name, statusHandler.Remove)
	RegisterBundleStatusHandler(ctx, controller, condition, name, statusHandler.Handle)
}

type bundleStatusHandler struct {
	client    BundleClient
	condition condition.Cond
	handler   BundleStatusHandler
}

// sync is executed on every resource addition or modification. Executes the configured handlers and sends the updated status to the Kubernetes API
func (a *bundleStatusHandler) sync(key string, obj *v1alpha1.Bundle) (*v1alpha1.Bundle, error) {
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

type bundleGeneratingHandler struct {
	BundleGeneratingHandler
	apply apply.Apply
	opts  generic.GeneratingHandlerOptions
	gvk   schema.GroupVersionKind
	name  string
	seen  sync.Map
}

// Remove handles the observed deletion of a resource, cascade deleting every associated resource previously applied
func (a *bundleGeneratingHandler) Remove(key string, obj *v1alpha1.Bundle) (*v1alpha1.Bundle, error) {
	if obj != nil {
		return obj, nil
	}

	obj = &v1alpha1.Bundle{}
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

// Handle executes the configured BundleGeneratingHandler and pass the resulting objects to apply.Apply, finally returning the new status of the resource
func (a *bundleGeneratingHandler) Handle(obj *v1alpha1.Bundle, status v1alpha1.BundleStatus) (v1alpha1.BundleStatus, error) {
	if !obj.DeletionTimestamp.IsZero() {
		return status, nil
	}

	objs, newStatus, err := a.BundleGeneratingHandler(obj, status)
	if err != nil {
		return newStatus, err
	}
	if !a.isNewResourceVersion(obj) {
		return newStatus, nil
	}

	err = generic.ConfigureApplyForObject(a.apply, obj, &a.opts).
		WithOwner(obj).
		WithSetID(a.name).
		ApplyObjects(objs...)
	if err != nil {
		return newStatus, err
	}
	a.storeResourceVersion(obj)
	return newStatus, nil
}

// isNewResourceVersion detects if a specific resource version was already successfully processed.
// Only used if UniqueApplyForResourceVersion is set in generic.GeneratingHandlerOptions
func (a *bundleGeneratingHandler) isNewResourceVersion(obj *v1alpha1.Bundle) bool {
	if !a.opts.UniqueApplyForResourceVersion {
		return true
	}

	// Apply once per resource version
	key := obj.Namespace + "/" + obj.Name
	previous, ok := a.seen.Load(key)
	return !ok || previous != obj.ResourceVersion
}

// storeResourceVersion keeps track of the latest resource version of an object for which Apply was executed
// Only used if UniqueApplyForResourceVersion is set in generic.GeneratingHandlerOptions
func (a *bundleGeneratingHandler) storeResourceVersion(obj *v1alpha1.Bundle) {
	if !a.opts.UniqueApplyForResourceVersion {
		return
	}

	key := obj.Namespace + "/" + obj.Name
	a.seen.Store(key, obj.ResourceVersion)
}
