package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/sms-service/internal/config"
	"github.com/sms-service/internal/entity"
	"github.com/sms-service/internal/queue"
	"github.com/sms-service/internal/ratelimit"
	"github.com/sms-service/pkg/logger"
)

type WebhookWorker struct {
	repo       queue.Repository
	limiter    *ratelimit.WebhookLimiter
	http       *http.Client
	maxRetries int
	idle       time.Duration
}

func NewWebhookWorker(repo queue.Repository, limiter *ratelimit.WebhookLimiter, cfg *config.Config) *WebhookWorker {
	timeout := time.Duration(cfg.Webhook.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &WebhookWorker{
		repo:       repo,
		limiter:    limiter,
		http:       &http.Client{Timeout: timeout},
		maxRetries: cfg.Webhook.MaxRetries,
		idle:       time.Duration(cfg.Scheduler.IntervalMs) * time.Millisecond,
	}
}

func (w *WebhookWorker) Run(ctx context.Context) {
	logger.Info("webhook worker started")
	for {
		select {
		case <-ctx.Done():
			logger.Info("webhook worker stopped")
			return
		default:
		}

		if !w.cycle(ctx) {
			w.sleep(ctx)
		}
	}
}

func (w *WebhookWorker) cycle(ctx context.Context) bool {
	clients, err := w.repo.Clients(ctx)
	if err != nil {
		logger.Error("webhook: list clients", err.Error())
		return false
	}

	worked := false
	for _, client := range clients {
		worked = w.drainClient(ctx, client) || worked
	}
	return worked
}

func (w *WebhookWorker) drainClient(ctx context.Context, client string) bool {
	worked := false
	for {
		select {
		case <-ctx.Done():
			return worked
		default:
		}

		events, err := w.repo.DequeueWebhook(ctx, client, 1)
		if err != nil {
			logger.Error("webhook: dequeue", client, err.Error())
			return worked
		}
		if len(events) == 0 {
			return worked
		}
		ev := events[0]

		allowed, err := w.limiter.Allow(ctx, client)
		if err != nil {
			logger.Error("webhook: rate limit check", client, err.Error())
			_ = w.repo.EnqueueWebhook(ctx, ev)
			return worked
		}
		if !allowed {
			_ = w.repo.EnqueueWebhook(ctx, ev)
			return worked
		}

		worked = true
		w.deliver(ctx, ev)
	}
}

func (w *WebhookWorker) deliver(ctx context.Context, ev entity.WebhookEvent) {
	if ev.WebhookURL == "" {
		logger.Error("webhook: missing url, dropping", ev.Client, ev.ID)
		return
	}

	body, err := json.Marshal(ev.Payload())
	if err != nil {
		logger.Error("webhook: marshal", ev.ID, err.Error())
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ev.WebhookURL, bytes.NewReader(body))
	if err != nil {
		w.retry(ctx, ev, err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.http.Do(req)
	if err != nil {
		w.retry(ctx, ev, err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		w.retry(ctx, ev, "http status "+resp.Status)
		return
	}

	_ = w.repo.IncrBatchStatus(ctx, ev.BatchID, "delivered", 1)
	logger.Info("webhook: delivered", "client", ev.Client, "id", ev.ID, "status", ev.Status)
}

func (w *WebhookWorker) retry(ctx context.Context, ev entity.WebhookEvent, reason string) {
	ev.Attempts++
	if w.maxRetries > 0 && ev.Attempts >= w.maxRetries {
		logger.Error("webhook: giving up", "client", ev.Client, "id", ev.ID, "attempts", ev.Attempts, "reason", reason)
		return
	}
	logger.Error("webhook: delivery failed, requeueing", "client", ev.Client, "id", ev.ID, "attempt", ev.Attempts, "reason", reason)
	_ = w.repo.EnqueueWebhook(ctx, ev)
}

func (w *WebhookWorker) sleep(ctx context.Context) {
	if w.idle <= 0 {
		w.idle = time.Second
	}
	t := time.NewTimer(w.idle)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}
