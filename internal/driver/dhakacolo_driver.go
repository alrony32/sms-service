package driver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sms-service/internal/config"
	"github.com/sms-service/internal/entity"
	"github.com/sms-service/pkg/logger"
)

type DhakaColoDriver struct {
	url    string
	apiKey string
	sender string
	client *http.Client
}

func NewDhakaColoDriver(cfg config.ProviderConfig) *DhakaColoDriver {
	return &DhakaColoDriver{
		url:    cfg.URL,
		apiKey: cfg.APIKey,
		sender: cfg.Sender,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

type dhakaColoRecipient struct {
	ID      string `json:"id"`
	To      string `json:"to"`
	Message string `json:"message"`
}

type dhakaColoRequest struct {
	Sender     string               `json:"sender"`
	Recipients []dhakaColoRecipient `json:"recipients"`
}

func (d *DhakaColoDriver) SendBatch(ctx context.Context, msgs []entity.SMS) ([]Result, error) {
	if len(msgs) == 0 {
		return nil, nil
	}

	reqBody := dhakaColoRequest{Sender: d.sender}
	for _, m := range msgs {
		sender := d.sender
		if sender == "" {
			sender = m.From
		}
		reqBody.Sender = sender
		reqBody.Recipients = append(reqBody.Recipients, dhakaColoRecipient{
			ID:      m.ID,
			To:      m.To,
			Message: m.Message,
		})
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if d.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+d.apiKey)
	}

	resp, err := d.client.Do(req)
	if err != nil {

		return allFailed(msgs, err.Error()), err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errMsg := fmt.Sprintf("dhakacolo http %d: %s", resp.StatusCode, truncate(string(body), 200))
		logger.Error("DHAKACOLO DRIVER - batch rejected", errMsg)
		return allFailed(msgs, errMsg), fmt.Errorf("%s", errMsg)
	}

	logger.Info("DHAKACOLO DRIVER - batch accepted", "count", len(msgs))
	results := make([]Result, 0, len(msgs))
	for _, m := range msgs {
		results = append(results, Result{ID: m.ID, Status: entity.StatusSent})
	}
	return results, nil
}

func allFailed(msgs []entity.SMS, reason string) []Result {
	results := make([]Result, 0, len(msgs))
	for _, m := range msgs {
		results = append(results, Result{ID: m.ID, Status: entity.StatusFailed, Error: reason})
	}
	return results
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
