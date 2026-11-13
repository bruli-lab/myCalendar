//go:build infra

package google_test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bruli-lab/myCalendar/internal/domain/auth"
	"github.com/bruli-lab/myCalendar/internal/fixtures"
	"github.com/bruli-lab/myCalendar/internal/infra/disk"
	"github.com/bruli-lab/myCalendar/internal/infra/google"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func tokenServer(t *testing.T, calls *atomic.Int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		assert.NoError(t, r.ParseForm())
		assert.Equal(t, "refresh_token", r.PostForm.Get("grant_type"))
		assert.Equal(t, "stored-refresh-token", r.PostForm.Get("refresh_token"))
		writeJSON(t, w, map[string]any{"access_token": "refreshed-access-token", "token_type": "Bearer", "expires_in": 3600})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestPersistentTokenSource_Token(t *testing.T) {
	refreshToken := "stored-refresh-token"
	tests := []struct {
		name                string
		expiry              time.Time
		expectedAccessToken string
		expectedCalls       int32
	}{
		{
			name:                "with a stored token that is still valid, then it returns it without refreshing",
			expiry:              time.Now().Add(time.Hour),
			expectedAccessToken: "access-token",
		},
		{
			name:                "with an expired stored token, then it refreshes it, keeps the refresh token and saves it",
			expiry:              time.Now().Add(-time.Hour),
			expectedAccessToken: "refreshed-access-token",
			expectedCalls:       1,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a PersistentTokenSource,
		when Token method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			var calls atomic.Int32
			srv := tokenServer(t, &calls)
			repo := disk.NewAuthenticationRepository(filepath.Join(t.TempDir(), "tokens.json"))
			require.NoError(t, repo.Save(t.Context(), fixtures.AuthBuilder{RefreshToken: &refreshToken, Expiry: &tt.expiry}.Build()))
			cfg := &oauth2.Config{ClientID: "id", ClientSecret: "secret", Endpoint: oauth2.Endpoint{TokenURL: srv.URL}}

			token, err := google.NewPersistentTokenSource(t.Context(), cfg, repo).Token()

			require.NoError(t, err)
			require.Equal(t, tt.expectedAccessToken, token.AccessToken)
			require.Equal(t, tt.expectedCalls, calls.Load())
			stored, err := repo.Read(t.Context())
			require.NoError(t, err)
			require.Equal(t, tt.expectedAccessToken, stored.AccessToken())
			require.Equal(t, refreshToken, stored.RefreshToken())
		})
	}

	t.Run(`Given a PersistentTokenSource with an expired token whose refresh token was revoked,
	when Token method is called,
	then it returns a refresh error`, func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"Token has been expired or revoked."}`))
		}))
		defer srv.Close()
		expired := time.Now().Add(-time.Hour)
		repo := disk.NewAuthenticationRepository(filepath.Join(t.TempDir(), "tokens.json"))
		require.NoError(t, repo.Save(t.Context(), fixtures.AuthBuilder{Expiry: &expired}.Build()))
		cfg := &oauth2.Config{ClientID: "id", ClientSecret: "secret", Endpoint: oauth2.Endpoint{TokenURL: srv.URL}}

		_, err := google.NewPersistentTokenSource(t.Context(), cfg, repo).Token()

		require.ErrorAs(t, err, &auth.RefreshError{})
	})

	t.Run(`Given a PersistentTokenSource without stored tokens,
	when Token method is called,
	then it returns an error`, func(t *testing.T) {
		t.Parallel()
		repo := disk.NewAuthenticationRepository(filepath.Join(t.TempDir(), "tokens.json"))

		_, err := google.NewPersistentTokenSource(t.Context(), &oauth2.Config{}, repo).Token()

		require.Error(t, err)
	})
}
