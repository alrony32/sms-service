package ratelimit

import (
	"context"
	"time"
)

const providerKey = "ss:rl:provider"

type SMSLimiter struct {
	limiter *Limiter
	perMin  int
}

func NewSMSLimiter(l *Limiter, perMin int) *SMSLimiter {
	return &SMSLimiter{limiter: l, perMin: perMin}
}

func (s *SMSLimiter) Allow(ctx context.Context) (bool, error) {
	return s.limiter.Allow(ctx, providerKey, s.perMin, time.Minute)
}
