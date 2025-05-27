package email

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucerr"
)

const smtpServerPlaceholderText = "********"

// SMTPServer keeps optional record for a custom smtp server connection
type SMTPServer struct {
	Host       string        `json:"host" yaml:"host" validate:"notempty"`
	Port       int           `json:"port" yaml:"port"`
	Username   string        `json:"username" yaml:"username" validate:"notempty"`
	PasswordUI string        `json:"password_ui" yaml:"password_ui" validate:"skip"`
	Password   secret.String `json:"password" yaml:"password"`
}

// IsConfigured returns true if the SMTPServer is configured
func (s SMTPServer) IsConfigured() bool {
	return s.Host != ""
}

// DecodeSecrets decodes any secrets in the SMTPServer configuration
func (s *SMTPServer) DecodeSecrets(ctx context.Context) error {
	if s.IsConfigured() {
		resolvedPassword, err := s.Password.Resolve(ctx)
		if err != nil {
			return ucerr.Wrap(err)
		}
		if resolvedPassword == "" {
			s.PasswordUI = ""
		} else {
			s.PasswordUI = smtpServerPlaceholderText
		}
	} else {
		s.PasswordUI = ""
	}

	s.Password = secret.String{}
	return nil
}

// EncodeSecrets encodes any secrets in the SMTPServer configuration
func (s *SMTPServer) EncodeSecrets(ctx context.Context, ownerID uuid.UUID, source SMTPServer) error {
	switch s.PasswordUI {
	case smtpServerPlaceholderText:
		s.Password = source.Password
	case "":
		s.Password = source.Password
	default:
		// user entered a new password
		password, err := secret.NewString(ctx, universe.ServiceName(), fmt.Sprintf("%v_email_password", ownerID), s.PasswordUI)
		if err != nil {
			return ucerr.Wrap(err)
		}
		s.Password = *password
	}

	s.PasswordUI = ""
	return nil
}

// NewClient returns a new SMTPClient for the SMTPServer configuration
func (s SMTPServer) NewClient() Client {
	return NewSMTPClient(s.Host, s.Port, s.Username, s.Password)
}

//go:generate genvalidate SMTPServer
