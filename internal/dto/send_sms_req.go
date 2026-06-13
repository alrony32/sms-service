package dto

type SingleSMSRequest struct {
	Client     string `json:"client" binding:"required,len=3"`
	From       string `json:"from" binding:"required"`
	WebhookURL string `json:"webhook_url" binding:"required,url"`
	ID         string `json:"id" binding:"required"`
	To         string `json:"to" binding:"required,e164_bd"`
	Message    string `json:"message" binding:"required"`
}

type BulkSMSRequest struct {
	Client     string        `json:"client" binding:"required,len=3"`
	From       string        `json:"from" binding:"required"`
	WebhookURL string        `json:"webhook_url" binding:"required,url"`
	Messages   []BulkSMSItem `json:"messages" binding:"required,min=1,dive"`
}

type BulkSMSItem struct {
	ID      string `json:"id" binding:"required"`
	To      string `json:"to" binding:"required,e164_bd"`
	Message string `json:"message" binding:"required"`
}
