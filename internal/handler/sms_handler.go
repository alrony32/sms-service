package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sms-service/internal/dto"
	"github.com/sms-service/internal/service"
	"github.com/sms-service/pkg/validation"
)

type Handler struct {
	service service.Service
}

func NewSMSHandler(svc service.Service) *Handler {
	return &Handler{service: svc}
}

func (h *Handler) SendSingle(c *gin.Context) {
	var req dto.SingleSMSRequest
	if !validation.ValidateJSON(c, &req) {
		return
	}

	batchID, err := h.service.SendSingle(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"status":   "queued",
		"batch_id": batchID,
		"count":    1,
	})
}

func (h *Handler) SendBulk(c *gin.Context) {
	var req dto.BulkSMSRequest
	if !validation.ValidateJSON(c, &req) {
		return
	}

	batchID, err := h.service.SendBulk(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"status":   "queued",
		"batch_id": batchID,
		"count":    len(req.Messages),
	})
}
