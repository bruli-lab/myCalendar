package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	httpinfra "github.com/bruli-lab/myCalendar/internal/infra/http"
	"github.com/stretchr/testify/require"
)

func TestCallbackServer(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		expectedStatus int
		expectedCode   string
		expectedErr    bool
	}{
		{
			name:           "without state, then it rejects the request without notifying anything",
			query:          "code=auth-code",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "with a wrong state, then it rejects the request without notifying anything",
			query:          "state=forged&code=auth-code",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "with a valid state and an error, then it notifies the error",
			query:          "state=expected-state&error=access_denied",
			expectedStatus: http.StatusBadRequest,
			expectedErr:    true,
		},
		{
			name:           "with a valid state and without code, then it notifies an error",
			query:          "state=expected-state",
			expectedStatus: http.StatusBadRequest,
			expectedErr:    true,
		},
		{
			name:           "with a valid state and a code, then it notifies the code",
			query:          "state=expected-state&code=auth-code",
			expectedStatus: http.StatusOK,
			expectedCode:   "auth-code",
		},
	}
	for _, tt := range tests {
		t.Run(`Given a callback server,
		when the callback is requested `+tt.name, func(t *testing.T) {
			t.Parallel()
			codeCh := make(chan string, 1)
			errCh := make(chan error, 1)
			srv := httpinfra.NewCallbackServer(":0", "expected-state", codeCh, errCh)
			rec := httptest.NewRecorder()

			srv.Handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/callback?"+tt.query, http.NoBody))

			require.Equal(t, tt.expectedStatus, rec.Code)
			require.Len(t, errCh, boolToInt(tt.expectedErr))
			if tt.expectedCode == "" {
				require.Empty(t, codeCh)
				return
			}
			require.Equal(t, tt.expectedCode, <-codeCh)
		})
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
