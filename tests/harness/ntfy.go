//go:build functional

package harness

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	ntfyImage      = "binwiederhier/ntfy:v2.37.2"
	messageTimeout = 20 * time.Second
	authPrefix     = "Click -> "
)

type Message struct {
	Title       string `json:"title"`
	Message     string `json:"message"`
	ContentType string `json:"content_type"`
}

func Topic() string {
	return "functional-" + strings.ToLower(rand.Text()[:12])
}

func StartNtfy(ctx context.Context) (endpoint string, terminate func(), err error) {
	container, err := testcontainers.Run(
		ctx,
		ntfyImage,
		testcontainers.WithCmd("serve"),
		testcontainers.WithExposedPorts("80/tcp"),
		testcontainers.WithEnv(map[string]string{
			"NTFY_AUTH_DEFAULT_ACCESS":             "read-write",
			"NTFY_VISITOR_REQUEST_LIMIT_BURST":     "100000",
			"NTFY_VISITOR_REQUEST_LIMIT_REPLENISH": "1ms",
			"NTFY_VISITOR_MESSAGE_DAILY_LIMIT":     "0",
		}),
		testcontainers.WithWaitStrategy(wait.ForHTTP("/v1/health").WithPort("80/tcp")),
	)
	if err != nil {
		return "", nil, err
	}
	endpoint, err = container.PortEndpoint(ctx, "80/tcp", "http")
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		return "", nil, err
	}
	return endpoint, func() { _ = testcontainers.TerminateContainer(container) }, nil
}

func Messages(t *testing.T, ntfyURL, topic string) []Message {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, ntfyURL+"/"+topic+"/json?poll=1", http.NoBody)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() {
		_ = resp.Body.Close()
	}()
	var messages []Message
	decoder := json.NewDecoder(resp.Body)
	for decoder.More() {
		var msg Message
		require.NoError(t, decoder.Decode(&msg))
		messages = append(messages, msg)
	}
	return messages
}

func WaitForMessage(t *testing.T, ntfyURL, topic string, match func(Message) bool) Message {
	t.Helper()
	var found Message
	require.Eventually(t, func() bool {
		for _, msg := range Messages(t, ntfyURL, topic) {
			if match(msg) {
				found = msg
				return true
			}
		}
		return false
	}, messageTimeout, 200*time.Millisecond, "message not published to topic %s", topic)
	return found
}

func IsAuthorization(msg Message) bool {
	return strings.HasPrefix(msg.Message, authPrefix)
}

func AuthorizationState(t *testing.T, msg Message) string {
	t.Helper()
	authURL, err := url.Parse(strings.TrimPrefix(msg.Message, authPrefix))
	require.NoError(t, err)
	state := authURL.Query().Get("state")
	require.NotEmpty(t, state, "authorization url without state")
	return url.QueryEscape(state)
}
