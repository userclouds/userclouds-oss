package email

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"

	"userclouds.com/infra/ucaws"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

const charset = "UTF-8"

// awsClient client implement the Client interface for our own (not customer defined) SES email service
type awsClient struct {
	client *sesv2.Client
}

// NewClient creates a new instance of Dispatcher which can send emails via AWS Simple Email Service.
func NewClient(ctx context.Context) (Client, error) {
	cfg, err := ucaws.NewConfigWithDefaultRegion(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &awsClient{client: sesv2.NewFromConfig(cfg)}, nil
}

func (c awsClient) makeContent(data string) *types.Content {
	return &types.Content{
		Charset: aws.String(charset),
		Data:    &data,
	}
}

// Send sends an email via SES with only a text body
// note that this requires the from address to be verified, etc
func (c awsClient) Send(ctx context.Context, to, from, subject, body string) error {
	return ucerr.Wrap(c.SendWithHTML(ctx, to, from, subject, body, ""))
}

// Send sends an email via SES with both HTML and alternative text body
// note that this requires the from address to be verified, etc
func (c *awsClient) SendWithHTML(ctx context.Context, to, from, subject, body, htmlBody string) error {
	uclog.Infof(ctx, "Sending email to '%s' from '%s' with subject '%s'", to, from, subject)
	msgBody := types.Body{Text: c.makeContent(body)}
	if htmlBody != "" {
		// If html body was provided send it as well
		msgBody.Html = c.makeContent(htmlBody)
	}
	input := &sesv2.SendEmailInput{
		FromEmailAddress: &from,
		Destination: &types.Destination{
			CcAddresses: []string{},
			ToAddresses: []string{to},
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Body:    &msgBody,
				Subject: c.makeContent(subject)},
		},
	}

	if _, err := c.client.SendEmail(ctx, input); err != nil {
		// log error messages if they occur.
		uclog.Warningf(ctx, "Got an error sending the email: %v", err)
		return ucerr.Wrap(err)
	}
	return nil
}
