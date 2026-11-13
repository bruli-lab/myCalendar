//go:build functional

package harness

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	AuthorizationCode   = "functional-code"
	InitialRefreshToken = "refresh-from-code"
)

type FakeGoogle struct {
	server       *httptest.Server
	mu           sync.Mutex
	revoked      bool
	expiresIn    int
	refreshCalls int
}

func NewFakeGoogle(t *testing.T) *FakeGoogle {
	t.Helper()
	f := &FakeGoogle{expiresIn: 3600}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /token", f.token(t))
	mux.HandleFunc("GET /calendar/v3/calendars/primary/events", f.authorized(t, map[string]any{
		"items": []map[string]any{
			{"id": "1", "summary": "Bank holiday", "htmlLink": "https://calendar.test/1", "start": map[string]string{"date": "2026-11-13"}},
			{"id": "2", "summary": "Team meeting", "htmlLink": "https://calendar.test/2", "start": map[string]string{"dateTime": "2026-11-13T08:30:00Z"}},
		},
	}))
	mux.HandleFunc("GET /tasks/v1/users/@me/lists", f.authorized(t, map[string]any{
		"items": []map[string]any{{"id": "default"}},
	}))
	mux.HandleFunc("GET /tasks/v1/lists/default/tasks", f.authorized(t, map[string]any{
		"items": []map[string]any{
			{"id": "1", "title": "Buy milk", "due": "2026-11-13T00:00:00.000Z", "webViewLink": "https://tasks.test/1"},
		},
	}))
	f.server = httptest.NewServer(mux)
	t.Cleanup(f.server.Close)
	return f
}

func (f *FakeGoogle) URL() string {
	return f.server.URL
}

func (f *FakeGoogle) RevokeRefreshToken() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.revoked = true
}

func (f *FakeGoogle) ExpireAccessTokensImmediately() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.expiresIn = 1
}

func (f *FakeGoogle) RefreshCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.refreshCalls
}

func (f *FakeGoogle) token(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if !assert.NoError(t, r.ParseForm()) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		switch r.PostForm.Get("grant_type") {
		case "authorization_code":
			if r.PostForm.Get("code") != AuthorizationCode {
				writeTokenError(t, w, "invalid_grant")
				return
			}
			writeJSON(t, w, http.StatusOK, f.tokenResponse("access-from-code", InitialRefreshToken))
		case "refresh_token":
			f.refreshCalls++
			if f.revoked {
				writeTokenError(t, w, "invalid_grant")
				return
			}
			writeJSON(t, w, http.StatusOK, f.tokenResponse("access-from-refresh", ""))
		default:
			writeTokenError(t, w, "unsupported_grant_type")
		}
	}
}

func (f *FakeGoogle) tokenResponse(accessToken, refreshToken string) map[string]any {
	body := map[string]any{"access_token": accessToken, "token_type": "Bearer", "expires_in": f.expiresIn}
	if refreshToken != "" {
		body["refresh_token"] = refreshToken
	}
	return body
}

func (f *FakeGoogle) authorized(t *testing.T, body any) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer access-") {
			writeJSON(t, w, http.StatusUnauthorized, map[string]any{"error": map[string]any{"code": 401, "message": "unauthorized"}})
			return
		}
		writeJSON(t, w, http.StatusOK, body)
	}
}

func writeTokenError(t *testing.T, w http.ResponseWriter, code string) {
	t.Helper()
	writeJSON(t, w, http.StatusBadRequest, map[string]string{"error": code, "error_description": code})
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, body any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	assert.NoError(t, json.NewEncoder(w).Encode(body))
}
