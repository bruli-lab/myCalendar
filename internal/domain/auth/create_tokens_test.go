package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bruli-lab/myCalendar/internal/domain/auth"
	"github.com/bruli-lab/myCalendar/internal/fixtures"
	"github.com/stretchr/testify/require"
)

func TestCreateTokens_Create(t *testing.T) {
	errTest := errors.New("test error")
	exchanged := fixtures.AuthBuilder{}.Build()
	tests := []struct {
		name                 string
		exchangeErr, saveErr error
		expectedErr          error
		expectedSaves        int
	}{
		{
			name:        "and exchange method returns an error, then it returns same error",
			exchangeErr: errTest,
			expectedErr: errTest,
		},
		{
			name:          "and save method returns an error, then it returns same error",
			saveErr:       errTest,
			expectedErr:   errTest,
			expectedSaves: 1,
		},
		{
			name:          "and all repositories succeed, then it saves the exchanged auth",
			expectedSaves: 1,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a CreateTokens service,
		when Create method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			tokenRepo := &TokenRepositoryMock{
				ExchangeFunc: func(_ context.Context, _ string) (*auth.Auth, error) {
					if tt.exchangeErr != nil {
						return nil, tt.exchangeErr
					}
					return exchanged, nil
				},
			}
			authRepo := &AuthenticationRepositoryMock{
				SaveFunc: func(_ context.Context, _ *auth.Auth) error {
					return tt.saveErr
				},
			}

			err := auth.NewCreateTokens(tokenRepo, authRepo).Create(t.Context(), "auth-code")

			require.ErrorIs(t, err, tt.expectedErr)
			require.Equal(t, "auth-code", tokenRepo.ExchangeCalls()[0].Code)
			require.Len(t, authRepo.SaveCalls(), tt.expectedSaves)
			if tt.expectedSaves > 0 {
				require.Equal(t, exchanged, authRepo.SaveCalls()[0].AuthMoqParam)
			}
		})
	}
}
