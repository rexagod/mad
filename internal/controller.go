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

package internal

import (
	"container/ring"
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"golang.org/x/time/rate"

	"github.com/davecgh/go-spew/spew"
	"github.com/rexagod/mad/pkg/apis/mad/v1alpha1"
	clientset "github.com/rexagod/mad/pkg/generated/clientset/versioned"
	madscheme "github.com/rexagod/mad/pkg/generated/clientset/versioned/scheme"
	informers "github.com/rexagod/mad/pkg/generated/informers/externalversions"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

// controllerName is the event source for the recorder.
const controllerName = "mad-controller"

const (
	AddEvent = iota
	UpdateEvent
	DeleteEvent
)

// Controller is the controller implementation for MetricsAnomalyDetectorResource resources.
type Controller struct {

	// namespace is the namespace in which the controller will operate.
	namespace string

	// kubeclientset is a standard kubernetes clientset, required for native operations.
	kubeclientset kubernetes.Interface

	// madClientset is a clientset for our own API group.
	madClientset clientset.Interface

	// madInformerFactory is a shared informer factory for mad resources.
	madInformerFactory informers.SharedInformerFactory

	// Querier is the querier used to query the endpoints for healthchecks.
	madQuerier *Querier

	// healthcheckEndpointStore is the store of healthcheck endpoints that are currently in-memory.
	// All event handlers must use this store to ensure consistency.
	healthcheckEndpointStore map[string]*endpointTracker

	// workqueue is a rate limited work queue. This is used to queue work to be processed instead of performing it as
	// soon as a change happens. This means we can ensure we only process a fixed amount of resources at a time, and
	// makes it easy to ensure we are never processing the same item simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface

	// recorder is an event recorder for recording Event resources to the Kubernetes API.
	recorder record.EventRecorder
}

// NewController returns a new sample controller.
func NewController(ctx context.Context, kubeClientset kubernetes.Interface, madClientset clientset.Interface) *Controller {

	// Add native resources to the default Kubernetes Scheme so Events can be logged for them.
	utilruntime.Must(madscheme.AddToScheme(scheme.Scheme))

	// Initialize the controller.
	logger := klog.FromContext(ctx)
	logger.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerName})
	ratelimiter := workqueue.NewMaxOfRateLimiter(
		workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 1000*time.Second),
		&workqueue.BucketRateLimiter{Limiter:
		// Burst is the maximum number of tokens
		// that can be consumed in a single call
		// to Allow, Reserve, or Wait, so higher
		// Burst values allow more events to
		// happen at once. A zero Burst allows no
		// events, unless limit == Inf.
		rate.NewLimiter(rate.Limit(50), 300)},
	)

	// Get the namespace from the environment. Pods are required to have a namespace, so this is a safe assumption.
	namespace, found := os.LookupEnv("NAMESPACE")
	if !found {
		panic("NAMESPACE environment variable not set")
	}
	controller := &Controller{
		namespace:                namespace,
		kubeclientset:            kubeClientset,
		madClientset:             madClientset,
		madInformerFactory:       informers.NewSharedInformerFactory(madClientset, time.Second*30),
		madQuerier:               NewQuerier(),
		healthcheckEndpointStore: make(map[string]*endpointTracker),
		workqueue:                workqueue.NewRateLimitingQueue(ratelimiter),
		recorder:                 recorder,
	}

	// Set up event handlers for MetricsAnomalyDetectorResource resources.
	logger.Info("Setting up event handlers")
	_, err := controller.madInformerFactory.Mad().V1alpha1().MetricsAnomalyDetectorResources().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			controller.enqueueMetricsAnomalyDetectorResource(obj, AddEvent)
		},
		UpdateFunc: func(old, new interface{}) {
			if old.(*v1alpha1.MetricsAnomalyDetectorResource).ResourceVersion == new.(*v1alpha1.MetricsAnomalyDetectorResource).ResourceVersion {
				return
			}
			controller.enqueueMetricsAnomalyDetectorResource(new, UpdateEvent)
		},
		DeleteFunc: func(obj interface{}) {
			controller.enqueueMetricsAnomalyDetectorResource(obj, DeleteEvent)
		},
	})
	if err != nil {
		klog.Fatal(err)
	}

	return controller
}

// enqueueMetricsAnomalyDetectorResource takes a MetricsAnomalyDetectorResource resource and converts it into a namespace/name key.
func (c *Controller) enqueueMetricsAnomalyDetectorResource(obj interface{}, event int) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add([2]string{key, strconv.Itoa(event)})
}

