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

func TestCreateTokens_Handle(t *testing.T) {
	errTest := errors.New("test error")
	exchanged := fixtures.AuthBuilder{}.Build()
	tests := []struct {
		name                 string
		cmd                  cqs.Command
		exchangeErr, saveErr error
		expectedErr          error
		expectedExchanges    int
		expectedSaves        int
	}{
		{
			name:        "with an invalid command, then it returns an invalid command error",
			cmd:         invalidCommand{},
			expectedErr: cqs.InvalidCommandError{},
		},
		{
			name:              "with a valid command and exchange method returns an error, then it returns same error",
			cmd:               app.CreateTokensCmd{Code: "auth-code"},
			exchangeErr:       errTest,
			expectedErr:       errTest,
			expectedExchanges: 1,
		},
		{
			name:              "with a valid command and save method returns an error, then it returns same error",
			cmd:               app.CreateTokensCmd{Code: "auth-code"},
			saveErr:           errTest,
			expectedErr:       errTest,
			expectedExchanges: 1,
			expectedSaves:     1,
		},
		{
			name:              "with a valid command and all repositories succeed, then it exchanges the code and saves the tokens",
			cmd:               app.CreateTokensCmd{Code: "auth-code"},
			expectedExchanges: 1,
			expectedSaves:     1,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a CreateTokens command handler,
		when Handle method is called `+tt.name, func(t *testing.T) {
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

			events, err := app.NewCreateTokens(tokenRepo, authRepo, tracer()).Handle(t.Context(), tt.cmd)

			require.Empty(t, events)
			if tt.expectedErr != nil {
				require.IsType(t, tt.expectedErr, err)
			} else {
				require.NoError(t, err)
			}
			require.Len(t, tokenRepo.ExchangeCalls(), tt.expectedExchanges)
			if tt.expectedExchanges > 0 {
				require.Equal(t, "auth-code", tokenRepo.ExchangeCalls()[0].Code)
			}
			require.Len(t, authRepo.SaveCalls(), tt.expectedSaves)
			if tt.expectedSaves > 0 {
				require.Equal(t, exchanged, authRepo.SaveCalls()[0].AuthMoqParam)
			}
		})
	}
}
