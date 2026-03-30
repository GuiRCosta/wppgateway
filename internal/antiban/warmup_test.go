package antiban_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/guilhermecosta/wpp-gateway/internal/antiban"
)

func TestWarmupBudget(t *testing.T) {
	tests := []struct {
		name     string
		budget   int
		day      int
		expected int
	}{
		{"day 0 (no warmup)", 200, 0, 200},
		{"day 1", 200, 1, 20},
		{"day 3", 200, 3, 20},
		{"day 4", 200, 4, 50},
		{"day 7", 200, 7, 50},
		{"day 8", 200, 8, 100},
		{"day 10", 200, 10, 100},
		{"day 11", 200, 11, 150},
		{"day 14", 200, 14, 150},
		{"day 15 (full)", 200, 15, 200},
		{"day 30 (full)", 200, 30, 200},
		{"small budget day 1", 5, 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := antiban.WarmupBudget(tt.budget, tt.day)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsWarmingUp(t *testing.T) {
	assert.False(t, antiban.IsWarmingUp(0))
	assert.True(t, antiban.IsWarmingUp(1))
	assert.True(t, antiban.IsWarmingUp(14))
	assert.False(t, antiban.IsWarmingUp(15))
	assert.False(t, antiban.IsWarmingUp(-1))
}
