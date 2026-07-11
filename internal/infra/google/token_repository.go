package google

import (
	"context"
	"fmt"

	"github.com/bruli-lab/myCalendar/internal/domain/auth"
	"golang.org/x/oauth2"
)

type TokenRepository struct {
	cfg *oauth2.Config
}

func (t TokenRepository) Refresh(ctx context.Context, refreshToken string) (*auth.Auth, error) {
	to := &oauth2.Token{
		RefreshToken: refreshToken,
	}
	ts := t.cfg.TokenSource(ctx, to)
	newToken, err := ts.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	return auth.New(newToken.AccessToken, newToken.RefreshToken, newToken.Expiry, newToken.TokenType)
}

func (t TokenRepository) Exchange(ctx context.Context, code string) (*auth.Auth, error) {
	token, err := t.cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %s", err.Error())
	}
	return auth.New(token.AccessToken, token.RefreshToken, token.Expiry, token.TokenType)
}

func NewTokenRepository(cfg *oauth2.Config) *TokenRepository {
	return &TokenRepository{cfg: cfg}
}
