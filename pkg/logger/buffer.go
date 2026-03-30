package logger

import (
	"sync"
	"time"
)

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Caller    string    `json:"caller,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

type RingBuffer struct {
	mu      sync.RWMutex
	entries []LogEntry
	size    int
	cursor  int
	count   int
}

func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		entries: make([]LogEntry, size),
		size:    size,
	}
}

func (rb *RingBuffer) Write(entry LogEntry) {
	rb.mu.Lock()
	rb.entries[rb.cursor] = entry
	rb.cursor = (rb.cursor + 1) % rb.size
	if rb.count < rb.size {
		rb.count++
	}
	rb.mu.Unlock()
}

func (rb *RingBuffer) Entries(level string, limit int) []LogEntry {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if limit <= 0 || limit > rb.count {
		limit = rb.count
	}

	result := make([]LogEntry, 0, limit)

	start := (rb.cursor - rb.count + rb.size) % rb.size
	for i := rb.count - 1; i >= 0 && len(result) < limit; i-- {
		idx := (start + i) % rb.size
		entry := rb.entries[idx]
		if level != "" && entry.Level != level {
			continue
		}
		result = append(result, entry)
	}

	return result
}
