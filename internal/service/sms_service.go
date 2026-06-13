package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/sms-service/internal/dto"
	"github.com/sms-service/internal/entity"
	"github.com/sms-service/internal/queue"
	"github.com/sms-service/internal/validator"
)

type Service interface {
	SendSingle(ctx context.Context, req dto.SingleSMSRequest) (batchID string, err error)
	SendBulk(ctx context.Context, req dto.BulkSMSRequest) (batchID string, err error)
	GetBatch(ctx context.Context, batchID string) (map[string]string, error)
	QueueSizes(ctx context.Context) (map[string]int64, error)
}

type SMSService struct {
	queue queue.Repository
}

func NewSMSService(q queue.Repository) Service {
	return &SMSService{queue: q}
}

func (s *SMSService) SendSingle(ctx context.Context, req dto.SingleSMSRequest) (string, error) {
	items := []dto.BulkSMSItem{{ID: req.ID, To: req.To, Message: req.Message}}
	return s.enqueueBatch(ctx, req.Client, req.From, req.WebhookURL, items, entity.PriorityHigh)
}

func (s *SMSService) SendBulk(ctx context.Context, req dto.BulkSMSRequest) (string, error) {
	return s.enqueueBatch(ctx, req.Client, req.From, req.WebhookURL, req.Messages, entity.PriorityNormal)
}

func (s *SMSService) enqueueBatch(ctx context.Context, client, from, webhookURL string, items []dto.BulkSMSItem, priority string) (string, error) {
	batchID := newBatchID()
	client = queue.NormalizeClient(client)
	now := time.Now().UTC()

	if err := s.queue.CreateBatch(ctx, batchID, client, len(items)); err != nil {
		return "", err
	}

	for _, item := range items {
		sms := entity.SMS{
			ID:         item.ID,
			BatchID:    batchID,
			From:       from,
			To:         validator.NormalizeBDPhone(item.To),
			Message:    item.Message,
			Client:     client,
			Status:     entity.StatusQueued,
			Priority:   priority,
			WebhookURL: webhookURL,
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		if err := s.queue.SetMessageStatus(ctx, sms); err != nil {
			return batchID, err
		}
		if err := s.queue.EnqueueSMS(ctx, sms); err != nil {
			return batchID, err
		}
	}

	return batchID, nil
}

func (s *SMSService) GetBatch(ctx context.Context, batchID string) (map[string]string, error) {
	return s.queue.GetBatch(ctx, batchID)
}

func (s *SMSService) QueueSizes(ctx context.Context) (map[string]int64, error) {
	return s.queue.QueueSizes(ctx)
}

func newBatchID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {

		return "batch-" + time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(b)
}
