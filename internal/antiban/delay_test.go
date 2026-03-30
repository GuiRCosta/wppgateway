package antiban_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/guilhermecosta/wpp-gateway/internal/antiban"
)

func TestHumanizedDelay(t *testing.T) {
	for range 50 {
		d := antiban.HumanizedDelay(2000, 5000)
		assert.GreaterOrEqual(t, d, 2000*time.Millisecond)
		assert.Less(t, d, 5000*time.Millisecond)
	}
}

func TestHumanizedDelayEqualBounds(t *testing.T) {
	d := antiban.HumanizedDelay(1000, 1000)
	assert.Equal(t, 1000*time.Millisecond, d)
}

func TestTypingDelay(t *testing.T) {
	short := antiban.TypingDelay(10)
	assert.GreaterOrEqual(t, short, 1*time.Second)

	long := antiban.TypingDelay(500)
	assert.LessOrEqual(t, long, 9*time.Second)
}

func TestChunkPause(t *testing.T) {
	d := antiban.ChunkPause()
	assert.GreaterOrEqual(t, d, 30*time.Second)
	assert.Less(t, d, 120*time.Second)
}
