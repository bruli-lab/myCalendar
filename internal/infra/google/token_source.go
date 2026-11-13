package google

import (
	"context"
	"fmt"
	"sync"

	"github.com/bruli-lab/myCalendar/internal/domain/auth"
	"golang.org/x/oauth2"
)

type AuthenticationRepository interface {
	Read(ctx context.Context) (*auth.Auth, error)
	Save(ctx context.Context, auth *auth.Auth) error
}

type PersistentTokenSource struct {
	ctx  context.Context
	cfg  *oauth2.Config
	repo AuthenticationRepository
	mu   sync.Mutex
}

func (p *PersistentTokenSource) Token() (*oauth2.Token, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	stored, err := p.repo.Read(p.ctx)
	if err != nil {
		return nil, err
	}
	current := toOAuthToken(stored)
	token, err := p.cfg.TokenSource(p.ctx, current).Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get a valid token: %w", asRefreshError(err))
	}
	if token.AccessToken == current.AccessToken {
		return token, nil
	}
	refreshed, err := toAuth(token)
	if err != nil {
		return nil, err
	}
	if err = p.repo.Save(p.ctx, refreshed); err != nil {
		return nil, fmt.Errorf("failed to save refreshed token: %w", err)
	}
	return token, nil
}

func NewPersistentTokenSource(ctx context.Context, cfg *oauth2.Config, repo AuthenticationRepository) *PersistentTokenSource {
	return &PersistentTokenSource{ctx: ctx, cfg: cfg, repo: repo}
}
