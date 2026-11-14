//go:build infra

package google_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	gcalendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
	gtasks "google.golang.org/api/tasks/v1"
)

func tracer() trace.Tracer {
	return noop.NewTracerProvider().Tracer("test")
}

func writeJSON(t *testing.T, w http.ResponseWriter, body any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	assert.NoError(t, json.NewEncoder(w).Encode(body))
}

func clientOptions(srv *httptest.Server) []option.ClientOption {
	return []option.ClientOption{option.WithEndpoint(srv.URL + "/"), option.WithHTTPClient(srv.Client())}
}

func calendarService(t *testing.T, srv *httptest.Server) *gcalendar.Service {
	t.Helper()
	svc, err := gcalendar.NewService(t.Context(), clientOptions(srv)...)
	require.NoError(t, err)
	return svc
}

func tasksService(t *testing.T, srv *httptest.Server) *gtasks.Service {
	t.Helper()
	svc, err := gtasks.NewService(t.Context(), clientOptions(srv)...)
	require.NoError(t, err)
	return svc
}
