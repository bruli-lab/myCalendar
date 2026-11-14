package config

import (
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/robfig/cron/v3"
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
	GoogleAPIURL   string        `env:"GOOGLE_API_URL"`
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
	if err := validateSchedules(cfg.DailySchedule, cfg.WeeklySchedule); err != nil {
		return nil, err
	}
	if cfg.GoogleAPIURL, err = normalizeGoogleAPIURL(cfg.GoogleAPIURL); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func validateSchedules(daily, weekly string) error {
	for name, schedule := range map[string]string{"DAILY_SCHEDULE": daily, "WEEKLY_SCHEDULE": weekly} {
		if _, err := cron.ParseStandard(schedule); err != nil {
			return fmt.Errorf("invalid %s %q: %w", name, schedule, err)
		}
	}
	return nil
}

func normalizeGoogleAPIURL(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid GOOGLE_API_URL %q: %w", raw, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" || parsed.Host == "" {
		return "", fmt.Errorf("invalid GOOGLE_API_URL %q: it must be an absolute http or https URL", raw)
	}
	return strings.TrimSuffix(raw, "/"), nil
}
