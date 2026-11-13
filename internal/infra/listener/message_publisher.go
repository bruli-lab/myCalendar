package listener

import "context"

//go:generate go tool moq -out message_publisher_mock_test.go -pkg listener_test . MessagePublisher

type MessagePublisher interface {
	Publish(ctx context.Context, title, body string) error
}
