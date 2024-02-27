package server

import "github.com/rexagod/mad/pkg/apis/mad/v1alpha1"

// evaluateHealth computes the overall health status.
// NOTE: The algorithm below is extremely naive and is only meant to serve as a placeholder.
func evaluateHealth(buffer []v1alpha1.HealthcheckRecord) ([]v1alpha1.HealthcheckRecord, float64) {

	// Filter out all unhealthy records.
	unhealthyRecords := make([]v1alpha1.HealthcheckRecord, 0)
	for _, record := range buffer {
		if !*record.Healthy {
			unhealthyRecords = append(unhealthyRecords, record)
		}
	}

	// Calculate the health score.
	if len(buffer) == 0 {
		return buffer, 0
	}
	healthScore := float64(len(buffer)-len(unhealthyRecords)) / float64(len(buffer))

	return unhealthyRecords, healthScore
}
