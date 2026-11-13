package app

import (
	"context"

	"github.com/bruli-lab/myCalendar/internal/domain/auth"
	"github.com/bruli-lab/myCalendar/internal/domain/calendar"
)

//go:generate go tool moq -out repositories_mock_test.go -pkg app_test . AuthenticationRepository TokenRepository EventsRepository TasksRepository

type AuthenticationRepository interface {
	Read(ctx context.Context) (*auth.Auth, error)
	Save(ctx context.Context, auth *auth.Auth) error
}

type TokenRepository interface {
	Refresh(ctx context.Context, refreshToken string) (*auth.Auth, error)
	Exchange(ctx context.Context, code string) (*auth.Auth, error)
}

type EventsRepository interface {
	FindByPeriod(ctx context.Context, period calendar.Period) ([]calendar.Event, error)
}

type TasksRepository interface {
	FindByPeriod(ctx context.Context, period calendar.Period) ([]calendar.Task, error)
}
