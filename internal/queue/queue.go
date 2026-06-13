package queue

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/sms-service/internal/entity"
)

const (
	smsQueuePrefix     = "ss-db:"
	smsHighQueuePrefix = "ss-db-hp:"
	webhookQueuePrefix = "ss-webhook:"
	clientsKey         = "ss:clients"
	batchPrefix        = "ss:batch:"
	msgPrefix          = "ss:msg:"
)

type Repository interface {
	EnqueueSMS(ctx context.Context, sms entity.SMS) error
	DequeueSMS(ctx context.Context, client string, n int) ([]entity.SMS, error)

	EnqueueWebhook(ctx context.Context, ev entity.WebhookEvent) error
	DequeueWebhook(ctx context.Context, client string, n int) ([]entity.WebhookEvent, error)

	Clients(ctx context.Context) ([]string, error)

	CreateBatch(ctx context.Context, batchID, client string, total int) error
	IncrBatchStatus(ctx context.Context, batchID, status string, delta int64) error

	SetMessageStatus(ctx context.Context, sms entity.SMS) error
	UpdateMessageStatus(ctx context.Context, client, id, status string) error
}

type RedisRepository struct {
	client *goredis.Client
}

func NewRedisRepository(client *goredis.Client) Repository {
	return &RedisRepository{client: client}
}

func (r *RedisRepository) EnqueueSMS(ctx context.Context, sms entity.SMS) error {
	client := NormalizeClient(sms.Client)
	sms.Client = client

	payload, err := json.Marshal(sms)
	if err != nil {
		return err
	}

	pipe := r.client.TxPipeline()
	pipe.RPush(ctx, smsQueueKey(sms.Priority, client), payload)
	pipe.SAdd(ctx, clientsKey, client)
	_, err = pipe.Exec(ctx)
	return err
}

func (r *RedisRepository) DequeueSMS(ctx context.Context, client string, n int) ([]entity.SMS, error) {
	client = NormalizeClient(client)

	raw, err := r.popN(ctx, smsHighQueuePrefix+client, n)
	if err != nil {
		return nil, err
	}
	if len(raw) < n {
		normal, err := r.popN(ctx, smsQueuePrefix+client, n-len(raw))
		if err != nil {
			return nil, err
		}
		raw = append(raw, normal...)
	}

	out := make([]entity.SMS, 0, len(raw))
	for _, item := range raw {
		var sms entity.SMS
		if err := json.Unmarshal([]byte(item), &sms); err != nil {

			continue
		}
		out = append(out, sms)
	}
	return out, nil
}

func (r *RedisRepository) EnqueueWebhook(ctx context.Context, ev entity.WebhookEvent) error {
	client := NormalizeClient(ev.Client)
	ev.Client = client

	payload, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	return r.client.RPush(ctx, webhookQueuePrefix+client, payload).Err()
}

func (r *RedisRepository) DequeueWebhook(ctx context.Context, client string, n int) ([]entity.WebhookEvent, error) {
	client = NormalizeClient(client)
	raw, err := r.popN(ctx, webhookQueuePrefix+client, n)
	if err != nil {
		return nil, err
	}

	out := make([]entity.WebhookEvent, 0, len(raw))
	for _, item := range raw {
		var ev entity.WebhookEvent
		if err := json.Unmarshal([]byte(item), &ev); err != nil {
			continue
		}
		out = append(out, ev)
	}
	return out, nil
}

func (r *RedisRepository) Clients(ctx context.Context) ([]string, error) {
	return r.client.SMembers(ctx, clientsKey).Result()
}

func (r *RedisRepository) CreateBatch(ctx context.Context, batchID, client string, total int) error {
	return r.client.HSet(ctx, batchPrefix+batchID, map[string]any{
		"batch_id":   batchID,
		"client":     NormalizeClient(client),
		"total":      total,
		"queued":     total,
		"sent":       0,
		"failed":     0,
		"delivered":  0,
		"created_at": time.Now().UTC().Format(time.RFC3339),
	}).Err()
}

func (r *RedisRepository) IncrBatchStatus(ctx context.Context, batchID, status string, delta int64) error {
	return r.client.HIncrBy(ctx, batchPrefix+batchID, status, delta).Err()
}

func (r *RedisRepository) SetMessageStatus(ctx context.Context, sms entity.SMS) error {
	return r.client.HSet(ctx, msgKey(sms.Client, sms.ID), map[string]any{
		"batch_id":   sms.BatchID,
		"client":     NormalizeClient(sms.Client),
		"to":         sms.To,
		"status":     sms.Status,
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	}).Err()
}

func (r *RedisRepository) UpdateMessageStatus(ctx context.Context, client, id, status string) error {
	return r.client.HSet(ctx, msgKey(client, id), map[string]any{
		"status":     status,
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	}).Err()
}

func (r *RedisRepository) popN(ctx context.Context, key string, n int) ([]string, error) {
	if n <= 0 {
		return nil, nil
	}

	var rangeCmd *goredis.StringSliceCmd
	_, err := r.client.TxPipelined(ctx, func(pipe goredis.Pipeliner) error {
		rangeCmd = pipe.LRange(ctx, key, 0, int64(n-1))
		pipe.LTrim(ctx, key, int64(n), -1)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return rangeCmd.Val(), nil
}

func smsQueueKey(priority, client string) string {
	if priority == entity.PriorityHigh {
		return smsHighQueuePrefix + client
	}
	return smsQueuePrefix + client
}

func msgKey(client, id string) string {
	return msgPrefix + NormalizeClient(client) + ":" + id
}

func NormalizeClient(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "default"
	}
	return value
}
