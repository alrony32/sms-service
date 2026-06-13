package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/sms-service/internal/config"
	"github.com/sms-service/internal/driver"
	"github.com/sms-service/internal/handler"
	"github.com/sms-service/internal/middleware"
	"github.com/sms-service/internal/queue"
	"github.com/sms-service/internal/ratelimit"
	"github.com/sms-service/internal/router"
	"github.com/sms-service/internal/service"
	"github.com/sms-service/internal/validator"
	"github.com/sms-service/internal/worker"
	"github.com/sms-service/pkg/logger"
	redisclient "github.com/sms-service/pkg/redis"
)

func main() {
	logger.Init()
	validator.Init()

	cfg := config.Load()

	rdb := redisclient.Client(cfg)
	if err := rdb.Ping(redisclient.Ctx).Err(); err != nil {
		log.Fatal(err)
	}

	if err := rdb.ConfigSet(redisclient.Ctx, "appendonly", "yes").Err(); err != nil {
		logger.Error("could not enable AOF persistence (set 'appendonly yes' in redis.conf)", err.Error())
	}

	repo := queue.NewRedisRepository(rdb)
	smsService := service.NewSMSService(repo)
	smsHandler := handler.NewSMSHandler(smsService)

	limiter := ratelimit.New(rdb)
	providerLimiter := ratelimit.NewSMSLimiter(limiter, cfg.Provider.RatePerMin)
	webhookLimiter := ratelimit.NewWebhookLimiter(limiter, cfg.Webhook.RatePerMin)
	drv := driver.NewFromConfig(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dispatcher := worker.NewDispatcher(repo, drv, providerLimiter, cfg)
	webhookWorker := worker.NewWebhookWorker(repo, webhookLimiter, cfg)
	go dispatcher.Run(ctx)
	go webhookWorker.Run(ctx)

	r := gin.Default()
	r.Use(middleware.APIKeyMiddleware(cfg.Auth.APIKey))
	router.SetupRoutes(r, router.Dependencies{SMSHandler: smsHandler})

	srv := &http.Server{Addr: ":" + cfg.App.Port, Handler: r}

	go func() {
		log.Println("server running on :" + cfg.App.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
