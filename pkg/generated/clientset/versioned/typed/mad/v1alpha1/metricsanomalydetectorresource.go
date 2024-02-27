/*
Copyright 2023 The Kubernetes mad Authors.

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

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/rexagod/mad/pkg/apis/mad/v1alpha1"
	scheme "github.com/rexagod/mad/pkg/generated/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// MetricsAnomalyDetectorResourcesGetter has a method to return a MetricsAnomalyDetectorResourceInterface.
// A group's client should implement this interface.
type MetricsAnomalyDetectorResourcesGetter interface {
	MetricsAnomalyDetectorResources(namespace string) MetricsAnomalyDetectorResourceInterface
}

// MetricsAnomalyDetectorResourceInterface has methods to work with MetricsAnomalyDetectorResource resources.
type MetricsAnomalyDetectorResourceInterface interface {
	Create(ctx context.Context, metricsAnomalyDetectorResource *v1alpha1.MetricsAnomalyDetectorResource, opts v1.CreateOptions) (*v1alpha1.MetricsAnomalyDetectorResource, error)
	Update(ctx context.Context, metricsAnomalyDetectorResource *v1alpha1.MetricsAnomalyDetectorResource, opts v1.UpdateOptions) (*v1alpha1.MetricsAnomalyDetectorResource, error)
	UpdateStatus(ctx context.Context, metricsAnomalyDetectorResource *v1alpha1.MetricsAnomalyDetectorResource, opts v1.UpdateOptions) (*v1alpha1.MetricsAnomalyDetectorResource, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.MetricsAnomalyDetectorResource, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.MetricsAnomalyDetectorResourceList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.MetricsAnomalyDetectorResource, err error)
	MetricsAnomalyDetectorResourceExpansion
}

// metricsAnomalyDetectorResources implements MetricsAnomalyDetectorResourceInterface
type metricsAnomalyDetectorResources struct {
	client rest.Interface
	ns     string
}

// newMetricsAnomalyDetectorResources returns a MetricsAnomalyDetectorResources
func newMetricsAnomalyDetectorResources(c *MadV1alpha1Client, namespace string) *metricsAnomalyDetectorResources {
	return &metricsAnomalyDetectorResources{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the metricsAnomalyDetectorResource, and returns the corresponding metricsAnomalyDetectorResource object, and an error if there is any.
func (c *metricsAnomalyDetectorResources) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.MetricsAnomalyDetectorResource, err error) {
	result = &v1alpha1.MetricsAnomalyDetectorResource{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("metricsanomalydetectorresources").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of MetricsAnomalyDetectorResources that match those selectors.
func (c *metricsAnomalyDetectorResources) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.MetricsAnomalyDetectorResourceList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.MetricsAnomalyDetectorResourceList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("metricsanomalydetectorresources").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested metricsAnomalyDetectorResources.
func (c *metricsAnomalyDetectorResources) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("metricsanomalydetectorresources").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a metricsAnomalyDetectorResource and creates it.  Returns the server's representation of the metricsAnomalyDetectorResource, and an error, if there is any.
func (c *metricsAnomalyDetectorResources) Create(ctx context.Context, metricsAnomalyDetectorResource *v1alpha1.MetricsAnomalyDetectorResource, opts v1.CreateOptions) (result *v1alpha1.MetricsAnomalyDetectorResource, err error) {
	result = &v1alpha1.MetricsAnomalyDetectorResource{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("metricsanomalydetectorresources").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(metricsAnomalyDetectorResource).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a metricsAnomalyDetectorResource and updates it. Returns the server's representation of the metricsAnomalyDetectorResource, and an error, if there is any.
func (c *metricsAnomalyDetectorResources) Update(ctx context.Context, metricsAnomalyDetectorResource *v1alpha1.MetricsAnomalyDetectorResource, opts v1.UpdateOptions) (result *v1alpha1.MetricsAnomalyDetectorResource, err error) {
	result = &v1alpha1.MetricsAnomalyDetectorResource{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("metricsanomalydetectorresources").
		Name(metricsAnomalyDetectorResource.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(metricsAnomalyDetectorResource).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *metricsAnomalyDetectorResources) UpdateStatus(ctx context.Context, metricsAnomalyDetectorResource *v1alpha1.MetricsAnomalyDetectorResource, opts v1.UpdateOptions) (result *v1alpha1.MetricsAnomalyDetectorResource, err error) {
	result = &v1alpha1.MetricsAnomalyDetectorResource{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("metricsanomalydetectorresources").
		Name(metricsAnomalyDetectorResource.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(metricsAnomalyDetectorResource).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the metricsAnomalyDetectorResource and deletes it. Returns an error if one occurs.
func (c *metricsAnomalyDetectorResources) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("metricsanomalydetectorresources").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *metricsAnomalyDetectorResources) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("metricsanomalydetectorresources").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched metricsAnomalyDetectorResource.
func (c *metricsAnomalyDetectorResources) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.MetricsAnomalyDetectorResource, err error) {
	result = &v1alpha1.MetricsAnomalyDetectorResource{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("metricsanomalydetectorresources").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}