// Run starts the controller.
func (c *Controller) Run(ctx context.Context, workers int) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	logger := klog.FromContext(ctx)
	logger.Info("Starting MetricsAnomalyDetectorResource controller")
	logger.Info("Waiting for informer caches to sync")

	// Start the informer factories to begin populating the informer caches.
	c.madInformerFactory.Start(ctx.Done())
	if ok := cache.WaitForCacheSync(ctx.Done(), c.madInformerFactory.Mad().V1alpha1().MetricsAnomalyDetectorResources().Informer().HasSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	// Launch `workers` amount of goroutines to process the work queue.
	logger.Info("Starting workers", "count", workers)
	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(ctx, func(ctx context.Context) {

			// Run every second. Nothing will be done if there are no enqueued items. Work-queues are thread-safe.
			for c.processNextWorkItem(ctx) {
			}
		}, time.Second)
	}

	logger.Info("Started workers")
	<-ctx.Done()
	logger.Info("Shutting down workers")

	return nil
}

// processNextWorkItem retrieves each queued item and takes the necessary handler action, if the item has a valid object key.
// Whether the item itself is a valid object or not (tombstone), is checked further down the line.
func (c *Controller) processNextWorkItem(ctx context.Context) bool {
	logger := klog.FromContext(ctx)

	// Retrieve the next item from the queue.
	objWithEventInterface, shutdown := c.workqueue.Get()
	objWithEvent := objWithEventInterface.([2]string)
	if shutdown {
		return false
	}

	// Wrap this block in a func, so we can defer c.workqueue.Done. Forget the item if its invalid or processed.
	err := func(objWithEvent [2]string) error {
		defer c.workqueue.Done(objWithEvent)
		key := objWithEvent[0]
		event, err := strconv.Atoi(objWithEvent[1])
		if err != nil {
			c.workqueue.Forget(objWithEvent)
			utilruntime.HandleError(fmt.Errorf("invalid event type: %w", err))
			return nil
		}
		if err := c.syncHandler(ctx, key, event); err != nil {

			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(objWithEvent)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}

		// Finally, if no error occurs we Forget this item, so it does not
		// get queued again until another change happens. Done has no effect
		// after Forget, so we must call it before.
		c.workqueue.Forget(objWithEvent)
		logger.Info("Successfully synced", "resourceName", key)
		return nil
	}(objWithEvent)
	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// syncHandler resolves the object key, and sends it down for processing.
func (c *Controller) syncHandler(ctx context.Context, key string, event int) error {

	// Extract the namespace and name from the key.
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the MetricsAnomalyDetectorResource resource with this namespace and name.
	madResource, err := c.madInformerFactory.Mad().V1alpha1().MetricsAnomalyDetectorResources().Lister().MetricsAnomalyDetectorResources(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("metricsAnomalyDetectorResource '%s' in work queue no longer exists", key))
			return nil
		}

		return fmt.Errorf("error getting metricsAnomalyDetectorResource '%s/%s': %w", namespace, name, err)
	}

	return c.handleObject(ctx, madResource, event)
}

func (c *Controller) handleObject(ctx context.Context, obj interface{}, event int) error {
	logger := klog.FromContext(ctx)

	// Check if the object is nil, and if so, handle it.
	if obj == nil {
		utilruntime.HandleError(fmt.Errorf("recieved nil object for handling, skipping"))

		// No point in re-queueing.
		return nil
	}

	// Check if the object is a valid tombstone, and if so, recover and process it.
	var (
		object metav1.Object
		ok     bool
	)
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object, invalid type"))

			// No point in re-queueing.
			return nil
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))

			// No point in re-queueing.
			return nil
		}
		logger.V(4).Info("Recovered deleted object", "resourceName", object.GetName())
	}

	// Process the object based on its type.
	logger.V(4).Info("Processing object", "object", klog.KObj(object))
	switch o := object.(type) {
	case *v1alpha1.MetricsAnomalyDetectorResource:
		handler := &madEventHandler{
			namespace:            object.GetNamespace(),
			clientset:            c.madClientset,
			endpointsGlobalStore: &c.healthcheckEndpointStore,
			querier:              c.madQuerier,
			rings:                make(map[string]*ring.Ring),
		}
		return handler.HandleEvent(ctx, o, event)
	default:
		utilruntime.HandleError(fmt.Errorf("unknown object type: %T, full schema below:\n%s", o, spew.Sdump(obj)))
	}

	return nil
}
