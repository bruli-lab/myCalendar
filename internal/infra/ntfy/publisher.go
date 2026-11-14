package ntfy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	defaultPriority = 3
	requestTimeout  = 10 * time.Second
)

type message struct {
	Topic    string `json:"topic"`
	Title    string `json:"title"`
	Message  string `json:"message"`
	Markdown bool   `json:"markdown"`
	Priority int    `json:"priority"`
}

type Publisher struct {
	user      string
	password  string
	serverURL *url.URL
	topic     string
	tracer    trace.Tracer
	client    *http.Client
}

func (p *Publisher) Publish(ctx context.Context, title, body string) error {
	ctx, span := p.tracer.Start(ctx, "Publisher.Publish")
	defer span.End()
	payload, err := json.Marshal(message{
		Topic:    p.topic,
		Title:    title,
		Message:  body,
		Markdown: true,
		Priority: defaultPriority,
	})
	if err != nil {
		return recordError(span, fmt.Errorf("failed to marshal message: %w", err))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.serverURL.String(), bytes.NewReader(payload))
	if err != nil {
		return recordError(span, fmt.Errorf("failed to create request: %w", err))
	}
	req.SetBasicAuth(p.user, p.password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return recordError(span, fmt.Errorf("failed to publish message: %w", err))
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= http.StatusMultipleChoices {
		return recordError(span, fmt.Errorf("ntfy returned status %d", resp.StatusCode))
	}
	span.SetStatus(codes.Ok, "message published")
	return nil
}

func recordError(span trace.Span, err error) error {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

func NewPublisher(user, password, baseURL, topic string, tracer trace.Tracer) (*Publisher, error) {
	serverURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse publisher url: %w", err)
	}
	return &Publisher{
		user:      user,
		password:  password,
		serverURL: serverURL,
		topic:     topic,
		tracer:    tracer,
		client:    &http.Client{Timeout: requestTimeout},
	}, nil
}
