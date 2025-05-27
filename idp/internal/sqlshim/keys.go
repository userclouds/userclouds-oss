package sqlshim

// TODO (sgarrity 6/24): this should probably live in it's own package?

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"

	"userclouds.com/infra/ucerr"
)

func generateKeys() (cert, pubKeyPEM, privKey []byte, err error) {
	// Generate a new private key
	// NB: a lot of clients seem to support ECDSA or RSA, but Retool in particular only supports RSA
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, nil, ucerr.Wrap(err)
	}

	// Create a certificate template
	notBefore := time.Now().UTC()
	notAfter := notBefore.Add(10 * 365 * 24 * time.Hour) // Valid for ten years

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, nil, ucerr.Wrap(err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"UserClouds"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Create the certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, nil, ucerr.Wrap(err)
	}

	var certOut []byte
	certWriter := bytes.NewBuffer(certOut)

	err = pem.Encode(certWriter, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	if err != nil {
		return nil, nil, nil, ucerr.Wrap(err)
	}

	var keyOut []byte
	keyWriter := bytes.NewBuffer(keyOut)

	privBytes := x509.MarshalPKCS1PrivateKey(priv)
	err = pem.Encode(keyWriter, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	if err != nil {
		return nil, nil, nil, ucerr.Wrap(err)
	}

	// grab the public key from the cert for the mysql auth algos to use
	crt, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, nil, ucerr.Wrap(err)
	}

	rsaPubKey, ok := crt.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, nil, nil, ucerr.New("failed to parse public key, this should never happen")
	}

	pubKey, err := x509.MarshalPKIXPublicKey(rsaPubKey)
	if err != nil {
		return nil, nil, nil, ucerr.Wrap(err)
	}

	pubKeyPem := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubKey})

	return certWriter.Bytes(), pubKeyPem, keyWriter.Bytes(), nil
}
