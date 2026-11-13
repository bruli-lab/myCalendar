package config_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/bruli-lab/myCalendar/internal/config"
	"github.com/stretchr/testify/require"
)

func setRequiredEnv(t *testing.T) {
	t.Helper()
	for key, value := range map[string]string{
		"CLIENT_ID":     "client-id",
		"CLIENT_SECRET": "client-secret",
		"CALLBACK_HOST": ":8081",
		"CALLBACK_URL":  "http://localhost:8081/callback",
		"TOKENS_FILE":   "tokens.json",
		"SERVER_HOST":   ":8080",
		"LOG_LEVEL":     "debug",
		"NTFY_URL":      "http://localhost:8085",
		"NTFY_TOPIC":    "mycalendar",
		"NTFY_USER":     "user",
		"NTFY_PASSWORD": "password",
	} {
		t.Setenv(key, value)
	}
}

func TestNew(t *testing.T) {
	t.Run(`Given only the required environment variables,
	when New function is called,
	then it returns a config with the default schedule, calendar and timezone`, func(t *testing.T) {
		setRequiredEnv(t)

		conf, err := config.New()

		require.NoError(t, err)
		require.Equal(t, slog.LevelDebug, conf.LogLevel)
		require.Equal(t, "primary", conf.CalendarID)
		require.Equal(t, "0 8 * * 0,2-6", conf.DailySchedule)
		require.Equal(t, "0 8 * * 1", conf.WeeklySchedule)
		require.Equal(t, 2*time.Minute, conf.JobTimeout)
		require.Equal(t, "Europe/Madrid", conf.Location.String())
	})

	t.Run(`Given an unknown log level,
	when New function is called,
	then it returns an error`, func(t *testing.T) {
		setRequiredEnv(t)
		t.Setenv("LOG_LEVEL", "verbose")

		_, err := config.New()

		require.Error(t, err)
	})

	t.Run(`Given an unknown timezone,
	when New function is called,
	then it returns an error`, func(t *testing.T) {
		setRequiredEnv(t)
		t.Setenv("TIMEZONE", "Mars/Olympus")

		_, err := config.New()

		require.Error(t, err)
	})

	t.Run(`Given an empty required variable,
	when New function is called,
	then it returns an error`, func(t *testing.T) {
		setRequiredEnv(t)
		t.Setenv("CLIENT_ID", "")

		_, err := config.New()

		require.Error(t, err)
	})
}
