package config

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	ClientID       string        `env:"CLIENT_ID,required,notEmpty"`
	ClientSecret   string        `env:"CLIENT_SECRET,required,notEmpty"`
	CallbackHost   string        `env:"CALLBACK_HOST,required,notEmpty"`
	CallbackURL    string        `env:"CALLBACK_URL,required,notEmpty"`
	TokensFile     string        `env:"TOKENS_FILE,required,notEmpty"`
	ServerHost     string        `env:"SERVER_HOST,required,notEmpty"`
	LogLevel       slog.Level    `env:"LOG_LEVEL,required,notEmpty"`
	NtfyURL        string        `env:"NTFY_URL,required,notEmpty"`
	NtfyTopic      string        `env:"NTFY_TOPIC,required,notEmpty"`
	NtfyUser       string        `env:"NTFY_USER,required,notEmpty"`
	NtfyPassword   string        `env:"NTFY_PASSWORD,required,notEmpty"`
	CalendarID     string        `env:"CALENDAR_ID" envDefault:"primary"`
	Timezone       string        `env:"TIMEZONE" envDefault:"Europe/Madrid"`
	DailySchedule  string        `env:"DAILY_SCHEDULE" envDefault:"0 8 * * 0,2-6"`
	WeeklySchedule string        `env:"WEEKLY_SCHEDULE" envDefault:"0 8 * * 1"`
	JobTimeout     time.Duration `env:"JOB_TIMEOUT" envDefault:"2m"`
	Location       *time.Location
}

func New() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return nil, fmt.Errorf("invalid TIMEZONE %q: %w", cfg.Timezone, err)
	}
	cfg.Location = loc
	return &cfg, nil
}
