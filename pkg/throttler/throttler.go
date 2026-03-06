// Package throttler provides per-operation-ID throttling within time windows.
package throttler

import (
	"sync"
	"time"
)

// Throttler limits operations per time window, grouped by operation ID.
type Throttler struct {
	mu            sync.Mutex
	windowMs      int64
	maxOperations int
	operations    map[string]int64
	counts        map[string]int
}

// New creates a Throttler with the given window and max operations per window.
func New(window time.Duration, maxOperations int) *Throttler {
	return &Throttler{
		windowMs:      window.Milliseconds(),
		maxOperations: maxOperations,
		operations:    make(map[string]int64),
		counts:        make(map[string]int),
	}
}

// TryThrottle checks if an operation is allowed. Returns true if allowed.
func (t *Throttler) TryThrottle(operationID string) bool {
	now := time.Now().UnixMilli()
	t.mu.Lock()
	defer t.mu.Unlock()

	lastOp, exists := t.operations[operationID]
	if !exists {
		t.operations[operationID] = now
		lastOp = now
	}

	if now-lastOp < t.windowMs {
		count := t.counts[operationID]
		if count+1 > t.maxOperations {
			return false
		}
		t.counts[operationID] = count + 1
	} else {
		t.operations[operationID] = now
		t.counts[operationID] = 1
	}

	return true
}

// Clear resets throttle state for an operation.
func (t *Throttler) Clear(operationID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.operations, operationID)
	delete(t.counts, operationID)
}
