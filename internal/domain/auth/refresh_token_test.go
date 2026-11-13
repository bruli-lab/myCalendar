package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bruli-lab/myCalendar/internal/domain/auth"
	"github.com/bruli-lab/myCalendar/internal/fixtures"
	"github.com/stretchr/testify/require"
)

func TestRefreshToken_Refresh(t *testing.T) {
	errTest := errors.New("test error")
	storedRefreshToken := "stored-refresh-token"
	stored := fixtures.AuthBuilder{RefreshToken: &storedRefreshToken}.Build()
	newAccessToken := "new-access-token"
	refreshed := fixtures.AuthBuilder{AccessToken: &newAccessToken}.Build()
	tests := []struct {
		name                         string
		readErr, refreshErr, saveErr error
		expectedErr                  error
		expectedSaves                int
	}{
		{
			name:        "and read method returns an error, then it returns same error",
			readErr:     errTest,
			expectedErr: errTest,
		},
		{
			name:        "and refresh method returns an error, then it returns same error",
			refreshErr:  errTest,
			expectedErr: errTest,
		},
		{
			name:          "and save method returns an error, then it returns same error",
			saveErr:       errTest,
			expectedErr:   errTest,
			expectedSaves: 1,
		},
		{
			name:          "and all repositories succeed, then it saves the refreshed auth",
			expectedSaves: 1,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a RefreshToken service,
		when Refresh method is called `+tt.name, func(t *testing.T) {
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

			err := auth.NewRefreshToken(authRepo, tokenRepo).Refresh(t.Context())

			require.ErrorIs(t, err, tt.expectedErr)
			require.Len(t, authRepo.SaveCalls(), tt.expectedSaves)
			if tt.expectedSaves > 0 {
				require.Equal(t, refreshed, authRepo.SaveCalls()[0].AuthMoqParam)
			}
			if tt.readErr == nil {
				require.Equal(t, storedRefreshToken, tokenRepo.RefreshCalls()[0].RefreshToken)
			}
		})
	}
}
