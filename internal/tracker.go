package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/rexagod/mad/pkg/apis/mad/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
)

// endpointTracker is a struct that keeps track of the number of handlers that depend on it.
type endpointTracker struct {

	// fn starts the querier on the parent endpoint.
	fn func(endpoint string) bool

	// stopChannel is the channel used to stop the querier on the parent endpoint.
	stopChannel chan struct{}

	// relayChannel is the channel used to relay the health status of the parent endpoint.
	relayChannel chan bool

	// associatedHandlerCount is the number of handlers currently depending on this endpointTracker.
	// An endpoint entry and its associated resources are released when this count reaches 0.
	associatedHandlerCount int
}

func newEndpointTracker() *endpointTracker {
	t := endpointTracker{}
	t.stopChannel = make(chan struct{})
	t.relayChannel = make(chan bool)
	return &t
}

// trackMAD starts the querier on the parent endpoint and tracks the state throughout the process.
func (t *endpointTracker) trackMAD(
	ctx context.Context,
	h *madEventHandler,
	resource *v1alpha1.MetricsAnomalyDetectorResource,
	key string,
	endpoint string,
) {
	go func(
		context.Context,
		*madEventHandler,
		*v1alpha1.MetricsAnomalyDetectorResource,
		string,
		string,
	) {
		logger := klog.LoggerWithValues(klog.FromContext(ctx), "name", resource.GetName(), "namespace", resource.GetNamespace(), "endpoint", endpoint, "component", "tracker")
		for {
			select {

			// Stop the querier.
			case <-t.stopChannel:
				return

			// Stop the querier.
			case <-ctx.Done():
				return

			// Query and update on the specified intervals.
			case <-time.After(time.Duration(resource.Spec.QueryInterval) * time.Second):

				// Get the resource before updating to avoid conflicts.
				var err error
				resource, err = h.clientset.MadV1alpha1().MetricsAnomalyDetectorResources(h.namespace).Get(ctx, resource.GetName(), metav1.GetOptions{})
				if err != nil {
					if errors.IsNotFound(err) {
						logger.V(4).Info("resource has been deleted")
						return
					}
					logger.Error(err, "failed to get resource")
					return
				}

				// Update the endpoint's health status.
				isHealthy := t.fn(endpoint)
				if isHealthy {
					if resource.Status.HealthcheckEndpointsHealthy == nil {
						resource.Status.HealthcheckEndpointsHealthy = make(map[string]bool)
					}
					resource.Status.HealthcheckEndpointsHealthy[endpoint] = true
				}
				resource.Status.LastHealthcheckQueryTime = metav1.Now()

				// Append the new record to the ring, and flush.
				h.rings[key] = ringAppend(h.rings[key], v1alpha1.HealthcheckRecord{
					Timestamp: ptr.To(metav1.Now()),
					Healthy:   ptr.To(isHealthy),
				})
				resource.Status.LastBuffer = flushRing(h.rings[key])

				// Update the status.
				_, err = h.clientset.MadV1alpha1().MetricsAnomalyDetectorResources(h.namespace).UpdateStatus(ctx, resource, metav1.UpdateOptions{})
				if err != nil {
					if errors.IsConflict(err) {
						logger.V(4).Info("resource was modified, retrying")
						continue
					}
					logger.Error(err, "failed to update status")
					return
				}

				logger.V(4).Info(fmt.Sprintf("updated status for %s", endpoint))
			}
		}
	}(
		ctx,
		h,
		resource,
		key,
		endpoint,
	)

	// Record the number of handlers that depend on this endpointTracker.
	t.associatedHandlerCount++
}
