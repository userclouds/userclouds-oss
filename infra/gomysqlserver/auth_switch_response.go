package gomysqlserver

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/tls"
	"fmt"

	"github.com/go-mysql-org/go-mysql/mysql"

	"userclouds.com/infra/ucerr"
)

func (c *Conn) handleAuthSwitchResponse() error {
	authData, err := c.readAuthSwitchRequestResponse()
	if err != nil {
		return ucerr.Wrap(err)
	}

	switch c.authPluginName {
	case mysql.AUTH_NATIVE_PASSWORD:
		if err := c.acquirePassword(); err != nil {
			return ucerr.Wrap(err)
		}
		return ucerr.Wrap(c.compareNativePasswordAuthData(authData, c.password))

	case mysql.AUTH_CACHING_SHA2_PASSWORD:
		if !c.cachingSha2FullAuth {
			// Switched auth method but no MoreData packet send yet
			if err := c.compareCacheSha2PasswordAuthData(); err != nil {
				return ucerr.Wrap(err)
			}

			if c.cachingSha2FullAuth {
				return ucerr.Wrap(c.handleAuthSwitchResponse())
			}
			return nil
		}
		// AuthMoreData packet already sent, do full auth
		if err := c.handleCachingSha2PasswordFullAuth(authData); err != nil {
			return ucerr.Wrap(err)
		}
		c.writeCachingSha2Cache()
		return nil

	case mysql.AUTH_SHA256_PASSWORD:
		cont, err := c.handlePublicKeyRetrieval(authData)
		if err != nil {
			return ucerr.Wrap(err)
		}
		if !cont {
			return nil
		}
		return ucerr.Wrap(c.compareSha256PasswordAuthData(authData))

	default:
		return ucerr.Errorf("unknown authentication plugin name '%s'", c.authPluginName)
	}
}

func (c *Conn) handleCachingSha2PasswordFullAuth(authData []byte) error {
	if tlsConn, ok := c.Conn.Conn.(*tls.Conn); ok {
		if !tlsConn.ConnectionState().HandshakeComplete {
			return ucerr.New("incomplete TSL handshake")
		}
		// connection is SSL/TLS, client should send plain password
		// deal with the trailing \NUL added for plain text password received
		if l := len(authData); l != 0 && authData[l-1] == 0x00 {
			authData = authData[:l-1]
		}

		ok, err := c.credentialProvider.CheckPassword(c.user, string(authData), c.capability)
		if err != nil {
			return ucerr.Wrap(err)
		}
		if ok {
			return nil
		}

		return ucerr.Wrap(errAccessDenied(c.password))
	}

	// client either request for the public key or send the encrypted password
	if len(authData) == 1 && authData[0] == 0x02 {
		// send the public key
		if err := c.writeAuthMoreDataPubkey(); err != nil {
			return ucerr.Wrap(err)
		}
		// read the encrypted password
		var err error
		if authData, err = c.readAuthSwitchRequestResponse(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	// the encrypted password
	// decrypt
	dbytes, err := rsa.DecryptOAEP(sha1.New(), rand.Reader, (c.serverConf.tlsConfig.Certificates[0].PrivateKey).(*rsa.PrivateKey), authData, nil)
	if err != nil {
		return ucerr.Wrap(err)
	}
	for i := range dbytes {
		j := i % len(c.salt)
		dbytes[i] ^= c.salt[j]
	}

	// trim the trailing \NUL
	password := string(dbytes[:len(dbytes)-1])

	ok, err := c.credentialProvider.CheckPassword(c.user, password, c.capability)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if ok {
		return nil
	}

	return ucerr.Wrap(errAccessDenied(c.password))
}

func (c *Conn) writeCachingSha2Cache() {
	// write cache
	if c.password == "" {
		return
	}
	// SHA256(PASSWORD)
	crypt := sha256.New()
	crypt.Write([]byte(c.password))
	m1 := crypt.Sum(nil)
	// SHA256(SHA256(PASSWORD))
	crypt.Reset()
	crypt.Write(m1)
	m2 := crypt.Sum(nil)
	// caching_sha2_password will maintain an in-memory hash of `user`@`host` => SHA256(SHA256(PASSWORD))
	c.serverConf.cacheShaPassword.Store(fmt.Sprintf("%s@%s", c.user, c.Conn.LocalAddr()), m2)
}
