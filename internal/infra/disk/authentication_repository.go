package disk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bruli-lab/myCalendar/internal/domain/auth"
)

type TokenFile struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	RefreshToken string    `json:"refresh_token"`
	Expiry       time.Time `json:"expiry"`
}

type AuthenticationRepository struct {
	filePath string
}

func (a AuthenticationRepository) Read(ctx context.Context) (*auth.Auth, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		if !a.fileExists() {
			return nil, auth.NewRefreshError(a.filePath + " file not exists")
		}
		data, err := os.ReadFile(a.filePath)
		if err != nil {
			return nil, err
		}
		var t TokenFile
		if err := json.Unmarshal(data, &t); err != nil {
			return nil, fmt.Errorf("failed to unmarshal file: %s", err.Error())
		}
		do, err := auth.New(t.AccessToken, t.RefreshToken, t.Expiry, t.TokenType)
		if err != nil {
			return nil, auth.NewRefreshError(err.Error())
		}
		return do, nil
	}
}

func (a AuthenticationRepository) Save(ctx context.Context, aut *auth.Auth) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		tok := TokenFile{
			AccessToken:  aut.AccessToken(),
			TokenType:    aut.TokenType(),
			RefreshToken: aut.RefreshToken(),
			Expiry:       aut.Expiry(),
		}
		data, err := json.MarshalIndent(tok, "", " ")
		if err != nil {
			return fmt.Errorf("failed to marshal token: %s", err.Error())
		}
		return a.writeAtomically(data)
	}
}

func (a AuthenticationRepository) writeAtomically(data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(a.filePath), "."+filepath.Base(a.filePath)+".*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary tokens file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if err = tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to set tokens file permissions: %w", err)
	}
	if _, err = tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to write tokens file: %w", err)
	}
	if err = tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to sync tokens file: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return fmt.Errorf("failed to close tokens file: %w", err)
	}
	if err = os.Rename(tmpPath, a.filePath); err != nil {
		return fmt.Errorf("failed to replace tokens file: %w", err)
	}
	return nil
}

func (a AuthenticationRepository) fileExists() bool {
	_, err := os.Stat(a.filePath)
	return !errors.Is(err, os.ErrNotExist)
}

func NewAuthenticationRepository(filePath string) *AuthenticationRepository {
	return &AuthenticationRepository{filePath: filePath}
}
