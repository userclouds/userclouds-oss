package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-http-utils/headers"

	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/auth/m2m"
	"userclouds.com/internal/companyconfig"
)

type employeeTokenSource struct {
	tenantURL   string
	employeeJWT string
	accessToken string
}

// NewEmployeeTokenSource creates a new JSON client option which is used to auto-create an access token to another service using M2M auth
func NewEmployeeTokenSource(ctx context.Context, tenant *companyconfig.Tenant, employeeJWT string) (jsonclient.Option, error) {
	accessToken, err := m2m.GetM2MSecretAuthHeader(ctx, tenant.ID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return jsonclient.TokenSource(employeeTokenSource{
		tenantURL:   tenant.TenantURL,
		employeeJWT: employeeJWT,
		accessToken: accessToken,
	}), nil
}

func (e employeeTokenSource) GetToken() (string, error) {
	query := url.Values{}
	query.Add("grant_type", "client_credentials")
	query.Add("subject_jwt", e.employeeJWT)

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/oidc/employeetoken", e.tenantURL), strings.NewReader(query.Encode()))
	if err != nil {
		return "", ucerr.Wrap(err)
	}
	req.Header.Add(headers.ContentType, "application/x-www-form-urlencoded")
	req.Header.Add(headers.Authorization, e.accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", ucerr.Wrap(err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		var oauthe ucerr.OAuthError
		if resp.Header.Get(headers.ContentType) == "application/json" {
			if err := json.NewDecoder(resp.Body).Decode(&oauthe); err != nil {
				return "", ucerr.Wrap(err)
			}

			oauthe.Code = resp.StatusCode
			return "", ucerr.Wrap(oauthe)
		}
		// Handle non-json response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", ucerr.Errorf("unexpected response from token endpoint %v: %v. Failed to read response body: %v", req.URL, resp.Status, err)
		}
		return "", ucerr.Errorf("unexpected response from token endpoint %v: %v: %v", req.URL, resp.Status, string(body))

	}
	var tresp oidc.TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tresp); err != nil {
		return "", ucerr.Wrap(err)
	}
	return tresp.AccessToken, nil
}
