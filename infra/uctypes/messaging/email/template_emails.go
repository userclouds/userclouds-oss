package email

import (
	"bytes"
	"context"
	htmltemplate "html/template"
	texttemplate "text/template"

	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/messaging/email/emailaddress"
	message "userclouds.com/internal/messageelements"
)

// LinkTemplate is just an anchor tag with a placeholder for the href.
// see comment below in SendWithHTMLTemplate for why this is necessary
const LinkTemplate = `<a href="%s">`

// SendWithHTMLTemplate sends an email with a templated subject, templated text, & HTML templated
// body using the Golang template libraries.
func SendWithHTMLTemplate(ctx context.Context, c Client, to string, getter message.ElementGetter, data any) error {
	if c == nil {
		return ucerr.New("unable to send email, no client initialized")
	}

	// combine sender name with sender email address
	sender, err := emailaddress.CombineAddress(getter(message.EmailSenderName), getter(message.EmailSender))
	if err != nil {
		return ucerr.Wrap(err)
	}

	// Make and parse the HTML & text templates
	subjectTemplate, err := texttemplate.New("subject").Parse(getter(message.EmailSubjectTemplate))
	if err != nil {
		return ucerr.Wrap(err)
	}

	// TODO (sgarrity 7/24): we are using html/template for the HTML email body, which is correct,
	// but there is a weird bug (tracked https://github.com/golang/go/issues/63586) where html/template
	// escapes characters that are inside an href attribute on an anchor tag, which breaks our invite tags
	// since post go 1.17 (before https://github.com/golang/go/issues/25192 they were accepted more loosely
	// by the http parsing code). Without working around this, our invite links would look like
	// https://host/email=foo&amp;otp_code=bar&amp;session=baz instead of email=foo&otp_code=bar&session=baz.
	// We switched to text/template for the HTML email body to avoid this issue when we upgraded to go1.17,
	// but this left open the possibility of a user injecting HTML into the invite email body. So instead,
	// we switch back to html/template but we render the anchor tag as a string and then insert it wholesale
	// into the template using `template.HTML` to mark it as safe, but allowed the rest of the template
	// (which includes all UGC) to be escaped. This also means that we need two Links (Link, which is the raw
	// Link, and WorkaroundLink, which includes the pre-rendered anchor tag), because we use Link the text
	// version of the emails as well.
	// This bug in html/template is tracked by a test in template_emails_test.go that will start to fail
	// if/when that bug is fixed, and we can remove this workaround.
	htmlTemplate, err := htmltemplate.New("html_body").Parse(getter(message.EmailHTMLTemplate))
	if err != nil {
		return ucerr.Wrap(err)
	}

	textTemplate, err := texttemplate.New("text_body").Parse(getter(message.EmailTextTemplate))
	if err != nil {
		return ucerr.Wrap(err)
	}

	// Template subject and both text and html emails
	buf := &bytes.Buffer{}
	if err := subjectTemplate.Execute(buf, data); err != nil {
		return ucerr.Wrap(err)
	}
	subject := buf.String()

	buf = &bytes.Buffer{}
	if err := htmlTemplate.Execute(buf, data); err != nil {
		return ucerr.Wrap(err)
	}
	htmlBody := buf.String()

	buf = &bytes.Buffer{}
	if err := textTemplate.Execute(buf, data); err != nil {
		return ucerr.Wrap(err)
	}
	textBody := buf.String()

	return ucerr.Wrap(c.SendWithHTML(ctx, to, sender, subject, textBody, htmlBody))
}
