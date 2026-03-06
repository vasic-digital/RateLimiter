package tokenbucket

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	tb := New(10, 5.0)
	assert.NotNil(t, tb)
	assert.Equal(t, 10, tb.AvailableTokens())
}

func TestTryAcquire_ConsumesToken(t *testing.T) {
	tb := New(5, 1.0)
	assert.True(t, tb.TryAcquire())
	assert.Equal(t, 4, tb.AvailableTokens())
}

func TestTryAcquire_FailsWhenEmpty(t *testing.T) {
	tb := New(1, 0.001)
	assert.True(t, tb.TryAcquire())
	assert.False(t, tb.TryAcquire())
}

func TestAcquire_SucceedsWhenAvailable(t *testing.T) {
	tb := New(5, 1.0)
	tb.Acquire()
	assert.Equal(t, 4, tb.AvailableTokens())
}

func TestAvailableTokens_Initial(t *testing.T) {
	tb := New(10, 5.0)
	assert.Equal(t, 10, tb.AvailableTokens())
}
