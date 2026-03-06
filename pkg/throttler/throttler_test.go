package throttler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	th := New(time.Second, 5)
	assert.NotNil(t, th)
}

func TestTryThrottle_FirstOperationAllowed(t *testing.T) {
	th := New(time.Second, 5)
	assert.True(t, th.TryThrottle("op1"))
}

func TestTryThrottle_ThrottlesAfterMax(t *testing.T) {
	th := New(time.Minute, 3)
	assert.True(t, th.TryThrottle("op1"))
	assert.True(t, th.TryThrottle("op1"))
	assert.True(t, th.TryThrottle("op1"))
	assert.False(t, th.TryThrottle("op1"))
}

func TestTryThrottle_DifferentIDsIndependent(t *testing.T) {
	th := New(time.Minute, 1)
	assert.True(t, th.TryThrottle("op1"))
	assert.True(t, th.TryThrottle("op2"))
	assert.False(t, th.TryThrottle("op1"))
}

func TestClear_ResetsThrottle(t *testing.T) {
	th := New(time.Minute, 1)
	assert.True(t, th.TryThrottle("op1"))
	assert.False(t, th.TryThrottle("op1"))
	th.Clear("op1")
	assert.True(t, th.TryThrottle("op1"))
}
