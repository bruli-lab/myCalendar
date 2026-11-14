package app_test

import (
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

type invalidCommand struct{}

func (invalidCommand) Name() string {
	return "invalid"
}

func tracer() trace.Tracer {
	return noop.NewTracerProvider().Tracer("test")
}
