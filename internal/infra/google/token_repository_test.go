//go:build infra

package google_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bruli-lab/myCalendar/internal/domain/auth"
	"github.com/bruli-lab/myCalendar/internal/infra/google"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestTokenRepository_Refresh(t *testing.T) {
	tests := []struct {
		name               string
		status             int
		body               string
		expectedRefreshErr bool
	}{
		{
			name:               "and Google answers invalid_grant, then it returns a refresh error",
			status:             http.StatusBadRequest,
			body:               `{"error":"invalid_grant","error_description":"Token has been expired or revoked."}`,
			expectedRefreshErr: true,
		},
		{
			name:   "and Google answers another error, then it returns an error that is not a refresh error",
			status: http.StatusInternalServerError,
			body:   `{"error":"server_error"}`,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a TokenRepository,
		when Refresh method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()
			repo := google.NewTokenRepository(&oauth2.Config{ClientID: "id", Endpoint: oauth2.Endpoint{TokenURL: srv.URL}})

			_, err := repo.Refresh(t.Context(), "revoked-refresh-token")

			require.Error(t, err)
			if tt.expectedRefreshErr {
				require.ErrorAs(t, err, &auth.RefreshError{})
				return
			}
			require.NotErrorAs(t, err, &auth.RefreshError{})
		})
	}
}
