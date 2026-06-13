package config

type Config struct {
	App       AppConfig
	Redis     RedisConfig
	Auth      AuthConfig
	Provider  ProviderConfig
	Webhook   WebhookConfig
	Scheduler SchedulerConfig
}

type AppConfig struct {
	Port string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type AuthConfig struct {
	APIKey string
}

type ProviderConfig struct {
	Driver     string
	URL        string
	APIKey     string
	Sender     string
	BatchSize  int
	RatePerMin int
}

type WebhookConfig struct {
	RatePerMin int
	TimeoutSec int
	MaxRetries int
}

type SchedulerConfig struct {
	IntervalMs int
}
