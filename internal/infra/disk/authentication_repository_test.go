//go:build infra

package disk_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bruli-lab/myCalendar/internal/domain/auth"
	"github.com/bruli-lab/myCalendar/internal/fixtures"
	"github.com/bruli-lab/myCalendar/internal/infra/disk"
	"github.com/stretchr/testify/require"
)

func tokensFile(t *testing.T, content *string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tokens.json")
	if content != nil {
		require.NoError(t, os.WriteFile(path, []byte(*content), 0o600))
	}
	return path
}

func TestAuthenticationRepository_Read(t *testing.T) {
	invalidJSON := "{"
	incompleteTokens := `{"access_token":"access","token_type":"Bearer"}`
	validTokens := `{"access_token":"access","token_type":"Bearer","refresh_token":"refresh","expiry":"2026-11-13T09:00:00Z"}`
	tests := []struct {
		name                string
		content             *string
		expectedRefreshErr  bool
		expectedErr         bool
		expectedAccessToken string
	}{
		{
			name:               "with a missing tokens file, then it returns a refresh error",
			expectedRefreshErr: true,
		},
		{
			name:        "with an invalid JSON file, then it returns an error",
			content:     &invalidJSON,
			expectedErr: true,
		},
		{
			name:               "with an incomplete tokens file, then it returns a refresh error",
			content:            &incompleteTokens,
			expectedRefreshErr: true,
		},
		{
			name:                "with a valid tokens file, then it returns the stored auth",
			content:             &validTokens,
			expectedAccessToken: "access",
		},
	}
	for _, tt := range tests {
		t.Run(`Given a disk AuthenticationRepository,
		when Read method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			repo := disk.NewAuthenticationRepository(tokensFile(t, tt.content))

			got, err := repo.Read(t.Context())

			switch {
			case tt.expectedRefreshErr:
				require.ErrorAs(t, err, &auth.RefreshError{})
			case tt.expectedErr:
				require.Error(t, err)
			default:
				require.NoError(t, err)
				require.Equal(t, tt.expectedAccessToken, got.AccessToken())
			}
		})
	}

	t.Run(`Given a disk AuthenticationRepository,
	when Read method is called with a cancelled context,
	then it returns a context canceled error`, func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		_, err := disk.NewAuthenticationRepository(tokensFile(t, &validTokens)).Read(ctx)

		require.ErrorIs(t, err, context.Canceled)
	})
}

func TestAuthenticationRepository_Save(t *testing.T) {
	t.Run(`Given a disk AuthenticationRepository with an existing world readable tokens file,
	when Save method is called,
	then it replaces the file with owner only permissions, leaves no temporary files and the auth can be read back`, func(t *testing.T) {
		t.Parallel()
		previous := "{}"
		path := tokensFile(t, &previous)
		require.NoError(t, os.Chmod(path, 0o644))
		repo := disk.NewAuthenticationRepository(path)
		expected := fixtures.AuthBuilder{}.Build()

		require.NoError(t, repo.Save(t.Context(), expected))

		info, err := os.Stat(path)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
		entries, err := os.ReadDir(filepath.Dir(path))
		require.NoError(t, err)
		require.Len(t, entries, 1)
		got, err := repo.Read(t.Context())
		require.NoError(t, err)
		require.Equal(t, expected.AccessToken(), got.AccessToken())
		require.Equal(t, expected.RefreshToken(), got.RefreshToken())
		require.Equal(t, expected.TokenType(), got.TokenType())
		require.True(t, expected.Expiry().Equal(got.Expiry()))
	})

	t.Run(`Given a disk AuthenticationRepository pointing to a missing directory,
	when Save method is called,
	then it returns an error`, func(t *testing.T) {
		t.Parallel()
		repo := disk.NewAuthenticationRepository(filepath.Join(t.TempDir(), "missing", "tokens.json"))

		require.Error(t, repo.Save(t.Context(), fixtures.AuthBuilder{}.Build()))
	})

	t.Run(`Given a disk AuthenticationRepository,
	when Save method is called with a cancelled context,
	then it returns a context canceled error and does not create the file`, func(t *testing.T) {
		t.Parallel()
		path := tokensFile(t, nil)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		err := disk.NewAuthenticationRepository(path).Save(ctx, fixtures.AuthBuilder{}.Build())

		require.ErrorIs(t, err, context.Canceled)
		require.NoFileExists(t, path)
	})
}
