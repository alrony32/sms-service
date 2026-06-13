package config

import (
	"log"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

func Load() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println(".env file not found, using system env")
	}

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	cfg := &Config{}

	cfg.App.Port = getDefault("APP_PORT", "8080")

	cfg.Redis.Host = getDefault("REDIS_HOST", "127.0.0.1")
	cfg.Redis.Port = getDefault("REDIS_PORT", "6379")
	cfg.Redis.Password = get("REDIS_PASSWORD")
	cfg.Redis.DB = getInt("REDIS_DB", 0)

	cfg.Auth.APIKey = get("API_KEY")

	cfg.Provider.Driver = getDefault("SMS_DRIVER", "log")
	cfg.Provider.URL = get("DHAKACOLO_URL")
	cfg.Provider.APIKey = get("DHAKACOLO_API_KEY")
	cfg.Provider.Sender = get("DHAKACOLO_SENDER")
	cfg.Provider.BatchSize = getInt("DHAKACOLO_BATCH_SIZE", 50)
	cfg.Provider.RatePerMin = getInt("DHAKACOLO_RATE_PER_MIN", 60)

	cfg.Webhook.RatePerMin = getInt("WEBHOOK_RATE_PER_MIN", 30)
	cfg.Webhook.TimeoutSec = getInt("WEBHOOK_TIMEOUT_SEC", 10)
	cfg.Webhook.MaxRetries = getInt("WEBHOOK_MAX_RETRIES", 5)

	cfg.Scheduler.IntervalMs = getInt("SCHEDULER_INTERVAL_MS", 1000)

	log.Println("config loaded successfully")

	return cfg
}

func get(key string) string {
	return viper.GetString(key)
}

func getDefault(key, fallback string) string {
	if v := viper.GetString(key); v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	if !viper.IsSet(key) || viper.GetString(key) == "" {
		return fallback
	}
	return viper.GetInt(key)
}
