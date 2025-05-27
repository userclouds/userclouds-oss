package acme

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/hlandau/acmeapi"

	"userclouds.com/infra/acme"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/acmestorage"
	"userclouds.com/internal/companyconfig"
)

const maxRetries = 10

// FinalizeNewTenantURL grabs the new cert and uploads it
func FinalizeNewTenantURL(ctx context.Context, acmeCfg *acme.Config, ccs *companyconfig.Storage, tenantID uuid.UUID, tenantDB *ucdb.DB, ucOrderID uuid.UUID) error {
	ucCert, privKey, err := finalizeACMEOrder(ctx, acmeCfg, ccs, tenantDB, ucOrderID)
	if err != nil {
		return ucerr.Errorf("failed to finalize ACME order: %w", err)
	}

	if err := uploadCertToALB(ctx, tenantID, ucCert, privKey); err != nil {
		// this is here just so we don't forget if we add another step
		return ucerr.Errorf("failed to upload new cert to AWS: %w", err)
	}
	return nil
}

func finalizeACMEOrder(ctx context.Context, cfg *acme.Config, ccs *companyconfig.Storage, tenantDB *ucdb.DB, ucOrderID uuid.UUID) (*acmestorage.Certificate, *rsa.PrivateKey, error) {
	as := acmestorage.New(tenantDB)
	ucOrder, err := as.GetOrder(ctx, ucOrderID)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	rc, acct, err := newClient(ctx, cfg)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	order := acmeapi.Order{
		URL: ucOrder.URL,
	}

	if err := rc.LoadOrder(ctx, acct, &order); err != nil {
		return nil, nil, ucerr.Wrap(err)
	}
	uclog.Debugf(ctx, "loaded order to create CSR: %+v", order)

	ucCert, privCertKey, err := loadOrCreatePrivateKey(ctx, as, ucOrderID)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// build the CSR
	subj := pkix.Name{
		CommonName:   ucOrder.Host,
		Country:      []string{"US"},
		Province:     []string{"CA"},
		Locality:     []string{"Palo Alto"},
		Organization: []string{"UserClouds"},
	}
	rawSubj := subj.ToRDNSequence()
	asn1Subj, err := asn1.Marshal(rawSubj)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	tmpl := x509.CertificateRequest{
		RawSubject:         asn1Subj,
		SignatureAlgorithm: x509.SHA256WithRSA,
	}
	csr, err := x509.CreateCertificateRequest(rand.Reader, &tmpl, privCertKey)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}
	uclog.Debugf(ctx, "waiting for order %s", order.URL)

	// wait to make sure it's ready to be finalized
	var retries int
	for {
		// WaitLoadOrder handles backoff timing for us
		if err := rc.WaitLoadOrder(ctx, acct, &order); err != nil {
			return nil, nil, ucerr.Wrap(err)
		}

		if order.Status == acmeapi.OrderReady {
			break
		}

		retries++
		if retries >= maxRetries {
			return nil, nil, ucerr.Errorf("order %v never became ready", order)
		}
	}

	uclog.Debugf(ctx, "order %v ready, finalizing", order.URL)
	// now that we've responded to the challenge, we can send our CSR to get a cert
	if err := rc.Finalize(ctx, acct, &order, csr); err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	ucOrder.Status = acmestorage.OrderStatusReady
	if err := as.SaveOrder(ctx, ucOrder); err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	ucCert.Status = acmestorage.CertificateStatusRequested
	if err := as.SaveCertificate(ctx, ucCert); err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// now we have to wait for the order to be "valid" (instead of processing)
	// because of the new "asynchronous order finalization" features in Lets Encrypt
	// https://community.letsencrypt.org/t/enabling-asynchronous-order-finalization/193522
	uclog.Debugf(ctx, "waiting for order %s to be valid", order.URL)
	retries = 0
	for {
		// WaitLoadOrder handles backoff timing for us
		if err := rc.WaitLoadOrder(ctx, acct, &order); err != nil {
			return nil, nil, ucerr.Wrap(err)
		}

		if order.Status == acmeapi.OrderValid {
			break
		}

		retries++
		if retries >= maxRetries {
			return nil, nil, ucerr.Errorf("order %v never became valid", order)
		}
	}

	// and download the cert
	cert := acmeapi.Certificate{
		URL: order.CertificateURL,
	}
	if err := rc.LoadCertificate(ctx, acct, &cert); err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// save it
	var actualCert []byte
	var certChain []byte
	for i, c := range cert.CertificateChain {
		crt, err := x509.ParseCertificate(c)
		if err != nil {
			return nil, nil, ucerr.Wrap(err)
		}

		certPEM, err := certToPEM(crt)
		if err != nil {
			return nil, nil, ucerr.Wrap(err)
		}

		if i == 0 {
			actualCert = certPEM
		} else if i == 1 {
			certChain = certPEM
		} else {
			certChain = append(certChain, certPEM...)
		}
	}

	ucCert.Certificate = string(actualCert)
	ucCert.CertificateChain = string(certChain)
	if err := as.SaveCertificate(ctx, ucCert); err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// and save the validity to the tenant URL as well
	crt, err := x509.ParseCertificate(cert.CertificateChain[0])
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}
	ucCert.NotAfter = crt.NotAfter

	tu, err := ccs.GetTenantURL(ctx, ucOrder.TenantURLID)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}
	tu.CertificateValidUntil = crt.NotAfter
	// TODO update SaveTenantURLWithCache after adding cache config to worker
	if err := ccs.SaveTenantURL(ctx, tu); err != nil {
		return nil, nil, ucerr.Wrap(err)
	}
	return ucCert, privCertKey, nil
}

