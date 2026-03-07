package adaptive

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAdjustRateLocked_ClampToMinRate verifies that the rate cannot drop below minRate.
func TestAdjustRateLocked_ClampToMinRate(t *testing.T) {
	a := New(2, 2, 10)
	// Trigger enough failures to push rate below minRate.
	// 3 failures = one decrement; starting at 2, min is 2, so after first
	// adjustRateLocked(-1) the rate should be clamped to minRate.
	for i := 0; i < 3; i++ {
		_ = a.Execute(context.Background(), func(_ context.Context) error {
			return errors.New("fail")
		})
	}
	assert.Equal(t, 2, a.CurrentRate(), "rate should be clamped to minRate")
}

// TestAdjustRateLocked_ClampToMaxRate verifies that the rate cannot exceed maxRate.
func TestAdjustRateLocked_ClampToMaxRate(t *testing.T) {
	a := New(9, 1, 10)
	// 11 successes = one increment from 9 to 10
	for i := 0; i < 11; i++ {
		_ = a.Execute(context.Background(), func(_ context.Context) error {
			return nil
		})
	}
	assert.Equal(t, 10, a.CurrentRate())

	// Another 11 successes should NOT push rate past maxRate
	for i := 0; i < 11; i++ {
		_ = a.Execute(context.Background(), func(_ context.Context) error {
			return nil
		})
	}
	assert.Equal(t, 10, a.CurrentRate(), "rate should be clamped to maxRate")
}

// TestAdjustRateLocked_MultipleDecrements ensures repeated failures keep
// decrementing down to minRate.
func TestAdjustRateLocked_MultipleDecrements(t *testing.T) {
	a := New(5, 1, 10)
	// 3 failures per decrement, need 4 decrements to go from 5 to 1
	for round := 0; round < 4; round++ {
		for i := 0; i < 3; i++ {
			_ = a.Execute(context.Background(), func(_ context.Context) error {
				return errors.New("fail")
			})
		}
	}
	assert.Equal(t, 1, a.CurrentRate(), "rate should have decremented to minRate")

	// One more round of failures should not drop below minRate
	for i := 0; i < 3; i++ {
		_ = a.Execute(context.Background(), func(_ context.Context) error {
			return errors.New("fail")
		})
	}
	assert.Equal(t, 1, a.CurrentRate(), "rate should stay at minRate")
}

// TestConcurrentExecute verifies thread safety of Execute.
func TestConcurrentExecute(t *testing.T) {
	a := New(50, 1, 100)
	var wg sync.WaitGroup

	// Mix of successes and failures concurrently
	for i := 0; i < 200; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			if i%2 == 0 {
				_ = a.Execute(context.Background(), func(_ context.Context) error {
					return nil
				})
			} else {
				_ = a.Execute(context.Background(), func(_ context.Context) error {
					return errors.New("fail")
				})
			}
		}()
	}
	wg.Wait()

	rate := a.CurrentRate()
	assert.GreaterOrEqual(t, rate, 1)
	assert.LessOrEqual(t, rate, 100)
}

// TestAllow_ConcurrentCalls verifies thread safety of Allow.
func TestAllow_ConcurrentCalls(t *testing.T) {
	a := New(10, 1, 100)
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := a.Allow(context.Background(), "key")
			assert.NoError(t, err)
			assert.True(t, result.Allowed)
		}()
	}
	wg.Wait()
}

// TestExecute_ContextPassed verifies the context is passed to the operation.
func TestExecute_ContextPassed(t *testing.T) {
	a := New(5, 1, 10)
	type ctxKey string
	ctx := context.WithValue(context.Background(), ctxKey("key"), "value")

	err := a.Execute(ctx, func(c context.Context) error {
		val := c.Value(ctxKey("key"))
		assert.Equal(t, "value", val)
		return nil
	})
	assert.NoError(t, err)
}

// TestNew_EqualMinMax verifies behavior when minRate == maxRate.
func TestNew_EqualMinMax(t *testing.T) {
	a := New(5, 5, 5)
	assert.Equal(t, 5, a.CurrentRate())

	// Successes should not increase beyond max
	for i := 0; i < 20; i++ {
		_ = a.Execute(context.Background(), func(_ context.Context) error {
			return nil
		})
	}
	assert.Equal(t, 5, a.CurrentRate())

	// Failures should not decrease below min
	for i := 0; i < 10; i++ {
		_ = a.Execute(context.Background(), func(_ context.Context) error {
			return errors.New("fail")
		})
	}
	assert.Equal(t, 5, a.CurrentRate())
}
