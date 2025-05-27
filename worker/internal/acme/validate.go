package acme

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/gofrs/uuid"
	"github.com/hlandau/acmeapi"

	"userclouds.com/infra/acme"
	"userclouds.com/infra/dnsclient"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/workerclient"
	"userclouds.com/internal/acmestorage"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/worker"
)

const maxTries = 5

const maxDNSTries = 10
const dnsRetryWait = time.Second * 30

// ValidateTenantURL just tells LetsEncrypt to check the DNS for a dns-01 challenge
func ValidateTenantURL(ctx context.Context,
	dnsClient dnsclient.Client,
	acmeCfg *acme.Config,
	companyConfigStorage *companyconfig.Storage,
	tenantID uuid.UUID,
	newURL string,
	tenantDB *ucdb.DB,
	wc workerclient.Client) error {

	u, err := url.Parse(newURL)
	if err != nil {
		return ucerr.Wrap(err)
	}
	host := u.Hostname()

	uclog.Infof(ctx, "validating DNS challenge for %v", newURL)

	// load the tenant URL
	tenantURL, err := companyConfigStorage.GetTenantURLByURL(ctx, newURL)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if tenantURL.TenantID != tenantID {
		return ucerr.Errorf("tenant ID mismatch in validateDNSChallenge: %v and %v", tenantURL.TenantID, tenantID)
	}

	as := acmestorage.New(tenantDB)

	// load the ucOrder object
	ucOrders, err := as.ListOrdersByTenantURLID(ctx, tenantURL.ID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	var ucOrder *acmestorage.Order
	for i, o := range ucOrders {
		if o.Status == acmestorage.OrderStatusPending {
			if ucOrder != nil {
				uclog.Errorf(ctx, "found at least two orders pending for host %s: %v and %v", o.Host, ucOrder.ID, o.ID)
			}
			ucOrder = &ucOrders[i]
		}
	}
	if ucOrder == nil {
		return ucerr.Errorf("no pending order found for host %s", newURL)
	}

	acmeChallengeHost := fmt.Sprintf("_acme-challenge.%s", host)

	// first let's make sure we can find it, since DNS can be slow
	var i int
	for {
		uclog.Infof(ctx, "checking for TXT record for %v: try %d", acmeChallengeHost, i)
		err := checkForTXTRecord(ctx, dnsClient, acmeChallengeHost, tenantURL.DNSVerifier)
		if err == nil {
			break
		}

		uclog.Warningf(ctx, "error checking for TXT record try %d for %v: %v", i, acmeChallengeHost, err)

		if i < maxDNSTries {
			i++
			time.Sleep(dnsRetryWait)
			continue
		}

		return ucerr.Wrap(err)
	}

	// double check (without actually locking) that we're not racing
	tu, err := companyConfigStorage.GetTenantURL(ctx, tenantURL.ID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if tu.Validated {
		// another worker validated it
		return nil
	}

	// record that it's validated
	tenantURL.Validated = true
	// TODO update SaveTenantURLWithCache after adding cache config to worker
	if err := companyConfigStorage.SaveTenantURL(ctx, tenantURL); err != nil {
		return ucerr.Wrap(err)
	}

	// now we'll tell LE to check it

	// get acmeapi.RealmClient, account
	rc, acct, err := newClient(ctx, acmeCfg)
	if err != nil {
		return ucerr.Wrap(err)
	}

	ch := acmeapi.Challenge{
		URL: ucOrder.ChallengeURL,
	}
	if err := rc.RespondToChallenge(ctx, acct, &ch, nil); err != nil {
		return ucerr.Wrap(err)
	}

	// now we have to wait for the challenge to be "checked" by the CA
	// we'll check the order status
	acmeOrder := acmeapi.Order{URL: ucOrder.URL}
	for i := range maxTries {
		uclog.Debugf(ctx, "waiting for order %v to be ready: try %d", acmeOrder.URL, i)

		if err := rc.WaitLoadOrder(ctx, acct, &acmeOrder); err != nil {
			return ucerr.Wrap(err)
		}

		if acmeOrder.Status == acmeapi.OrderReady {
			break
		}
	}

	if acmeOrder.Status == acmeapi.OrderReady {
		tenantURL.Validated = true
		// TODO update SaveTenantURLWithCache after adding cache config to worker
		if err := companyConfigStorage.SaveTenantURL(ctx, tenantURL); err != nil {
			return ucerr.Wrap(err)
		}

		ucOrder.Status = acmestorage.OrderStatusReady
		if err := as.SaveOrder(ctx, ucOrder); err != nil {
			return ucerr.Wrap(err)
		}

		msg := worker.CreateFinalizeTenantCNAMEMessage(tenantID, ucOrder.ID)
		if err := wc.Send(ctx, msg); err != nil {
			return ucerr.Wrap(err)
		}
		return nil
	}

	return ucerr.Errorf("failed to load ready order: UC %v, ACME %v", ucOrder.ID, acmeOrder.URL)
}

func checkForTXTRecord(ctx context.Context, dnsClient dnsclient.Client, host, expected string) error {
	txts, err := dnsClient.LookupTXT(ctx, host)
	if err != nil {
		return ucerr.Wrap(err)
	}

	// look through all the answers for one that points to us
	// seems unusual it wouldn't be just one, but at least one is sufficient for us?
	// 0 could be ok if they didn't set it up in advance
	if len(txts) > 1 {
		uclog.Warningf(ctx, "got %d answers for CNAME lookup of %v, expected 1", len(txts), host)
	}

	for _, txt := range txts {
		uclog.Debugf(ctx, "got TXT %v for %v", txt, host)

		if slices.Contains(txt, expected) {
			return nil
		}
	}

	return ucerr.New("no valid TXT record found")
}

// CheckAllCNAMEsHandler sets up a handler to check all CNAMEs for all tenants every X min (from EB cron)
func CheckAllCNAMEsHandler(ccs *companyconfig.Storage, tenantStateMap *tenantmap.StateMap, wc workerclient.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		uclog.SetHandlerName(ctx, "checkcnames")
		if err := dispatchCheckTenantCNAME(ctx, ccs, wc, tenantStateMap); err != nil {
			uchttp.Error(ctx, w, err, http.StatusInternalServerError)
			return
		}
	}
}

func dispatchCheckTenantCNAME(ctx context.Context, ccs *companyconfig.Storage, wc workerclient.Client, tenantStateMap *tenantmap.StateMap) error {
	pager, err := companyconfig.NewTenantURLPaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return ucerr.Wrap(err)
	}

	for {
		tus, pr, err := ccs.ListTenantURLsPaginated(ctx, *pager)
		if err != nil {
			return ucerr.Wrap(err)
		}

		for _, tu := range tus {
			if tu.System {
				continue
			}
			if _, err := tenantStateMap.GetTenantStateForID(ctx, tu.TenantID); errors.Is(err, sql.ErrNoRows) {
				if err := deleteTenantURL(ctx, ccs, &tu); err != nil {
					uclog.Errorf(ctx, "error deleting tenant URL: %v", err)
				}
				continue
			}
			if err := wc.Send(ctx, worker.CreateCheckTenantCNameMessage(tu.TenantID, tu.ID)); err != nil {
				uclog.Errorf(ctx, "error sending check tenant CNAME message for %v: %v", err, tu.ID)
			}
		}
		if !pager.AdvanceCursor(*pr) {
			break
		}
	}
	return nil
}

