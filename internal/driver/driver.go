package driver

import (
	"context"

	"github.com/sms-service/internal/entity"
)

type Result struct {
	ID     string
	Status string
	Error  string
}

type Driver interface {
	SendBatch(ctx context.Context, msgs []entity.SMS) ([]Result, error)
}
