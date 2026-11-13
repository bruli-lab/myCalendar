package app

import (
	"context"

	"github.com/bruli-lab/go-core/cqs"
	"github.com/bruli-lab/go-core/event"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const CreateTokensCmdName = "create_tokens"

type CreateTokensCmd struct {
	Code string
}

func (CreateTokensCmd) Name() string {
	return CreateTokensCmdName
}

type CreateTokens struct {
	tokenRepo TokenRepository
	authRepo  AuthenticationRepository
	tracer    trace.Tracer
}

func (c CreateTokens) Handle(ctx context.Context, cmd cqs.Command) ([]event.Event, error) {
	ctx, span := c.tracer.Start(ctx, "CreateTokens")
	defer span.End()
	co, ok := cmd.(CreateTokensCmd)
	if !ok {
		return nil, recordError(span, cqs.NewInvalidCommandError(CreateTokensCmdName, cmd.Name()))
	}
	exchanged, err := c.tokenRepo.Exchange(ctx, co.Code)
	if err != nil {
		return nil, recordError(span, err)
	}
	if err = c.authRepo.Save(ctx, exchanged); err != nil {
		return nil, recordError(span, err)
	}
	span.SetStatus(codes.Ok, "tokens created")
	return nil, nil
}

func NewCreateTokens(tokenRepo TokenRepository, authRepo AuthenticationRepository, tracer trace.Tracer) *CreateTokens {
	return &CreateTokens{tokenRepo: tokenRepo, authRepo: authRepo, tracer: tracer}
}
