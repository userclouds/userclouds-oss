package builder

import (
	"userclouds.com/infra/secret"
	"userclouds.com/infra/uctypes/messaging/email"
)

// EmailServerBuilder introduces SMTPServer building methods for a TenantConfig email server
type EmailServerBuilder struct {
	TenantConfigBuilder
}

// ConfigureEmailServer returns an EmailServerBuilder that can configure a tenant config
// email server
func (tcb *TenantConfigBuilder) ConfigureEmailServer() *EmailServerBuilder {
	return &EmailServerBuilder{*tcb}
}

// ClearEmailServer clears all values for the email server
func (esb *EmailServerBuilder) ClearEmailServer() *EmailServerBuilder {
	esb.plexMap.EmailServer = email.SMTPServer{}
	return esb
}

// SetHost sets the host for the email server
func (esb *EmailServerBuilder) SetHost(host string) *EmailServerBuilder {
	esb.plexMap.EmailServer.Host = host
	return esb
}

// SetPort sets the port for the email server
func (esb *EmailServerBuilder) SetPort(port int) *EmailServerBuilder {
	esb.plexMap.EmailServer.Port = port
	return esb
}

// SetPassword sets the password for the email server
func (esb *EmailServerBuilder) SetPassword(password secret.String) *EmailServerBuilder {
	esb.plexMap.EmailServer.Password = password
	return esb
}

// SetUsername sets the username for the email server
func (esb *EmailServerBuilder) SetUsername(username string) *EmailServerBuilder {
	esb.plexMap.EmailServer.Username = username
	return esb
}
