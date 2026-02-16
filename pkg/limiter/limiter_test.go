package limiter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	require.NotNil(t, cfg)
	assert.Equal(t, 100, cfg.Rate)
	assert.Equal(t, time.Minute, cfg.Window)
	assert.Equal(t, 0, cfg.Burst)
}

func TestEffectiveBurst(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected int
	}{
		{
			name:     "burst zero defaults to rate",
			config:   Config{Rate: 50, Burst: 0},
			expected: 50,
		},
		{
			name:     "burst negative defaults to rate",
			config:   Config{Rate: 30, Burst: -1},
			expected: 30,
		},
		{
			name:     "explicit burst value",
			config:   Config{Rate: 100, Burst: 200},
			expected: 200,
		},
		{
			name:     "burst equal to rate",
			config:   Config{Rate: 10, Burst: 10},
			expected: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config.EffectiveBurst())
		})
	}
}

func TestResultFields(t *testing.T) {
	now := time.Now()
	r := &Result{
		Allowed:    true,
		Remaining:  99,
		Limit:      100,
		RetryAfter: 0,
		ResetAt:    now.Add(time.Minute),
	}

	assert.True(t, r.Allowed)
	assert.Equal(t, 99, r.Remaining)
	assert.Equal(t, 100, r.Limit)
	assert.Equal(t, time.Duration(0), r.RetryAfter)
	assert.WithinDuration(t, now.Add(time.Minute), r.ResetAt, time.Millisecond)
}

func TestResultDenied(t *testing.T) {
	r := &Result{
		Allowed:    false,
		Remaining:  0,
		Limit:      10,
		RetryAfter: 5 * time.Second,
		ResetAt:    time.Now().Add(5 * time.Second),
	}

	assert.False(t, r.Allowed)
	assert.Equal(t, 0, r.Remaining)
	assert.Equal(t, 5*time.Second, r.RetryAfter)
}
