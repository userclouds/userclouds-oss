package acme

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	"github.com/hlandau/acmeapi"
	"github.com/miekg/dns"

	"userclouds.com/infra/acme"
	"userclouds.com/infra/dnsclient"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/acmestorage"
	"userclouds.com/internal/companyconfig"
)

// SetupNewTenantURL handles checking DNS for a new tenant URL etc
func SetupNewTenantURL(ctx context.Context,
	dnsClient dnsclient.Client,
	acmeCfg *acme.Config,
	tenantDB *ucdb.DB,
	tenantID uuid.UUID,
	url string,
	companyConfigStorage *companyconfig.Storage) error {
	tenant, err := companyConfigStorage.GetTenant(ctx, tenantID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	hostname, tenantURL, err := checkNewTenantDNS(ctx, dnsClient, tenant, url, companyConfigStorage)
	if err != nil {
		return ucerr.Errorf("failed to validate new tenant: %w", err)
	}
	if err := createNewACMEOrder(ctx, companyConfigStorage, acmeCfg, tenantDB, tenantURL, hostname); err != nil {
		return ucerr.Errorf("failed to create new ACME order: %w", err)
	}
	return nil
}

// this pattern just makes error logging / testing easier, since we can't return
// errors from goroutines (without pissing off errcheck)
func checkNewTenantDNS(ctx context.Context, dnsClient dnsclient.Client, tenant *companyconfig.Tenant, newURL string, companyConfigStorage *companyconfig.Storage) (string, *companyconfig.TenantURL, error) {
	// check DNS points to us (and to correct main tenant URL)
	valid, newHostname, err := checkCNAME(ctx, tenant, dnsClient, newURL)
	if err != nil {
		return "", nil, ucerr.Wrap(err)
	}

	// if it's already valid, mark it as such
	// we can't use GetTenantURLbyURL here because a) it's not yet validated, and
	// b) we don't have a scheme
	tus, err := companyConfigStorage.ListTenantURLsForTenant(ctx, tenant.ID)
	if err != nil {
		return "", nil, ucerr.Wrap(err)
	}

	var tenantURL *companyconfig.TenantURL
	for i, tu := range tus {
		u, err := url.Parse(tu.TenantURL)
		if err != nil {
			return "", nil, ucerr.Wrap(err)
		}

		uclog.Debugf(ctx, "checking %v (%v) against %v", u.Hostname(), tu.ID, newHostname)

		if u.Hostname() == newHostname {
			tenantURL = &tus[i]
			tenantURL.Validated = valid
			tenantURL.Active = valid // this particular check is both
			// TODO update SaveTenantURLWithCache after adding cache config to worker
			if err := companyConfigStorage.SaveTenantURL(ctx, tenantURL); err != nil {
				return "", nil, ucerr.Wrap(err)
			}
			break
		}
	}

	if tenantURL == nil {
		return "", nil, ucerr.Errorf("couldn't find tenantURL for %v", newHostname)
	}

	return newHostname, tenantURL, nil
}

func checkCNAME(ctx context.Context, tenant *companyconfig.Tenant, dnsClient dnsclient.Client, newURL string) (bool, string, error) {
	// check DNS points to us (and to correct main tenant URL)
	u, err := url.Parse(newURL)
	if err != nil {
		return false, "", ucerr.Wrap(err)
	}
	newHostname := u.Hostname()
	cnames, err := dnsClient.LookupCNAME(ctx, newHostname)
	if err != nil {
		return false, "", ucerr.Wrap(err)
	}

	uclog.Debugf(ctx, "got %d answers for CNAME lookup of %v", len(cnames), newHostname)

	// look through all the answers for one that points to us
	// seems unusual it wouldn't be just one, but at least one is sufficient for us?
	// 0 could be ok if they didn't set it up in advance
	if len(cnames) > 1 {
		uclog.Warningf(ctx, "got %d answers for CNAME lookup of %v, expected 1", len(cnames), newHostname)
	}

	var valid bool
	for _, cname := range cnames {

		tu, err := url.Parse(tenant.TenantURL)
		if err != nil {
			return false, "", ucerr.Wrap(err)
		}

		if cname != dns.Fqdn(tu.Host) {
			uclog.Warningf(ctx, "expected CNAME target %q, got %q", dns.Fqdn(tu.Hostname()), cname)
		} else {
			valid = true
			break
		}
	}

	return valid, newHostname, nil
}

func createNewACMEOrder(ctx context.Context,
	companyConfigStorage *companyconfig.Storage,
	cfg *acme.Config,
	tenantDB *ucdb.DB,
	tu *companyconfig.TenantURL,
	host string) error {

	uclog.Debugf(ctx, "creating new ACME order for %v (%v)", tu.TenantURL, tu.ID)

	as := acmestorage.New(tenantDB)

	// get acmeapi.RealmClient, account
	rc, acct, err := newClient(ctx, cfg)
	if err != nil {
		return ucerr.Wrap(err)
	}

	// create an order for this domain
	order := acmeapi.Order{
		Identifiers: []acmeapi.Identifier{{
			Type:  "dns",
			Value: host,
		}},
	}

	ucOrder := acmestorage.Order{
		BaseModel:   ucdb.NewBase(),
		Host:        host,
		Status:      acmestorage.OrderStatusNew,
		TenantURLID: tu.ID,
	}
	if err := as.SaveOrder(ctx, &ucOrder); err != nil {
		return ucerr.Wrap(err)
	}

	if err := rc.NewOrder(ctx, acct, &order); err != nil {
		return ucerr.Wrap(err)
	}

	ucOrder.Status = acmestorage.OrderStatusPending
	ucOrder.URL = order.URL
	if err := as.SaveOrder(ctx, &ucOrder); err != nil {
		return ucerr.Wrap(err)
	}

	// an order should set up an authorization request
	if len(order.AuthorizationURLs) < 1 {
		return ucerr.New("no authorizations")
	}
	if len(order.AuthorizationURLs) > 1 {
		uclog.Errorf(ctx, "more than one authorization URL found for order: %v", order.URL)
	}

	az := acmeapi.Authorization{
		URL: order.AuthorizationURLs[0],
	}
	if err := rc.LoadAuthorization(ctx, acct, &az); err != nil {
		return ucerr.Wrap(err)
	}

	if tu.Validated {
		return ucerr.Wrap(createHTTPChallenge(ctx, as, acct, rc, &ucOrder, az))
	}

	return ucerr.Wrap(createDNSChallenge(ctx, companyConfigStorage, as, &ucOrder, cfg, tu, az))
}

func createHTTPChallenge(ctx context.Context,
	as *acmestorage.Storage,
	acct *acmeapi.Account,
	rc *acmeapi.RealmClient,
	ucOrder *acmestorage.Order,
	az acmeapi.Authorization) error {

	// which should have a challenge we can handle
	// if we already validated DNS, we look for http-01 and we'll do the whole
	// thing seamlessly. If not, dns-01 and the customer has to create a TXT record
	// for our CA to validate (and we piggyback on that)
	ch, err := findChallenge(az, "http-01")
	if err != nil {
		return ucerr.Wrap(err)
	}

	// save the token so that plex challenge handler can validate it
	ucOrder.Token = ch.Token
	ucOrder.ChallengeURL = ch.URL // not need today but might as well keep it (used for dns-01 challenges)
	if err := as.SaveOrder(ctx, ucOrder); err != nil {
		return ucerr.Wrap(err)
	}

	if err := rc.RespondToChallenge(ctx, acct, ch, nil); err != nil {
		return ucerr.Wrap(err)
	}

	// now we have to wait for the challenge to be "checked" by the CA,
	// and we'll kick off the next phase from the challenge handler in plex

	return nil
}

func findChallenge(az acmeapi.Authorization, typ string) (*acmeapi.Challenge, error) {
	for i, ch := range az.Challenges {
		if ch.Type == typ {
			return &az.Challenges[i], nil
		}
	}

	return nil, ucerr.Errorf("no %v challenge found for %v", typ, az.URL)
}

func createDNSChallenge(ctx context.Context,
	companyConfigStorage *companyconfig.Storage,
	as *acmestorage.Storage,
	ucOrder *acmestorage.Order,
	cfg *acme.Config,
	tu *companyconfig.TenantURL,
	az acmeapi.Authorization) error {

	uclog.Debugf(ctx, "creating DNS challenge for %v", tu.TenantURL)

	// which should have a challenge we can handle
	// if we couldn't validate DNS, we use dns-01 and the customer has to
	// create a TXT record for our CA to validate (and we piggyback on that)
	ch, err := findChallenge(az, "dns-01")
	if err != nil {
		return ucerr.Wrap(err)
	}

	ucOrder.ChallengeURL = ch.URL
	if err := as.SaveOrder(ctx, ucOrder); err != nil {
		return ucerr.Wrap(err)
	}

	pk, err := cfg.PrivateKey.Resolve(context.Background())
	if err != nil {
		return ucerr.Wrap(err)
	}

	tb, err := acme.ComputeThumbprint(context.Background(), pk)
	if err != nil {
		return ucerr.Wrap(err)
	}

	// https://www.rfc-editor.org/rfc/rfc8555.html#section-8.4
	keyAuth := fmt.Sprintf("%s.%s", ch.Token, tb)
	hash := sha256.Sum256([]byte(keyAuth))

	tu.DNSVerifier = base64.RawURLEncoding.EncodeToString(hash[:])
	// TODO update SaveTenantURLWithCache after adding cache config to worker
	if err := companyConfigStorage.SaveTenantURL(ctx, tu); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}
