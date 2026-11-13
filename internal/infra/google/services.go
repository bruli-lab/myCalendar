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

func NewOAuthConfig(clientID, clientSecret, redirectURL, apiURL string) *oauth2.Config {
	endpoint := google.Endpoint
	if apiURL != "" {
		endpoint = oauth2.Endpoint{AuthURL: apiURL + "/o/oauth2/auth", TokenURL: apiURL + "/token"}
	}
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			gcalendar.CalendarReadonlyScope,
			gtasks.TasksReadonlyScope,
		},
		Endpoint: endpoint,
	}
}

func NewCalendarService(ctx context.Context, ts oauth2.TokenSource, apiURL string) (*gcalendar.Service, error) {
	svc, err := gcalendar.NewService(ctx, clientOptions(ts, apiURL, "/calendar/v3/")...)
	if err != nil {
		return nil, fmt.Errorf("failed to create calendar client: %w", err)
	}
	return svc, nil
}

func NewTasksService(ctx context.Context, ts oauth2.TokenSource, apiURL string) (*gtasks.Service, error) {
	svc, err := gtasks.NewService(ctx, clientOptions(ts, apiURL, "/")...)
	if err != nil {
		return nil, fmt.Errorf("failed to create tasks client: %w", err)
	}
	return svc, nil
}

func clientOptions(ts oauth2.TokenSource, apiURL, basePath string) []option.ClientOption {
	opts := []option.ClientOption{option.WithTokenSource(ts)}
	if apiURL != "" {
		opts = append(opts, option.WithEndpoint(apiURL+basePath))
	}
	return opts
}
