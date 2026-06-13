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

func (h *Handler) GetBatch(c *gin.Context) {
	batchID := c.Param("batch_id")

	batch, err := h.service.GetBatch(c.Request.Context(), batchID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(batch) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "batch not found"})
		return
	}

	c.JSON(http.StatusOK, batch)
}

func (h *Handler) Queues(c *gin.Context) {
	sizes, err := h.service.QueueSizes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sizes)
}
