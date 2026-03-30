package orchestrator

import (
	"context"
	"encoding/json"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/guilhermecosta/wpp-gateway/internal/antiban"
	"github.com/guilhermecosta/wpp-gateway/internal/domain"
	"github.com/guilhermecosta/wpp-gateway/internal/instance"
)

// Dispatcher processes broadcast jobs, distributing messages across instances.
type Dispatcher struct {
	orchestrator  *GroupOrchestrator
	manager       *instance.Manager
	broadcastRepo domain.BroadcastRepository
	log           zerolog.Logger

	activeJobs map[uuid.UUID]context.CancelFunc
	mu         sync.Mutex
}

func NewDispatcher(
	orch *GroupOrchestrator,
	mgr *instance.Manager,
	bcastRepo domain.BroadcastRepository,
	log zerolog.Logger,
) *Dispatcher {
	return &Dispatcher{
		orchestrator:  orch,
		manager:       mgr,
		broadcastRepo: bcastRepo,
		log:           log.With().Str("component", "dispatcher").Logger(),
		activeJobs:    make(map[uuid.UUID]context.CancelFunc),
	}
}

// StartBroadcast begins processing a broadcast asynchronously.
func (d *Dispatcher) StartBroadcast(ctx context.Context, broadcast *domain.Broadcast) {
	jobCtx, cancel := context.WithCancel(ctx)

	d.mu.Lock()
	d.activeJobs[broadcast.ID] = cancel
	d.mu.Unlock()

	go d.processBroadcast(jobCtx, broadcast)
}

// PauseBroadcast pauses an active broadcast.
func (d *Dispatcher) PauseBroadcast(broadcastID uuid.UUID) error {
	d.mu.Lock()
	cancel, exists := d.activeJobs[broadcastID]
	d.mu.Unlock()

	if exists {
		cancel()
		delete(d.activeJobs, broadcastID)
	}
	return d.broadcastRepo.UpdateStatus(context.Background(), broadcastID, domain.BcastPaused)
}

// ResumeBroadcast resumes a paused broadcast.
func (d *Dispatcher) ResumeBroadcast(ctx context.Context, broadcastID uuid.UUID) error {
	broadcast, err := d.broadcastRepo.FindByID(ctx, broadcastID)
	if err != nil || broadcast == nil {
		return err
	}

	if err := d.broadcastRepo.UpdateStatus(ctx, broadcastID, domain.BcastProcessing); err != nil {
		return err
	}

	d.StartBroadcast(ctx, broadcast)
	return nil
}

// CancelBroadcast cancels an active broadcast.
func (d *Dispatcher) CancelBroadcast(broadcastID uuid.UUID) error {
	d.mu.Lock()
	cancel, exists := d.activeJobs[broadcastID]
	if exists {
		cancel()
		delete(d.activeJobs, broadcastID)
	}
	d.mu.Unlock()

	return d.broadcastRepo.UpdateStatus(context.Background(), broadcastID, domain.BcastCancelled)
}

func (d *Dispatcher) processBroadcast(ctx context.Context, broadcast *domain.Broadcast) {
	defer func() {
		d.mu.Lock()
		delete(d.activeJobs, broadcast.ID)
		d.mu.Unlock()
	}()

	if err := d.broadcastRepo.MarkStarted(ctx, broadcast.ID); err != nil {
		d.log.Error().Err(err).Msg("failed to mark broadcast as started")
		return
	}

	d.log.Info().
		Str("broadcast_id", broadcast.ID.String()).
		Int("total", broadcast.Total).
		Msg("starting broadcast")

	// Parse options
	var opts domain.BroadcastOptions
	_ = json.Unmarshal(broadcast.Options, &opts)

	// Parse variables
	var variables map[string]map[string]string
	_ = json.Unmarshal(broadcast.Variables, &variables)

	// Parse content
	var content struct {
		Body string `json:"body"`
	}
	_ = json.Unmarshal(broadcast.Content, &content)

	sent, failed := 0, 0
	chunkSize := 20

	for {
		select {
		case <-ctx.Done():
			d.log.Info().Str("broadcast_id", broadcast.ID.String()).Msg("broadcast cancelled/paused")
			return
		default:
		}

		recipients, err := d.broadcastRepo.GetPendingRecipients(ctx, broadcast.ID, chunkSize)
		if err != nil {
			d.log.Error().Err(err).Msg("failed to get pending recipients")
			break
		}
		if len(recipients) == 0 {
			break
		}

		// Shuffle if enabled
		if opts.ShuffleRecipients {
			rand.Shuffle(len(recipients), func(i, j int) {
				recipients[i], recipients[j] = recipients[j], recipients[i]
			})
		}

		for _, recipient := range recipients {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Select instance via orchestrator
			inst, selectErr := d.orchestrator.SelectInstance(ctx, broadcast.GroupID)
			if selectErr != nil {
				d.log.Error().Err(selectErr).Msg("no instance available for broadcast")
				_ = d.broadcastRepo.UpdateRecipientFailed(ctx, recipient.ID, "no_instance")
				failed++
				continue
			}

			// Build message text with variable substitution
			msgText := content.Body
			if variables != nil {
				if vars, ok := variables[recipient.Recipient]; ok {
					for key, val := range vars {
						msgText = replaceVar(msgText, key, val)
					}
				}
			}

			// Apply spintax if enabled
			if opts.Spintax {
				msgText = antiban.ProcessSpintax(msgText)
			}

			// Send message
			_, sendErr := d.manager.SendTextMessage(ctx, inst.ID, recipient.Recipient, msgText)
			if sendErr != nil {
				d.log.Warn().Err(sendErr).
					Str("recipient", recipient.Recipient).
					Msg("failed to send broadcast message")
				_ = d.broadcastRepo.UpdateRecipientFailed(ctx, recipient.ID, "send_failed")
				failed++
			} else {
				_ = d.broadcastRepo.UpdateRecipientSent(ctx, recipient.ID, inst.ID)
				sent++
			}

			// Update progress periodically
			if (sent+failed)%10 == 0 {
				_ = d.broadcastRepo.UpdateProgress(ctx, broadcast.ID, sent, 0, failed)
			}

			// Humanized delay between messages
			delay := antiban.HumanizedDelay(2000, 5000)
			time.Sleep(delay)
		}

		// Pause between chunks
		time.Sleep(antiban.HumanizedDelay(5000, 15000))
	}

	// Final update
	_ = d.broadcastRepo.UpdateProgress(ctx, broadcast.ID, sent, 0, failed)
	_ = d.broadcastRepo.MarkCompleted(ctx, broadcast.ID)

	d.log.Info().
		Str("broadcast_id", broadcast.ID.String()).
		Int("sent", sent).
		Int("failed", failed).
		Msg("broadcast completed")
}

func replaceVar(text, key, value string) string {
	placeholder := "{{" + key + "}}"
	result := ""
	for i := 0; i < len(text); {
		if i+len(placeholder) <= len(text) && text[i:i+len(placeholder)] == placeholder {
			result += value
			i += len(placeholder)
		} else {
			result += string(text[i])
			i++
		}
	}
	return result
}
