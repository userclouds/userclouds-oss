package gomysqlserver

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"

	"userclouds.com/infra/ucerr"
)

// NewServerTLSConfig generate TLS config for server side
// controlling the security level by authType
func NewServerTLSConfig(caPem, certPem, keyPem []byte, authType tls.ClientAuthType) (*tls.Config, error) {
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPem) {
		return nil, ucerr.New("Failed to append CA certificate")
	}

	cert, err := tls.X509KeyPair(certPem, keyPem)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	config := &tls.Config{
		ClientAuth:   authType,
		Certificates: []tls.Certificate{cert},
		ClientCAs:    pool,
	}
	return config, nil
}

// extract RSA public key from certificate
func getPublicKeyFromCert(certPem []byte) ([]byte, error) {
	block, _ := pem.Decode(certPem)
	crt, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	pubKey, err := x509.MarshalPKIXPublicKey(crt.PublicKey.(*rsa.PublicKey))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubKey}), nil
}

// generate and sign RSA certificates with given CA
// see: https://fale.io/blog/2017/06/05/create-a-pki-in-golang/
func generateAndSignRSACerts(caPem, caKey []byte) ([]byte, []byte, error) {
	// Load CA
	catls, err := tls.X509KeyPair(caPem, caKey)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}
	ca, err := x509.ParseCertificate(catls.Certificate[0])
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// use the CA to sign certificates
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}
	cert := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization:  []string{"ORGANIZATION_NAME"},
			Country:       []string{"COUNTRY_CODE"},
			Province:      []string{"PROVINCE"},
			Locality:      []string{"CITY"},
			StreetAddress: []string{"ADDRESS"},
			PostalCode:    []string{"POSTAL_CODE"},
		},
		NotBefore:    time.Now().UTC(),
		NotAfter:     time.Now().UTC().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// sign the certificate
	certB, err := x509.CreateCertificate(rand.Reader, ca, cert, &priv.PublicKey, catls.PrivateKey)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}
	certPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certB})
	keyPem := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	return certPem, keyPem, nil
}

// generate CA in PEM
// see: https://github.com/golang/go/blob/master/src/crypto/tls/generate_cert.go
func generateCA() ([]byte, []byte, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization:  []string{"ORGANIZATION_NAME"},
			Country:       []string{"COUNTRY_CODE"},
			Province:      []string{"PROVINCE"},
			Locality:      []string{"CITY"},
			StreetAddress: []string{"ADDRESS"},
			PostalCode:    []string{"POSTAL_CODE"},
		},
		NotBefore:             time.Now().UTC(),
		NotAfter:              time.Now().UTC().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	caPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	caKey := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	return caPem, caKey, nil
}
