//go:build functional

package harness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type StoredTokens struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	RefreshToken string    `json:"refresh_token"`
	Expiry       time.Time `json:"expiry"`
}

func TokensFile(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "tokens.json")
}

func WriteTokens(t *testing.T, path string, tokens StoredTokens) {
	t.Helper()
	data, err := json.Marshal(tokens)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o600))
}

func ReadTokens(t *testing.T, path string) (tokens StoredTokens, mode os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &tokens))
	return tokens, info.Mode().Perm()
}
