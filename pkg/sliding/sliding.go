package sliding

import (
	"sync"
	"time"
)

// Window implements a sliding window counter algorithm.
// It divides time into sub-windows and counts requests per sub-window.
// This provides a more accurate rate limit than a simple fixed window,
// avoiding the burst problem at window boundaries.
type Window struct {
	mu         sync.Mutex
	subWindows map[int64]int // timestamp (sub-window start) -> count
	rate       int           // max requests per window
	window     time.Duration // total window duration
	subSize    time.Duration // sub-window size
	numSubs    int           // number of sub-windows
}

// NewWindow creates a new sliding window counter.
// The window is divided into granularity sub-windows for smoother rate limiting.
// A higher granularity gives more accurate results but uses more memory.
func NewWindow(rate int, window time.Duration, granularity int) *Window {
	if granularity <= 0 {
		granularity = 10
	}
	if rate <= 0 {
		rate = 1
	}
	if window <= 0 {
		window = time.Minute
	}

	subSize := window / time.Duration(granularity)
	if subSize == 0 {
		subSize = time.Millisecond
	}

	return &Window{
		subWindows: make(map[int64]int),
		rate:       rate,
		window:     window,
		subSize:    subSize,
		numSubs:    granularity,
	}
}

// Allow checks if a request is allowed and records it if so.
// Returns allowed status, the current count within the window, and
// the time when the window will reset enough to allow a new request.
func (w *Window) Allow(now time.Time) (allowed bool, currentCount int, resetAt time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.cleanup(now)

	count := w.count(now)
	if count >= w.rate {
		// Find the earliest sub-window that contributes to the count
		// and calculate when it will expire
		resetAt = w.earliestExpiry(now)
		return false, count, resetAt
	}

	// Record the request in the current sub-window
	subKey := w.subWindowKey(now)
	w.subWindows[subKey]++

	return true, count + 1, now.Add(w.window)
}

// Count returns the current request count within the sliding window
// without recording a new request.
func (w *Window) Count(now time.Time) int {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.cleanup(now)
	return w.count(now)
}

// Reset clears all recorded requests.
func (w *Window) Reset() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.subWindows = make(map[int64]int)
}

// Rate returns the configured maximum requests per window.
func (w *Window) Rate() int {
	return w.rate
}

// WindowDuration returns the configured window duration.
func (w *Window) WindowDuration() time.Duration {
	return w.window
}

// count returns the total count across all active sub-windows.
// Must be called with lock held.
func (w *Window) count(now time.Time) int {
	total := 0
	windowStart := now.Add(-w.window)

	for ts, c := range w.subWindows {
		subStart := time.Unix(0, ts)
		if !subStart.Before(windowStart) {
			total += c
		}
	}

	return total
}

// cleanup removes expired sub-windows.
// Must be called with lock held.
func (w *Window) cleanup(now time.Time) {
	windowStart := now.Add(-w.window)

	for ts := range w.subWindows {
		subStart := time.Unix(0, ts)
		if subStart.Before(windowStart) {
			delete(w.subWindows, ts)
		}
	}
}

// subWindowKey returns the sub-window key for the given time.
// Must be called with lock held.
func (w *Window) subWindowKey(t time.Time) int64 {
	// Truncate to sub-window boundary
	nanos := t.UnixNano()
	subNanos := w.subSize.Nanoseconds()
	return (nanos / subNanos) * subNanos
}

// earliestExpiry finds when the earliest active sub-window will expire.
// Must be called with lock held.
func (w *Window) earliestExpiry(now time.Time) time.Time {
	windowStart := now.Add(-w.window)
	earliest := now.Add(w.window) // worst case

	for ts := range w.subWindows {
		subStart := time.Unix(0, ts)
		if !subStart.Before(windowStart) {
			// This sub-window expires at subStart + window
			expiry := subStart.Add(w.window)
			if expiry.Before(earliest) {
				earliest = expiry
			}
		}
	}

	return earliest
}
