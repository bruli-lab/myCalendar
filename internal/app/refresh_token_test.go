package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/myCalendar/internal/app"
	"github.com/bruli-lab/myCalendar/internal/domain/auth"
	"github.com/bruli-lab/myCalendar/internal/fixtures"
	"github.com/stretchr/testify/require"
)

func TestRefreshToken_Handle(t *testing.T) {
	errTest := errors.New("test error")
	storedRefreshToken := "stored-refresh-token"
	stored := fixtures.AuthBuilder{RefreshToken: &storedRefreshToken}.Build()
	newAccessToken := "new-access-token"
	refreshed := fixtures.AuthBuilder{AccessToken: &newAccessToken}.Build()
	tests := []struct {
		name                         string
		cmd                          cqs.Command
		readErr, refreshErr, saveErr error
		expectedErr                  error
		expectedRefreshes            int
		expectedSaves                int
	}{
		{
			name:        "with an invalid command, then it returns an invalid command error",
			cmd:         invalidCommand{},
			expectedErr: cqs.InvalidCommandError{},
		},
		{
			name:        "with a valid command and read method returns an error, then it returns same error",
			cmd:         app.RefreshTokenCmd{},
			readErr:     errTest,
			expectedErr: errTest,
		},
		{
			name:              "with a valid command and refresh method returns an error, then it returns same error",
			cmd:               app.RefreshTokenCmd{},
			refreshErr:        errTest,
			expectedErr:       errTest,
			expectedRefreshes: 1,
		},
		{
			name:              "with a valid command and save method returns an error, then it returns same error",
			cmd:               app.RefreshTokenCmd{},
			saveErr:           errTest,
			expectedErr:       errTest,
			expectedRefreshes: 1,
			expectedSaves:     1,
		},
		{
			name:              "with a valid command and all repositories succeed, then it refreshes the stored token and saves it",
			cmd:               app.RefreshTokenCmd{},
			expectedRefreshes: 1,
			expectedSaves:     1,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a RefreshToken command handler,
		when Handle method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			authRepo := &AuthenticationRepositoryMock{
				ReadFunc: func(_ context.Context) (*auth.Auth, error) {
					if tt.readErr != nil {
						return nil, tt.readErr
					}
					return stored, nil
				},
				SaveFunc: func(_ context.Context, _ *auth.Auth) error {
					return tt.saveErr
				},
			}
			tokenRepo := &TokenRepositoryMock{
				RefreshFunc: func(_ context.Context, _ string) (*auth.Auth, error) {
					if tt.refreshErr != nil {
						return nil, tt.refreshErr
					}
					return refreshed, nil
				},
			}

			events, err := app.NewRefreshToken(authRepo, tokenRepo, tracer()).Handle(t.Context(), tt.cmd)

			require.Empty(t, events)
			if tt.expectedErr != nil {
				require.IsType(t, tt.expectedErr, err)
			} else {
				require.NoError(t, err)
			}
			require.Len(t, tokenRepo.RefreshCalls(), tt.expectedRefreshes)
			if tt.expectedRefreshes > 0 {
				require.Equal(t, storedRefreshToken, tokenRepo.RefreshCalls()[0].RefreshToken)
			}
			require.Len(t, authRepo.SaveCalls(), tt.expectedSaves)
			if tt.expectedSaves > 0 {
				require.Equal(t, refreshed, authRepo.SaveCalls()[0].AuthMoqParam)
			}
		})
	}
}
