package auth_test

import (
	"testing"
	"time"

	"github.com/bruli-lab/myCalendar/internal/domain/auth"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	expiry := time.Date(2026, time.November, 13, 9, 0, 0, 0, time.UTC)
	tests := []struct {
		name                                 string
		accessToken, refreshToken, tokenType string
		expiry                               time.Time
		expectedErr                          error
	}{
		{
			name:         "with an empty access token, then it returns an empty access token error",
			refreshToken: "refresh",
			tokenType:    "Bearer",
			expiry:       expiry,
			expectedErr:  auth.ErrEmptyAccessToken,
		},
		{
			name:        "with an empty refresh token, then it returns an empty refresh token error",
			accessToken: "access",
			tokenType:   "Bearer",
			expiry:      expiry,
			expectedErr: auth.ErrEmptyRefreshToken,
		},
		{
			name:         "with an empty token type, then it returns an empty token type error",
			accessToken:  "access",
			refreshToken: "refresh",
			expiry:       expiry,
			expectedErr:  auth.ErrEmptyTokenType,
		},
		{
			name:         "with an empty expiry, then it returns an empty expiry error",
			accessToken:  "access",
			refreshToken: "refresh",
			tokenType:    "Bearer",
			expectedErr:  auth.ErrEmptyExpiry,
		},
		{
			name:         "with all the values, then it returns a valid auth",
			accessToken:  "access",
			refreshToken: "refresh",
			tokenType:    "Bearer",
			expiry:       expiry,
		},
	}
	for _, tt := range tests {
		t.Run(`Given an auth constructor,
		when New function is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			a, err := auth.New(tt.accessToken, tt.refreshToken, tt.expiry, tt.tokenType)
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				require.Nil(t, a)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.accessToken, a.AccessToken())
			require.Equal(t, tt.refreshToken, a.RefreshToken())
			require.Equal(t, tt.tokenType, a.TokenType())
			require.Equal(t, tt.expiry, a.Expiry())
		})
	}
}