func deleteTenantURL(ctx context.Context, ccs *companyconfig.Storage, tenantURL *companyconfig.TenantURL) error {
	uclog.Infof(ctx, "tenant %v (%v) no longer exists, deleting tenant URL %v", tenantURL.TenantID, tenantURL.TenantURL, tenantURL.ID)
	return ucerr.Wrap(ccs.DeleteTenantURL(ctx, tenantURL.ID))

}

// CheckTenantURL checks to ensure that the tenant URL is still valid (or not)
func CheckTenantURL(ctx context.Context, dnsClient dnsclient.Client, ccs *companyconfig.Storage, tenantID, tenantURLID uuid.UUID) error {
	tenantURL, err := ccs.GetTenantURL(ctx, tenantURLID)
	if err != nil {
		return ucerr.Errorf("error getting tenant URL: %w", err)
	}
	tenant, err := ccs.GetTenant(ctx, tenantURL.TenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if err := deleteTenantURL(ctx, ccs, tenantURL); err != nil {
				return ucerr.Wrap(err)
			}
			return nil
		}
		return ucerr.Wrap(err)
	}

	valid, _, err := checkCNAME(ctx, tenant, dnsClient, tenantURL.TenantURL)
	if err != nil {
		uclog.Errorf(ctx, "error checking tenant URL: %v", err)
	}
	if !valid && tenantURL.Active {
		uclog.Warningf(ctx, "tenant URL %v (%v) was active, and is no longer", tenantURL.TenantURL, tenantURL.ID)
		return nil
	}

	// no need to save if nothing changed
	if tenantURL.Active == valid {
		return nil
	}

	tenantURL.Active = valid
	// TODO update SaveTenantURLWithCache after adding cache config to worker
	if err := ccs.SaveTenantURL(ctx, tenantURL); err != nil {
		return ucerr.Errorf("error saving tenant URL: %w", err)
	}
	return nil
}
