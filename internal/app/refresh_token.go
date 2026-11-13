package app

import (
	"context"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const RefreshTokenCmdName = "refresh_token"

type RefreshTokenCmd struct{}

func (RefreshTokenCmd) Name() string {
	return RefreshTokenCmdName
}

type RefreshToken struct {
	authRepo  AuthenticationRepository
	tokenRepo TokenRepository
	tracer    trace.Tracer
}

func (r RefreshToken) Handle(ctx context.Context, cmd cqs.Command) ([]event.Event, error) {
	ctx, span := r.tracer.Start(ctx, "RefreshToken")
	defer span.End()
	if _, ok := cmd.(RefreshTokenCmd); !ok {
		return nil, recordError(span, cqs.NewInvalidCommandError(RefreshTokenCmdName, cmd.Name()))
	}
	stored, err := r.authRepo.Read(ctx)
	if err != nil {
		return nil, recordError(span, err)
	}
	refreshed, err := r.tokenRepo.Refresh(ctx, stored.RefreshToken())
	if err != nil {
		return nil, recordError(span, err)
	}
	if err = r.authRepo.Save(ctx, refreshed); err != nil {
		return nil, recordError(span, err)
	}
	span.SetStatus(codes.Ok, "token refreshed")
	return nil, nil
}

func NewRefreshToken(authRepo AuthenticationRepository, tokenRepo TokenRepository, tracer trace.Tracer) *RefreshToken {
	return &RefreshToken{authRepo: authRepo, tokenRepo: tokenRepo, tracer: tracer}
}
