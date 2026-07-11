package disk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
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
			return nil, auth.NewRefreshError(fmt.Sprintf("%s file not exists", a.filePath))
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
		return os.WriteFile(a.filePath, data, 0o644)
	}
}

func (a AuthenticationRepository) fileExists() bool {
	_, err := os.Stat(a.filePath)
	return !errors.Is(err, os.ErrNotExist)
}

func NewAuthenticationRepository(filePath string) *AuthenticationRepository {
	return &AuthenticationRepository{filePath: filePath}
}
