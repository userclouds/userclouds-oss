package email

import (
	"bytes"
	"context"
	"fmt"
	"net/mail"
	"net/smtp"
	"text/template"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

// We use a fixed template to fill in a multi part email with both HTML and alternative text body
const emailTemp = `Subject: {{ .Subject }}
From: {{ .From }}
To: {{ .To }}
Date: {{ .Date }}{{ if .ListUnsubscribe }}
List-Unsubscribe: {{ .ListUnsubscribe }}{{ end }}
MIME-Version: 1.0
Content-Type: multipart/alternative; boundary={{ .Boundary }}

--{{ .Boundary }}
Content-Type: text/plain; charset=utf-8
Content-Transfer-Encoding: 7bit

{{ .TextQP }}

--{{ .Boundary }}
Content-Type: text/html; charset=utf-8
Content-Transfer-Encoding: 7bit

{{ .HTMLQP }}

--{{ .Boundary }}--

`

// emailInfo is used to fill in the mime part template
type emailInfo struct {
	From            string
	To              string
	Subject         string
	Boundary        string
	ListUnsubscribe string
	HTMLQP          string
	TextQP          string
	Date            string
}

// smtpClient client implement the Client interface for our external SMTP email service
type smtpClient struct {
	Host     string        `json:"host" yaml:"host"`
	Port     int           `json:"port" yaml:"port"`
	Username string        `json:"username" yaml:"username"`
	Password secret.String `json:"password" yaml:"password"`
}

// NewSMTPClient creates a new instance of Client which can send emails using the smpt server provided
func NewSMTPClient(host string, port int, username string, password secret.String) Client {
	return &smtpClient{Host: host, Port: port, Username: username, Password: password}
}

// Send sends an email via given smpt server with only a text body
// note that this requires the from address to be verified, etc
func (s smtpClient) Send(ctx context.Context, to, from, subject, body string) error {
	return ucerr.Wrap(s.SendWithHTML(ctx, to, from, subject, body, ""))
}

// Send sends an email via  via given smpt server with both HTML and alternative text body
// note that this requires the from address to be verified, etc
func (s smtpClient) SendWithHTML(ctx context.Context, to, from, subject, body, htmlBody string) error {

	// Connect to the remote SMTP server.
	uclog.Verbosef(ctx, "Sending SMTP mail[%s,%d] as %s to %s from %s with subj %s", s.Host, s.Port, s.Username, from, to, subject)
	// Set up authentication information.
	password, err := s.Password.Resolve(ctx)
	if err != nil {
		return ucerr.Wrap(err)
	}
	auth := smtp.PlainAuth("", s.Username, password, s.Host)

	// The RFC for this is here - https://datatracker.ietf.org/doc/html/rfc2045. There are a couple of libraries like
	// https://github.com/go-gomail/gomail/ but doesn't seem worth pulling them in given how constrained our scenario is
	boundaryMarker := fmt.Sprintf("Part_%s", uuid.Must(uuid.NewV4()).String())

	// Prepare the struct to pass to the template
	tmpl, err := template.New("Email Template").Parse(emailTemp)
	if err != nil {
		return ucerr.Wrap(err)
	}
	e := emailInfo{
		From:            from,
		To:              to,
		Boundary:        boundaryMarker,
		ListUnsubscribe: "", // TODO we likely need this
		Subject:         subject,
		Date:            time.Now().UTC().Format(time.RFC1123Z),
		HTMLQP:          htmlBody,
		TextQP:          body,
	}

	// Compile the template with current email information
	var outputBuffer bytes.Buffer
	err = tmpl.Execute(&outputBuffer, e)
	if err != nil {
		return ucerr.Wrap(err)
	}

	// If we were passed name in RFC 5322 format like "Barry Gibbs <bg@example.com>" extract the email address only since that is what smtp.SendMail expects
	fromAddress, err := mail.ParseAddress(from)
	if err != nil {
		return ucerr.Errorf("invalid email: %w", err)
	}
	toAddress, err := mail.ParseAddress(to)
	if err != nil {
		return ucerr.Errorf("invalid email: %w", err)
	}

	if err := smtp.SendMail(fmt.Sprintf("%s:%d", s.Host, s.Port), auth, fromAddress.Address, []string{toAddress.Address}, outputBuffer.Bytes()); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}
