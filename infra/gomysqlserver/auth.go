package gomysqlserver

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/tls"

	"github.com/go-mysql-org/go-mysql/mysql"

	"userclouds.com/infra/ucerr"
)

// Error definitions
var (
	ErrAccessDenied           = ucerr.Friendlyf(nil, "access denied")
	ErrAccessDeniedNoPassword = ucerr.Friendlyf(ErrAccessDenied, "access denied without password")
)

func (c *Conn) compareAuthData(authPluginName string, clientAuthData []byte) error {
	switch authPluginName {
	case mysql.AUTH_NATIVE_PASSWORD:
		if err := c.acquirePassword(); err != nil {
			return ucerr.Wrap(err)
		}
		return ucerr.Wrap(c.compareNativePasswordAuthData(clientAuthData, c.password))

	case mysql.AUTH_CACHING_SHA2_PASSWORD:
		if err := c.compareCacheSha2PasswordAuthData(); err != nil {
			return ucerr.Wrap(err)
		}
		if c.cachingSha2FullAuth {
			return ucerr.Wrap(c.handleAuthSwitchResponse())
		}
		return nil

	case mysql.AUTH_SHA256_PASSWORD:
		cont, err := c.handlePublicKeyRetrieval(clientAuthData)
		if err != nil {
			return ucerr.Wrap(err)
		}
		if !cont {
			return nil
		}
		return ucerr.Wrap(c.compareSha256PasswordAuthData(clientAuthData))

	default:
		return ucerr.Errorf("unknown authentication plugin name '%s'", authPluginName)
	}
}

func (c *Conn) acquirePassword() error {
	password, found, err := c.credentialProvider.GetCredential(c.user)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if !found {
		// TODO (sgarrity 6/24): I think this is actually intended as no-such-user referencing eg. an object owner,
		// as opposed to an auth error, since the error message returned by the mysql client is
		// different when you try MySQL with a non-existent user instead of us. Reading the docs this should be
		// MySQL error 1045 ER_ACCESS_DENIED_ERROR
		return ucerr.Wrap(mysql.NewDefaultError(mysql.ER_NO_SUCH_USER, c.user, c.RemoteAddr().String()))
	}
	c.password = password
	return nil
}

func errAccessDenied(password string) error {
	if password == "" {
		return ucerr.Wrap(ErrAccessDeniedNoPassword)
	}

	return ucerr.Wrap(ErrAccessDenied)
}

func (c *Conn) compareNativePasswordAuthData(clientAuthData []byte, password string) error {
	if bytes.Equal(mysql.CalcPassword(c.salt, []byte(password)), clientAuthData) {
		return nil
	}
	return ucerr.Wrap(errAccessDenied(password))
}

func (c *Conn) compareSha256PasswordAuthData(clientAuthData []byte) error {
	password := ""

	// figure out what password we should actually check
	if len(clientAuthData) == 0 {
		// Empty passwords are not hashed, but sent as empty string
		password = "" // this is a no-op but it sets up the else
	} else if tlsConn, ok := c.Conn.Conn.(*tls.Conn); ok {
		if !tlsConn.ConnectionState().HandshakeComplete {
			return ucerr.New("incomplete TSL handshake")
		}
		// connection is SSL/TLS, client should send plain password
		// deal with the trailing \NUL added for plain text password received
		if l := len(clientAuthData); l != 0 && clientAuthData[l-1] == 0x00 {
			clientAuthData = clientAuthData[:l-1]
		}

		password = string(clientAuthData)
	} else {
		// client should send encrypted password, so decrypt it
		dbytes, err := rsa.DecryptOAEP(sha1.New(), rand.Reader, (c.serverConf.tlsConfig.Certificates[0].PrivateKey).(*rsa.PrivateKey), clientAuthData, nil)
		if err != nil {
			return ucerr.Wrap(err)
		}

		// reverse out the salt so we can use CP to check the password
		for i := range dbytes {
			j := i % len(c.salt)
			dbytes[i] ^= c.salt[j]
		}

		// trim the trailing \NUL
		password = string(dbytes)[:len(dbytes)-1]
	}

	// use credentialProvider.CheckPassword if we can
	ok, err := c.credentialProvider.CheckPassword(c.user, password, c.capability)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if ok {
		return nil
	}
	return ucerr.Wrap(errAccessDenied(password))
}

func (c *Conn) compareCacheSha2PasswordAuthData() error {
	// TODO (sgarrity 6/24): caching SHA2 passwords requires that we cache a hash in memory from
	// prior authentications, and I haven't implemented that yet ... not sure if we ever should,
	// given that we'd also have to store the plaintext password in memory to allow us to connect
	// to the target DB as the correct user. If we decide to implement this, git log this change :)
	//
	// Instead, at least for now, we just "miss" on all cache attempts and do the full auth protocol

	// cache miss, do full auth
	if err := c.writeAuthMoreDataFullAuth(); err != nil {
		return ucerr.Wrap(err)
	}
	c.cachingSha2FullAuth = true
	return nil
}
