package adaptive

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	a := New(5, 1, 20)
	assert.NotNil(t, a)
	assert.Equal(t, 5, a.CurrentRate())
}

func TestExecute_Success(t *testing.T) {
	a := New(5, 1, 20)
	err := a.Execute(context.Background(), func(_ context.Context) error {
		return nil
	})
	assert.NoError(t, err)
}

func TestExecute_Failure(t *testing.T) {
	a := New(5, 1, 20)
	err := a.Execute(context.Background(), func(_ context.Context) error {
		return errors.New("fail")
	})
	assert.Error(t, err)
}

func TestRateIncreasesAfterSuccesses(t *testing.T) {
	a := New(5, 1, 20)
	for i := 0; i < 15; i++ {
		_ = a.Execute(context.Background(), func(_ context.Context) error {
			return nil
		})
	}
	rate := a.CurrentRate()
	assert.Greater(t, rate, 5)
}

func TestRateDecreasesAfterFailures(t *testing.T) {
	a := New(5, 1, 20)
	for i := 0; i < 5; i++ {
		_ = a.Execute(context.Background(), func(_ context.Context) error {
			return errors.New("fail")
		})
	}
	rate := a.CurrentRate()
	assert.Less(t, rate, 5)
}

func TestRateDoesNotExceedBounds(t *testing.T) {
	a := New(5, 2, 8)
	for i := 0; i < 100; i++ {
		_ = a.Execute(context.Background(), func(_ context.Context) error {
			return nil
		})
	}
	rate := a.CurrentRate()
	assert.GreaterOrEqual(t, rate, 2)
	assert.LessOrEqual(t, rate, 8)
}

func TestAllow_ReturnsResult(t *testing.T) {
	a := New(5, 1, 20)
	result, err := a.Allow(context.Background(), "key")
	assert.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, 5, result.Limit)
}
