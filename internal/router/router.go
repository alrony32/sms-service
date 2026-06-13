package router

import (
	"github.com/gin-gonic/gin"
	"github.com/sms-service/internal/handler"
)

type Dependencies struct {
	SMSHandler *handler.Handler
}

func SetupRoutes(r *gin.Engine, deps Dependencies) {

	v1 := r.Group("/api/v1")

	smsGroup := v1.Group("/sms")
	{
		smsGroup.POST("/send", deps.SMSHandler.SendSingle)
		smsGroup.POST("/bulk", deps.SMSHandler.SendBulk)
	}
}
