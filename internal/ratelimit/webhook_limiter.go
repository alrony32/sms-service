package ratelimit

import (
	"context"
	"time"
)

type WebhookLimiter struct {
	limiter *Limiter
	perMin  int
}

func NewWebhookLimiter(l *Limiter, perMin int) *WebhookLimiter {
	return &WebhookLimiter{limiter: l, perMin: perMin}
}

func (w *WebhookLimiter) Allow(ctx context.Context, client string) (bool, error) {
	return w.limiter.Allow(ctx, "ss:rl:webhook:"+client, w.perMin, time.Minute)
}
