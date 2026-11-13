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
	newToken, err := t.cfg.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken}).Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", asRefreshError(err))
	}
	return toAuth(newToken)
}

func (t TokenRepository) Exchange(ctx context.Context, code string) (*auth.Auth, error) {
	token, err := t.cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}
	return toAuth(token)
}

func NewTokenRepository(cfg *oauth2.Config) *TokenRepository {
	return &TokenRepository{cfg: cfg}
}

func toAuth(token *oauth2.Token) (*auth.Auth, error) {
	return auth.New(token.AccessToken, token.RefreshToken, token.Expiry, token.TokenType)
}

func toOAuthToken(a *auth.Auth) *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  a.AccessToken(),
		RefreshToken: a.RefreshToken(),
		Expiry:       a.Expiry(),
		TokenType:    a.TokenType(),
	}
}
