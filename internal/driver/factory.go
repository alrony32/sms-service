package driver

import (
	"github.com/sms-service/internal/config"
)

func NewFromConfig(cfg *config.Config) Driver {
	switch cfg.Provider.Driver {
	case "dhakacolo":
		return NewDhakaColoDriver(cfg.Provider)
	default:
		return NewLogDriver()
	}
}
