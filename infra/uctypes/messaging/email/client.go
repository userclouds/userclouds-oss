package email

import (
	"context"
)

// Client is an interface for objects which can send email via an
// email provider service.
type Client interface {
	Send(ctx context.Context, to, from, subject, body string) error
	SendWithHTML(ctx context.Context, to, from, subject, body, htmlBody string) error
}