// TODO: these should probably be in a helper package somewhere
func privateKeyToPEM(privCertKey *rsa.PrivateKey) (string, error) {
	bs := x509.MarshalPKCS1PrivateKey(privCertKey)
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	if err := pem.Encode(w, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: bs}); err != nil {
		return "", ucerr.Wrap(err)
	}
	w.Flush()

	return buf.String(), nil
}

func privateKeyFromPEM(pkPEM string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pkPEM))
	if block == nil {
		return nil, ucerr.Errorf("failed to parse PEM block containing the key")
	}

	privCertKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return privCertKey, nil
}

func certToPEM(cert *x509.Certificate) ([]byte, error) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	if err := pem.Encode(w, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}); err != nil {
		return nil, ucerr.Wrap(err)
	}
	w.Flush()

	return buf.Bytes(), nil
}

func loadOrCreatePrivateKey(ctx context.Context, as *acmestorage.Storage, ucOrderID uuid.UUID) (*acmestorage.Certificate, *rsa.PrivateKey, error) {
	// first, see if we already have a private key for this order
	ucCert, err := as.GetCertificateByOrderID(ctx, ucOrderID)
	if err == nil {
		uclog.Debugf(ctx, "found existing ucCert %v for order %v", ucCert.ID, ucOrderID)
		pkPEM, err := ucCert.PrivateKey.Resolve(ctx)
		if err != nil {
			return nil, nil, ucerr.Wrap(err)
		}
		privCertKey, err := privateKeyFromPEM(pkPEM)
		if err != nil {
			return nil, nil, ucerr.Wrap(err)
		}

		return ucCert, privCertKey, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return nil, nil, ucerr.Wrap(err)
	}

	// otherwise sql.ErrNoRows so generate a new one
	uclog.Debugf(ctx, "generating new CSR for order %v", ucOrderID)

	// generate a key for this CSR
	privCertKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	// save it in AWS Secrets Manager for now
	pckPem, err := privateKeyToPEM(privCertKey)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}
	sec, err := secret.NewString(ctx, universe.ServiceName(), fmt.Sprintf("acme-order-%s-private-key", ucOrderID), pckPem)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}
	ucCert = &acmestorage.Certificate{
		BaseModel:  ucdb.NewBase(),
		OrderID:    ucOrderID,
		Status:     acmestorage.CertificateStatusNew,
		PrivateKey: *sec,
	}
	if err := as.SaveCertificate(ctx, ucCert); err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	return ucCert, privCertKey, nil
}
