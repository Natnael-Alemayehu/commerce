package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	Environment string `env:"ENV" envDefault:"development"`
	Port        string `env:"PORT" envDefault:"8080"`

	DatabaseURL string `env:"DATABASE_URL,required"`

	JWTSecret       string        `env:"JWT_SECRET,required"`
	JWTAccessTTL    time.Duration `env:"JWT_ACCESS_TTL" envDefault:"15m"`
	JWTRefreshTTL   time.Duration `env:"JWT_REFRESH_TTL" envDefault:"168h"` // 7 days

	Argon2Time    uint32 `env:"ARGON2_TIME" envDefault:"3"`
	Argon2Memory  uint32 `env:"ARGON2_MEMORY" envDefault:"65536"` // 64 MB
	Argon2Threads uint8  `env:"ARGON2_THREADS" envDefault:"4"`
	Argon2KeyLen  uint32 `env:"ARGON2_KEYLEN" envDefault:"32"`
	Argon2SaltLen uint32 `env:"ARGON2_SALTLEN" envDefault:"16"`
}

// Load parses environment variables into Config.
func Load() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return &cfg, nil
}
