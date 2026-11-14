//go:build functional

package functional_test

import (
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bruli-lab/myCalendar/tests/harness"
	"github.com/stretchr/testify/require"
)

const expectedDailyBody = "**Events**\n" +
	"- All day · [**Bank holiday**](https://calendar.test/1)\n" +
	"- 09:30 · [**Team meeting**](https://calendar.test/2)\n" +
	"\n**Tasks**\n" +
	"- [**Buy milk**](https://tasks.test/1)"

func isDailySummary(msg harness.Message) bool {
	return strings.HasPrefix(msg.Title, "📅 ") && !strings.HasPrefix(msg.Title, "📅 Week")
}

func TestAuthorizationAndDailySummary(t *testing.T) {
	t.Run(`Given the app without stored tokens,
	when the user opens the published authorization link and Google redirects back with the right state,
	then it stores the tokens, publishes the daily summary as markdown and stops cleanly on SIGTERM`, func(t *testing.T) {
		t.Parallel()
		topic := harness.Topic()
		google := harness.NewFakeGoogle(t)
		tokensFile := harness.TokensFile(t)
		app := harness.Start(t, harness.Options{NtfyURL: ntfyURL, Topic: topic, GoogleURL: google.URL(), TokensFile: tokensFile})

		authMsg := harness.WaitForMessage(t, ntfyURL, topic, harness.IsAuthorization)
		require.Equal(t, http.StatusBadRequest, app.Authorize(t, "forged", harness.AuthorizationCode))
		require.Equal(t, http.StatusOK, app.Authorize(t, harness.AuthorizationState(t, authMsg), harness.AuthorizationCode))

		summary := harness.WaitForMessage(t, ntfyURL, topic, isDailySummary)
		require.Equal(t, expectedDailyBody, summary.Message)
		require.Equal(t, "text/markdown", summary.ContentType)
		tokens, mode := harness.ReadTokens(t, tokensFile)
		require.Equal(t, harness.InitialRefreshToken, tokens.RefreshToken)
		require.Equal(t, os.FileMode(0o600), mode)
		require.Equal(t, 0, app.Stop())
	})
}

func TestRevokedRefreshToken(t *testing.T) {
	t.Run(`Given stored tokens whose refresh token was revoked,
	when the app starts,
	then it publishes a new authorization link instead of stopping`, func(t *testing.T) {
		t.Parallel()
		topic := harness.Topic()
		google := harness.NewFakeGoogle(t)
		google.RevokeRefreshToken()
		tokensFile := harness.TokensFile(t)
		harness.WriteTokens(t, tokensFile, harness.StoredTokens{
			AccessToken:  "access-old",
			TokenType:    "Bearer",
			RefreshToken: "revoked",
			Expiry:       time.Now().Add(-time.Hour),
		})
		app := harness.Start(t, harness.Options{NtfyURL: ntfyURL, Topic: topic, GoogleURL: google.URL(), TokensFile: tokensFile})

		harness.WaitForMessage(t, ntfyURL, topic, harness.IsAuthorization)
		require.True(t, app.Healthy(t.Context()))
	})

	t.Run(`Given a running app with valid tokens,
	when Google revokes the refresh token while the access token has expired,
	then the next summary job publishes a new authorization link and the app keeps running`, func(t *testing.T) {
		t.Parallel()
		topic := harness.Topic()
		google := harness.NewFakeGoogle(t)
		tokensFile := harness.TokensFile(t)
		harness.WriteTokens(t, tokensFile, harness.StoredTokens{
			AccessToken:  "access-old",
			TokenType:    "Bearer",
			RefreshToken: "valid",
			Expiry:       time.Now().Add(-time.Hour),
		})
		google.ExpireAccessTokensImmediately()
		app := harness.Start(t, harness.Options{NtfyURL: ntfyURL, Topic: topic, GoogleURL: google.URL(), TokensFile: tokensFile})
		harness.WaitForMessage(t, ntfyURL, topic, isDailySummary)

		google.RevokeRefreshToken()

		harness.WaitForMessage(t, ntfyURL, topic, harness.IsAuthorization)
		require.True(t, app.Healthy(t.Context()))
		require.Positive(t, google.RefreshCalls())
	})
}
