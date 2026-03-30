package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

type Event struct {
	Name       string `json:"event"`
	GroupID    string `json:"group_id"`
	InstanceID string `json:"instance_id,omitempty"`
	Phone      string `json:"phone_number,omitempty"`
	Timestamp  string `json:"timestamp"`
	Data       any    `json:"data"`
}

type Emitter struct {
	client     *http.Client
	maxRetries int
	events     chan emitRequest
	log        zerolog.Logger
}

type emitRequest struct {
	url    string
	secret string
	event  Event
}

func NewEmitter(timeout time.Duration, maxRetries int, log zerolog.Logger) *Emitter {
	e := &Emitter{
		client: &http.Client{
			Timeout: timeout,
		},
		maxRetries: maxRetries,
		events:     make(chan emitRequest, 1000),
		log:        log,
	}
	return e
}

func (e *Emitter) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case req := <-e.events:
				e.deliver(req)
			}
		}
	}()
}

func (e *Emitter) Emit(url string, secret string, event Event) {
	if url == "" {
		return
	}
	select {
	case e.events <- emitRequest{url: url, secret: secret, event: event}:
	default:
		e.log.Warn().Str("event", event.Name).Msg("webhook channel full, dropping event")
	}
}

func (e *Emitter) deliver(req emitRequest) {
	body, err := json.Marshal(req.event)
	if err != nil {
		e.log.Error().Err(err).Msg("failed to marshal webhook payload")
		return
	}

	signature := Sign(body, req.secret)

	for attempt := range e.maxRetries {
		if err := e.send(req.url, body, signature, req.event.Name); err != nil {
			e.log.Warn().
				Err(err).
				Int("attempt", attempt+1).
				Str("url", req.url).
				Msg("webhook delivery failed")

			backoff := time.Duration(1<<uint(attempt)) * time.Second
			time.Sleep(backoff)
			continue
		}
		return
	}

	e.log.Error().
		Str("url", req.url).
		Str("event", req.event.Name).
		Msg("webhook delivery failed after all retries")
}

func (e *Emitter) send(url string, body []byte, signature string, eventType string) error {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", signature)
	req.Header.Set("X-Event-Type", eventType)

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("server error: status %d", resp.StatusCode)
	}

	return nil
}
