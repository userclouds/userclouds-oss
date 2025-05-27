package dnsclient

import (
	"context"

	"github.com/miekg/dns"

	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

type client struct {
	config *Config
}

// LookupCNAME implements Client
func (c client) LookupCNAME(ctx context.Context, host string) ([]string, error) {
	cl := new(dns.Client)
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(host), dns.TypeCNAME)
	m.RecursionDesired = true // since Cloudflare isn't likely the authoritative server
	res, _, err := cl.Exchange(m, c.config.HostAndPort)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	var cnames []string
	for _, ans := range res.Answer {
		cname, ok := ans.(*dns.CNAME)
		if !ok {
			uclog.Warningf(ctx, "expected CNAME, got %T: %+v", ans, ans)
			continue
		}

		cnames = append(cnames, cname.Target)
	}

	return cnames, nil
}

// LookupTXT implements Client
func (c client) LookupTXT(ctx context.Context, host string) ([][]string, error) {
	cl := new(dns.Client)
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(host), dns.TypeTXT)
	m.RecursionDesired = true // since Cloudflare isn't likely the authoritative server
	res, _, err := cl.Exchange(m, c.config.HostAndPort)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	var txts [][]string // all answers
	for _, ans := range res.Answer {
		txt, ok := ans.(*dns.TXT)
		if !ok {
			uclog.Warningf(ctx, "expected TXT, got %T: %+v", ans, ans)
			continue
		}

		// each answer can have an array of strings
		txts = append(txts, txt.Txt)
	}

	return txts, nil
}
