package antiban

import (
	"math/rand/v2"
	"time"
)

// HumanizedDelay returns a random duration between minMs and maxMs milliseconds.
func HumanizedDelay(minMs, maxMs int) time.Duration {
	if maxMs <= minMs {
		return time.Duration(minMs) * time.Millisecond
	}
	jitter := rand.IntN(maxMs - minMs)
	return time.Duration(minMs+jitter) * time.Millisecond
}

// TypingDelay returns a delay proportional to message length,
// simulating human typing speed.
func TypingDelay(messageLen int) time.Duration {
	// Average human types ~40 chars per second
	// But we want a minimum of 1s and max of 8s
	charsPerSecond := 40
	seconds := messageLen / charsPerSecond

	if seconds < 1 {
		seconds = 1
	}
	if seconds > 8 {
		seconds = 8
	}

	// Add some jitter
	jitter := rand.IntN(1000)
	return time.Duration(seconds)*time.Second + time.Duration(jitter)*time.Millisecond
}

// ChunkPause returns a longer pause for between chunks of messages.
// Should be called every 20-30 messages.
func ChunkPause() time.Duration {
	return HumanizedDelay(30000, 120000) // 30s to 2min
}
