package internal

import (
	"container/ring"
	"context"
	"fmt"

	"github.com/rexagod/mad/pkg/apis/mad/v1alpha1"
	clientset "github.com/rexagod/mad/pkg/generated/clientset/versioned"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
)

// madEventHandler implements the EventHandler interface.
var _ eventHandler = &madEventHandler{}

// eventHandler knows how to handle informer events.
type eventHandler interface {

	// HandleEvent handles events received from the informer.
	HandleEvent(ctx context.Context, o metav1.Object, event int) error
}

// madEventHandler implements the EventHandler interface.
type madEventHandler struct {

	// namespace is the namespace of the mad resource.
	namespace string

	// clientset is the clientset used to update the status of the mad resource.
	clientset clientset.Interface

	// endpointsGlobalStore is the global store of endpoints that are currently being tracked.
	endpointsGlobalStore *map[string]*endpointTracker

	// querier is the querier used to query the healthcheck endpoints.
	querier *Querier

	// rings is the map of object keys to their respective circular buffers that holds the last `bufferSize` events (by querying the endpoints of all associated components).
	rings map[string]*ring.Ring
}

// HandleEvent handles events received from the informer.
func (h *madEventHandler) HandleEvent(ctx context.Context, o metav1.Object, event int) error {
	logger := klog.LoggerWithValues(klog.FromContext(ctx), "name", o.GetName(), "namespace", o.GetNamespace(), "event", event, "component", "handler")
	logger.V(4).Info("Handling event")

	resource, ok := o.(*v1alpha1.MetricsAnomalyDetectorResource)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("failed to cast object to %s", resource.GetObjectKind()))
		return nil // Do not requeue.
	}
	key, err := cache.MetaNamespaceKeyFunc(resource)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("failed to get key for %s %s/%s: %w", resource.GetObjectKind(), resource.GetNamespace(), resource.GetName(), err))
		return nil // Do not requeue.
	}

	// Handle the event.
	switch event {

	// Add and update events are handled the same way.
	case AddEvent, UpdateEvent:

		// Try to retrieve the ring from the map, even if this is an add event, as this signals a situation where the delete event was not registered.
		// This logic also considers the case where the controller was restarted, but one (or more) CRs persisted,
		resourceRingPtr, ok := h.rings[key]

		// We do not have an in-memory buffer for this resource.
		if !ok {
			bufferSize := resource.Spec.BufferSize
			h.rings[key] = newRecordRing(bufferSize)

			// TODO: Verify if this backup logic works in case of a stray MAD CR that pre-dates the controller.
			observedBuffer := resource.Status.LastBuffer
			if len(observedBuffer) > bufferSize {

				// Keep the last bufferSize events.
				observedBuffer = observedBuffer[len(observedBuffer)-bufferSize:]
			}
			for i := 0; i < len(observedBuffer); i++ {
				h.rings[key] = ringAppend(h.rings[key], observedBuffer[i])
			}
			resourceRingPtr = h.rings[key]
		} else {

			// Check if the bufferSize was updated between restarts.
			if resourceRingPtr.Len() != resource.Spec.BufferSize {
				oldRingPtr := resourceRingPtr
				newRingPtr := newRecordRing(resource.Spec.BufferSize)

				// Copy the old ring into the new one.
				newRingPtrCopy := newRingPtr
				oldRingPtr.Do(func(p interface{}) {
					if p != nil {
						newRingPtr.Value = p
						newRingPtr = newRingPtr.Next()
					}
				})

				// Reset the ring pointer.
				resourceRingPtr = newRingPtrCopy
			}
		}

		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			resource, err = h.clientset.MadV1alpha1().MetricsAnomalyDetectorResources(resource.Namespace).Get(ctx, resource.Name, metav1.GetOptions{})
			if err != nil {
				if errors.IsNotFound(err) {

					// The object has been deleted, stop the update process, don't requeue.
					return nil
				}
				return fmt.Errorf("failed to get %s/%s (%s): %w", resource.GetNamespace(), resource.GetName(), resource.GetObjectKind().GroupVersionKind(), err)
			}

			// Modify the resource status.
			resource.Status.CurrentBufferSize = resourceRingPtr.Len()
			resource.Status.LastBufferModificationTime = metav1.Now()
			resource.Status.LastBuffer = make([]v1alpha1.HealthcheckRecord, resourceRingPtr.Len())
			resourceRingPtr.Do(func(p interface{}) {
				if p == nil {
					return
				}
				pTyped, ok := p.(v1alpha1.HealthcheckRecord)
				if !ok {
					logger.Error(fmt.Errorf("failed to cast object (%T) to %v", p, pTyped), "failed to cast object")
					return
				}
				if pTyped.Healthy != nil && pTyped.Timestamp != nil {
					resource.Status.LastBuffer = append(resource.Status.LastBuffer, pTyped)
				}
			})

			// Update the status.
			_, err := h.clientset.MadV1alpha1().MetricsAnomalyDetectorResources(h.namespace).UpdateStatus(ctx, resource, metav1.UpdateOptions{})
			if err != nil {
				if errors.IsConflict(err) {
					logger.V(4).Info("resource was modified, retrying")
					return err
				}
				logger.Error(err, "failed to update status")

				// Don't requeue.
				return nil
			}
			return nil
		})

		// Start tracking the endpoints.
		for _, endpoint := range resource.Spec.HealthcheckEndpoints {

			// Check if the endpoint is already being tracked in the global registry. If not, add and start tracking it.
			store := h.endpointsGlobalStore
			if _, ok := (*h.endpointsGlobalStore)[endpoint]; !ok {
				(*store)[endpoint] = newEndpointTracker()

				// We assume that every endpoint maps to a single behavior, but this may be scaled to support multiple behaviors per endpoint, if need be.
				(*store)[endpoint].fn = h.querier.DoMADQuery
			}
			(*store)[endpoint].trackMAD(
				ctx,      // Cancel on context cancellation.
				h,        // Update rings and resource status.
				resource, // Get resource spec details without querying everytime.
				key,      // Narrow down the ring to the current resource.
				endpoint, // Endpoint to query.
			)
		}

	// Release any in-memory resources associated with the current mad resource.
	case DeleteEvent:

		// Release associated object ring.
		delete(h.rings, key)

		// Release associated endpoint trackers.
		for _, endpoint := range resource.Spec.HealthcheckEndpoints {
			store := h.endpointsGlobalStore
			(*store)[endpoint].associatedHandlerCount -= 1
			if (*store)[endpoint].associatedHandlerCount == 0 {
				(*store)[endpoint].stopChannel <- struct{}{}
				delete(*store, endpoint)
			}
		}
	default:

		// This should never happen.
		logger.Error(fmt.Errorf("unknown event type: %d", event), "unknown event type")
	}

	return nil
}
