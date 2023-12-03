package internal

import (
	"container/ring"

	"github.com/rexagod/mad/pkg/apis/mad/v1alpha1"
)

// flushRing flushes the ring into a slice.
func flushRing(ptr *ring.Ring) []v1alpha1.HealthcheckRecord {

	buffer := make([]v1alpha1.HealthcheckRecord, 0, ptr.Len())
	ptr.Do(func(x interface{}) {
		if x != nil {
			buffer = append(buffer, x.(v1alpha1.HealthcheckRecord))
		}
	})

	return buffer
}

// ringAppend appends a new record to the ring.
func ringAppend(ptr *ring.Ring, record v1alpha1.HealthcheckRecord) *ring.Ring {

	// Generally, a ptr should never be nil.
	if ptr == nil {
		return nil
	}

	// ptr will be at the last entry.
	if _atCapacity(ptr) {

		// Move event record one index up.
		// Put the new record in the last place, since this is the oldest recorded event we will drop.
		ptr = ptr.Move(ptr.Len() - 1)
		ptr.Value = record
		return ptr
	}

	// Put the new record in the next place.
	if ptr.Value.(v1alpha1.HealthcheckRecord).Timestamp == nil {
		ptr.Value = record
	} else {
		ptr = ptr.Next()
		ptr.Value = record
	}

	return ptr
}

// _atCapacity checks if the ring is at capacity.
// NOTE: This is a mutating function, it will reset the ring pointer to the most recent event.
func _atCapacity(ptr *ring.Ring) bool {

	// Ring is at capacity if the previous element is the last element (the next element is either the oldest (at capacity) or nil).
	_resetToLastEntry(ptr)
	return ptr.Next().Value.(v1alpha1.HealthcheckRecord).Timestamp != nil
}

// _resetToLastEntry resets the ring pointer to the most recent event.
func _resetToLastEntry(ptr *ring.Ring) {

	// Generally, a ptr should never be nil.
	if ptr == nil {
		return
	}

	// If the ptr is empty, or has only one element, there is nothing to do.
	current := ptr
	ahead := current.Next()
	if current == ahead {
		return
	}

	// Find the most recent event.
	for {
		currentTS := current.Value.(v1alpha1.HealthcheckRecord).Timestamp
		aheadTS := ahead.Value.(v1alpha1.HealthcheckRecord).Timestamp
		if currentTS == nil || aheadTS == nil {
			break
		}
		if !currentTS.After(aheadTS.Time) {
			current = ahead
			ahead = ahead.Next()
		} else {
			break
		}
	}
}

// newRecordRing creates a new ring of records.
func newRecordRing(size int) *ring.Ring {
	r := ring.New(size)
	for i := 0; i < size; i++ {
		r.Value = v1alpha1.HealthcheckRecord{}
		r = r.Next()
	}
	return r
}
