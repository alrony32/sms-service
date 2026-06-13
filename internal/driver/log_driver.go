package driver

import (
	"context"

	"github.com/sms-service/internal/entity"
	"github.com/sms-service/pkg/logger"
)

type LogDriver struct{}

func NewLogDriver() *LogDriver {
	return &LogDriver{}
}

func (d *LogDriver) SendBatch(_ context.Context, msgs []entity.SMS) ([]Result, error) {
	results := make([]Result, 0, len(msgs))
	for _, m := range msgs {
		logger.Info("LOG DRIVER - sending SMS", "id", m.ID, "to", m.To, "client", m.Client)
		results = append(results, Result{ID: m.ID, Status: entity.StatusSent})
	}
	return results, nil
}
