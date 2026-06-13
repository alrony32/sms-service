package worker

import (
	"context"
	"time"

	"github.com/sms-service/internal/config"
	"github.com/sms-service/internal/driver"
	"github.com/sms-service/internal/entity"
	"github.com/sms-service/internal/queue"
	"github.com/sms-service/internal/ratelimit"
	"github.com/sms-service/pkg/logger"
)

type Dispatcher struct {
	repo      queue.Repository
	drv       driver.Driver
	limiter   *ratelimit.SMSLimiter
	batchSize int
	idle      time.Duration
}

func NewDispatcher(repo queue.Repository, drv driver.Driver, limiter *ratelimit.SMSLimiter, cfg *config.Config) *Dispatcher {
	batchSize := cfg.Provider.BatchSize
	if batchSize <= 0 {
		batchSize = 50
	}
	return &Dispatcher{
		repo:      repo,
		drv:       drv,
		limiter:   limiter,
		batchSize: batchSize,
		idle:      time.Duration(cfg.Scheduler.IntervalMs) * time.Millisecond,
	}
}

func (d *Dispatcher) Run(ctx context.Context) {
	logger.Info("dispatcher started", "batch_size", d.batchSize)
	for {
		select {
		case <-ctx.Done():
			logger.Info("dispatcher stopped")
			return
		default:
		}

		worked := d.cycle(ctx)
		if !worked {

			d.sleep(ctx)
		}
	}
}

func (d *Dispatcher) cycle(ctx context.Context) bool {
	clients, err := d.repo.Clients(ctx)
	if err != nil {
		logger.Error("dispatcher: list clients", err.Error())
		return false
	}

	worked := false
	for _, client := range clients {

		allowed, err := d.limiter.Allow(ctx)
		if err != nil {
			logger.Error("dispatcher: rate limit check", err.Error())
			continue
		}
		if !allowed {

			return worked
		}

		msgs, err := d.repo.DequeueSMS(ctx, client, d.batchSize)
		if err != nil {
			logger.Error("dispatcher: dequeue", client, err.Error())
			continue
		}
		if len(msgs) == 0 {
			continue
		}

		worked = true
		d.dispatch(ctx, client, msgs)
	}
	return worked
}

func (d *Dispatcher) dispatch(ctx context.Context, client string, msgs []entity.SMS) {
	results, err := d.drv.SendBatch(ctx, msgs)
	if err != nil {
		logger.Error("dispatcher: provider send", client, err.Error())
	}

	byID := make(map[string]driver.Result, len(results))
	for _, r := range results {
		byID[r.ID] = r
	}

	batchID := ""
	for _, m := range msgs {
		batchID = m.BatchID
		status := entity.StatusFailed
		errMsg := "no provider response"
		if r, ok := byID[m.ID]; ok {
			status = r.Status
			errMsg = r.Error
		}

		_ = d.repo.UpdateMessageStatus(ctx, client, m.ID, status)

		_ = d.repo.IncrBatchStatus(ctx, m.BatchID, "queued", -1)
		_ = d.repo.IncrBatchStatus(ctx, m.BatchID, status, 1)

		_ = d.repo.EnqueueWebhook(ctx, entity.WebhookEvent{
			ID:         m.ID,
			BatchID:    m.BatchID,
			Client:     client,
			To:         m.To,
			Status:     status,
			Error:      errMsg,
			WebhookURL: m.WebhookURL,
		})
	}
	logger.Info("dispatcher: batch sent", "client", client, "count", len(msgs), "batch", batchID)
}

func (d *Dispatcher) sleep(ctx context.Context) {
	if d.idle <= 0 {
		d.idle = time.Second
	}
	t := time.NewTimer(d.idle)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}
