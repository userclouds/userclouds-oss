package telephony

import (
	"bytes"
	"context"
	texttemplate "text/template"

	"userclouds.com/infra/ucerr"
	phone "userclouds.com/infra/uctypes/messaging/telephony/phonenumber"
	message "userclouds.com/internal/messageelements"
)

// SendSMSWithTemplate sends an SMS message with a templated body via the specified provider
func SendSMSWithTemplate(ctx context.Context, c Client, to phone.PhoneNumber, getter message.ElementGetter, data any) error {
	from := phone.PhoneNumber(getter(message.SMSSender))
	if err := from.Validate(); err != nil {
		return ucerr.Wrap(err)
	}

	bodyTemplate, err := texttemplate.New(string(message.SMSBodyTemplate)).Parse(getter(message.SMSBodyTemplate))
	if err != nil {
		return ucerr.Wrap(err)
	}

	buf := &bytes.Buffer{}
	if err := bodyTemplate.Execute(buf, data); err != nil {
		return ucerr.Wrap(err)
	}
	body := buf.String()

	return ucerr.Wrap(c.SendSMS(ctx, to, from, body))
}
