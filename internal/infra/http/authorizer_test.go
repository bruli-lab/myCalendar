//go:build infra

package http_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"github.com/bruli-lab/myCalendar/internal/app"
	httpinfra "github.com/bruli-lab/myCalendar/internal/infra/http"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func freeHost(t *testing.T) string {
	t.Helper()
	ln, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	host := ln.Addr().String()
	require.NoError(t, ln.Close())
	return host
}

func oauthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:    "client-id",
		RedirectURL: "http://localhost/callback",
		Endpoint:    oauth2.Endpoint{AuthURL: "https://accounts.example.com/auth"},
	}
}

func callCallback(t *testing.T, host, query string) int {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+host+"/callback?"+query, http.NoBody)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	return resp.StatusCode
}

func publishedState(t *testing.T, published <-chan string) string {
	t.Helper()
	select {
	case body := <-published:
		authURL, err := url.Parse(body[len("Click -> "):])
		require.NoError(t, err)
		require.Equal(t, "offline", authURL.Query().Get("access_type"))
		state := authURL.Query().Get("state")
		require.NotEmpty(t, state)
		return state
	case <-time.After(5 * time.Second):
		t.Fatal("authorization url not published")
		return ""
	}
}

func TestAuthorizer_Authorize(t *testing.T) {
	t.Run(`Given an Authorizer,
	when Authorize method is called and the callback first receives a forged state and then the published state with a code,
	then it ignores the forged request, creates the tokens with the code and returns nil`, func(t *testing.T) {
		t.Parallel()
		host := freeHost(t)
		published := make(chan string, 1)
		publisher := &MessagePublisherMock{PublishFunc: func(_ context.Context, _, body string) error {
			published <- body
			return nil
		}}
		commands := &cqs.CommandHandlerMock{HandleFunc: func(_ context.Context, _ cqs.Command) ([]event.Event, error) {
			return nil, nil
		}}
		authorizer := httpinfra.NewAuthorizer(oauthConfig(), publisher, commands, host)
		result := make(chan error, 1)

		go func() { result <- authorizer.Authorize(t.Context()) }()
		state := publishedState(t, published)

		require.Equal(t, http.StatusBadRequest, callCallback(t, host, "state=forged&code=evil"))
		require.Equal(t, http.StatusOK, callCallback(t, host, "state="+url.QueryEscape(state)+"&code=auth-code"))
		require.NoError(t, <-result)
		require.Len(t, commands.HandleCalls(), 1)
		require.Equal(t, app.CreateTokensCmd{Code: "auth-code"}, commands.HandleCalls()[0].Cmd)
	})

	t.Run(`Given an Authorizer,
	when Authorize method is called and the user denies the access,
	then it returns an error without creating tokens`, func(t *testing.T) {
		t.Parallel()
		host := freeHost(t)
		published := make(chan string, 1)
		publisher := &MessagePublisherMock{PublishFunc: func(_ context.Context, _, body string) error {
			published <- body
			return nil
		}}
		commands := &cqs.CommandHandlerMock{}
		authorizer := httpinfra.NewAuthorizer(oauthConfig(), publisher, commands, host)
		result := make(chan error, 1)

		go func() { result <- authorizer.Authorize(t.Context()) }()
		state := publishedState(t, published)

		require.Equal(t, http.StatusBadRequest, callCallback(t, host, "state="+url.QueryEscape(state)+"&error=access_denied"))
		require.Error(t, <-result)
		require.Empty(t, commands.HandleCalls())
	})

	t.Run(`Given an Authorizer with an authorization in progress,
	when Authorize method is called again,
	then it returns an authorization in progress error`, func(t *testing.T) {
		t.Parallel()
		host := freeHost(t)
		published := make(chan string, 1)
		publisher := &MessagePublisherMock{PublishFunc: func(_ context.Context, _, body string) error {
			published <- body
			return nil
		}}
		authorizer := httpinfra.NewAuthorizer(oauthConfig(), publisher, &cqs.CommandHandlerMock{}, host)
		ctx, cancel := context.WithCancel(t.Context())
		result := make(chan error, 1)

		go func() { result <- authorizer.Authorize(ctx) }()
		publishedState(t, published)

		require.ErrorIs(t, authorizer.Authorize(t.Context()), httpinfra.ErrAuthorizationInProgress)
		cancel()
		require.ErrorIs(t, <-result, context.Canceled)
	})

	t.Run(`Given an Authorizer whose publisher fails,
	when Authorize method is called,
	then it returns an error`, func(t *testing.T) {
		t.Parallel()
		publisher := &MessagePublisherMock{PublishFunc: func(_ context.Context, _, _ string) error {
			return errors.New("ntfy down")
		}}
		authorizer := httpinfra.NewAuthorizer(oauthConfig(), publisher, &cqs.CommandHandlerMock{}, freeHost(t))

		require.Error(t, authorizer.Authorize(t.Context()))
	})
}
