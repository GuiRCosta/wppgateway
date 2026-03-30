package logger

import (
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

var Buffer = NewRingBuffer(500)

type bufferWriter struct {
	buf *RingBuffer
}

func (bw *bufferWriter) Write(p []byte) (n int, err error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(p, &raw); err != nil {
		return len(p), nil
	}

	entry := LogEntry{
		Timestamp: time.Now(),
		Fields:    make(map[string]interface{}),
	}

	if v, ok := raw["level"].(string); ok {
		entry.Level = v
	}
	if v, ok := raw["message"].(string); ok {
		entry.Message = v
	}
	if v, ok := raw["caller"].(string); ok {
		entry.Caller = v
	}

	for k, v := range raw {
		switch k {
		case "level", "message", "caller", "time":
			continue
		default:
			entry.Fields[k] = v
		}
	}

	bw.buf.Write(entry)
	return len(p), nil
}

func New(level string) zerolog.Logger {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}

	console := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}

	multi := io.MultiWriter(console, &bufferWriter{buf: Buffer})

	return zerolog.New(multi).
		Level(lvl).
		With().
		Timestamp().
		Caller().
		Logger()
}
