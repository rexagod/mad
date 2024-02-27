package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/time/rate"

	"github.com/rexagod/mad/pkg/apis/mad/v1alpha1"
	"github.com/rexagod/mad/pkg/generated/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
)

// Run starts the server and listens for incoming requests.
// The lifecycle of a request is as follows:
// * extract the time-intervals from the request,
// * gather the health buffer for the given time-intervals,
// * detect anomalies in the health buffer, and,
// * relay the response back to the client.
func Run(clientset *versioned.Clientset, logger klog.Logger, ctx context.Context) {

	// Create a rate limiter.
	var limiter = rate.NewLimiter(1, 5)

	// Define the server's mux.
	mux := http.NewServeMux()
	mux.HandleFunc("/compute_health", func(w http.ResponseWriter, r *http.Request) {
		logger.Info(fmt.Sprintf("Received request from %s", r.RemoteAddr))

		// Rate limit the requests.
		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			logger.Info(fmt.Sprintf("Rate limit exceeded for %s", r.RemoteAddr))
			return
		}

		// Parse the form.
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Error parsing form", http.StatusBadRequest)
			return
		}

		// Extract the time-intervals from the request.
		key := r.Form.Get("key")  // Should be of the form: "namespace/name".
		tsA := r.Form.Get("ts_a") // Should be of the form: "2006-01-02T15:04:05Z07:00" (RFC3339).
		tsB := r.Form.Get("ts_b") // Should be of the form: "2006-01-02T15:04:05Z07:00" (RFC3339).

		// Validate the input.
		if key == "" || tsA == "" || tsB == "" {
			http.Error(w, "Missing parameters", http.StatusBadRequest)
			return
		}

		// Extract the namespace and name from the key.
		keyParts := strings.Split(key, "/")
		if len(keyParts) != 2 {
			http.Error(w, "Invalid key", http.StatusBadRequest)
			return
		}
		namespace := keyParts[0]
		name := keyParts[1]

		// Extract the time-intervals from the request.
		tsATyped, err := time.Parse(time.RFC3339, tsA)
		if err != nil {
			http.Error(w, "Invalid timestamp A", http.StatusBadRequest)
			return
		}
		tsBTyped, err := time.Parse(time.RFC3339, tsB)
		if err != nil {
			http.Error(w, "Invalid timestamp B", http.StatusBadRequest)
			return
		}
		tsAMetaV1 := metav1.NewTime(tsATyped)
		tsBMetaV1 := metav1.NewTime(tsBTyped)

		// Get the resource.
		var resource *v1alpha1.MetricsAnomalyDetectorResource
		err = retry.OnError(wait.Backoff{
			Duration: 100 * time.Millisecond,
			Factor:   3,
			Jitter:   1,
			Steps:    5,
		}, func(err error) bool {
			return true
		}, func() error {
			resource, err = clientset.MadV1alpha1().MetricsAnomalyDetectorResources(namespace).Get(ctx, name, metav1.GetOptions{})
			return err
		})
		if err != nil {
			http.Error(w, "Error getting resource", http.StatusInternalServerError)
			return
		}

		// Gather the health buffer for the given time-intervals.
		healthBuffer := make([]v1alpha1.HealthcheckRecord, 0)
		for _, healthRecord := range resource.Status.LastBuffer {
			if healthRecord.Timestamp.After(tsAMetaV1.Time) &&
				!healthRecord.Timestamp.After(tsBMetaV1.Time) {
				healthBuffer = append(healthBuffer, healthRecord)
			}
		}

		// Detect anomalies in the health buffer.
		evictedRecords, healthScore := evaluateHealth(healthBuffer)

		// Relay the response back to the client.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(`{"health_score": ` + fmt.Sprintf("%f", healthScore) + `, "unhealthy_records": ` + fmt.Sprintf("%d", len(evictedRecords)) + `, "healthy_records": ` + fmt.Sprintf("%d", len(healthBuffer)-len(evictedRecords)) + `}`))
		if err != nil {
			http.Error(w, "Error writing response", http.StatusInternalServerError)
			return
		}
	})

	// Define the server.
	s := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// Start listening.
	go func() {
		if err := s.ListenAndServe(); err != nil {
			logger.Error(err, "Error starting server")
		}
	}()

	// Shutdown the server gracefully.
	<-ctx.Done()
	if err := s.Shutdown(ctx); err != nil {
		logger.Error(err, "Error shutting down server")
	}
}
