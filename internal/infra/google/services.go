package google

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	gcalendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
	gtasks "google.golang.org/api/tasks/v1"
)

func NewOAuthConfig(clientID, clientSecret, redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			gcalendar.CalendarReadonlyScope,
			gtasks.TasksReadonlyScope,
		},
		Endpoint: google.Endpoint,
	}
}

func NewCalendarService(ctx context.Context, ts oauth2.TokenSource) (*gcalendar.Service, error) {
	svc, err := gcalendar.NewService(ctx, option.WithTokenSource(ts))
	if err != nil {
		return nil, fmt.Errorf("failed to create calendar client: %w", err)
	}
	return svc, nil
}

func NewTasksService(ctx context.Context, ts oauth2.TokenSource) (*gtasks.Service, error) {
	svc, err := gtasks.NewService(ctx, option.WithTokenSource(ts))
	if err != nil {
		return nil, fmt.Errorf("failed to create tasks client: %w", err)
	}
	return svc, nil
}
