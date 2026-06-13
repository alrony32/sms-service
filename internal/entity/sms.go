package entity

import "time"

const (
	StatusQueued    = "queued"
	StatusSent      = "sent"
	StatusFailed    = "failed"
	StatusDelivered = "delivered"
)

const (
	PriorityNormal = "normal"
	PriorityHigh   = "high"
)

type SMS struct {
	ID         string    `json:"id"`
	BatchID    string    `json:"batch_id"`
	From       string    `json:"from"`
	To         string    `json:"to"`
	Message    string    `json:"message"`
	Client     string    `json:"client"`
	Status     string    `json:"status,omitempty"`
	Priority   string    `json:"priority,omitempty"`
	WebhookURL string    `json:"webhook_url"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type WebhookEvent struct {
	ID         string `json:"id"`
	BatchID    string `json:"batch_id"`
	Client     string `json:"client"`
	To         string `json:"to"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
	WebhookURL string `json:"webhook_url"`
	Attempts   int    `json:"attempts"`
}

type WebhookPayload struct {
	ID      string `json:"id"`
	BatchID string `json:"batch_id"`
	Client  string `json:"client"`
	To      string `json:"to"`
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
}

func (e WebhookEvent) Payload() WebhookPayload {
	return WebhookPayload{
		ID:      e.ID,
		BatchID: e.BatchID,
		Client:  e.Client,
		To:      e.To,
		Status:  e.Status,
		Error:   e.Error,
	}
}
