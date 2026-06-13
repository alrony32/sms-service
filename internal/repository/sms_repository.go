package repository

import (
	"context"

	"github.com/sms-service/internal/entity"
)

type Repository interface {
	Create(
		ctx context.Context,
		sms entity.SMS,
	) error

	CreateBatch(
		ctx context.Context,
		sms []entity.SMS,
	) error
}
