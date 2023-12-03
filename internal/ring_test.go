package internal

import (
	"testing"

	"github.com/rexagod/mad/pkg/apis/mad/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestRing(t *testing.T) {
	r := newRecordRing(3)

	// The ring should not be at capacity.
	if _atCapacity(r) {
		t.Errorf("Expected the ring to not be at capacity")
	}

	// Append a record to the ring.
	r = ringAppend(r, v1alpha1.HealthcheckRecord{Healthy: ptr.To(true), Timestamp: ptr.To(metav1.Now())})

	// The last record should be true.
	if r.Value.(v1alpha1.HealthcheckRecord).Healthy == nil ||
		!*r.Value.(v1alpha1.HealthcheckRecord).Healthy {
		t.Errorf("Expected last record to be true, got false")
	}

	// The ring should not be at capacity.
	if _atCapacity(r) {
		t.Errorf("Expected the ring to not be at capacity")
	}

	// Append a record to the ring.
	r = ringAppend(r, v1alpha1.HealthcheckRecord{Healthy: ptr.To(false), Timestamp: ptr.To(metav1.Now())})

	// The last record should be true.
	if r.Value.(v1alpha1.HealthcheckRecord).Healthy == nil ||
		*r.Value.(v1alpha1.HealthcheckRecord).Healthy ||
		!*r.Prev().Value.(v1alpha1.HealthcheckRecord).Healthy {
		t.Errorf("Expected last record to be false, got true")
	}

	// The ring should not be at capacity.
	if _atCapacity(r) {
		t.Errorf("Expected the ring to be at capacity")
	}

	r = ringAppend(r, v1alpha1.HealthcheckRecord{Healthy: ptr.To(true), Timestamp: ptr.To(metav1.Now())})

	// The last record should be true.
	if r.Value.(v1alpha1.HealthcheckRecord).Healthy == nil ||
		!*r.Value.(v1alpha1.HealthcheckRecord).Healthy ||
		*r.Prev().Value.(v1alpha1.HealthcheckRecord).Healthy ||
		!*r.Prev().Prev().Value.(v1alpha1.HealthcheckRecord).Healthy {
		t.Errorf("Expected last record to be true, got false")
	}

	// The ring should be at capacity.
	if !_atCapacity(r) {
		t.Errorf("Expected the ring to be at capacity")
	}
}
