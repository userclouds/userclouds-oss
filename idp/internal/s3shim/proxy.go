package s3shim

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/gofrs/uuid"

	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/internal/userstore"
	"userclouds.com/infra/cache"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/auth"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/tenantmap"
)

func copyHeader(dst, src http.Header, bodyLength int) {
	for k, vv := range src {
		if k == "Content-Length" {
			dst.Set("Content-Length", fmt.Sprintf("%d", bodyLength))
			continue
		}
		if k == "X-Forwarded-For" || k == "X-Forwarded-Host" || k == "X-Forwarded-Proto" {
			continue
		}
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

type proxy struct {
	port                 int
	companyConfigStorage *companyconfig.Storage
	tm                   *tenantmap.StateMap
	cacheConfig          *cache.Config
	jwtVerifier          auth.Verifier
}

var tokenRegex = regexp.MustCompile(`(?i)jwtstart(.*)jwtend/`)
var keyValueRegex = regexp.MustCompile(`(?i)credential=([^\s/]+)`)

// Error is a struct for marshalling error messages to XML
type Error struct {
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}

func getFriendlyXMLError(err error, code, message string, args ...any) error {
	out, marshalErr := xml.Marshal(Error{Code: code, Message: fmt.Sprintf(message, args...)})
	if marshalErr != nil {
		out = []byte{}
	}
	return ucerr.Friendlyf(err, `<?xml version="1.0" encoding="UTF-8" ?>%s`, string(out))
}

func (p *proxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	uclog.Verbosef(ctx, "Incoming S3 shim request: %s %s\nHeaders: %s", req.Method, req.URL.String(), req.Header)
	rawPath := req.URL.EscapedPath()
	// We expect a rooted relative path at this point, e.g. /<uuid>[/remainder]
	if !strings.HasPrefix(rawPath, "/") {
		uchttp.Error(ctx, w, getFriendlyXMLError(nil, "BadRequest", "expected rooted subpath: '%s'", rawPath), http.StatusBadRequest)
		return
	}

	// Split path into the first path segment and the remainder.
	parts := strings.SplitN(strings.TrimPrefix(rawPath, "/"), "/", 2)
	if len(parts) != 2 {
		uchttp.Error(ctx, w, getFriendlyXMLError(nil, "BadRequest", "invalid s3shim path: '%s'", rawPath), http.StatusBadRequest)
		return
	}
	id, err := uuid.FromString(parts[0])
	if err != nil {
		uchttp.Error(ctx, w, getFriendlyXMLError(err, "BadRequest", "no UUID found in path '%s'", rawPath), http.StatusBadRequest)
		return
	}
	path := parts[1]

	ts := multitenant.MustGetTenantState(ctx)
	configStorage := storage.NewFromTenantState(ctx, ts)
	objectStoreInfo, err := configStorage.GetShimObjectStore(ctx, id)
	if err != nil {
		uchttp.Error(ctx, w, getFriendlyXMLError(err, "BadRequest", "failed to get object store info: %s", ucerr.UserFriendlyMessage(err)), http.StatusBadRequest)
		return
	}

	// Check the request method
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		uchttp.Error(ctx, w, getFriendlyXMLError(nil, "NotAllowed", "only GET and HEAD methods allowed"), http.StatusMethodNotAllowed)
		return
	}

	jwt := ""

	// First try to get the JWT token from the URL
	tokenMatch := tokenRegex.FindStringSubmatch(path)
	if len(tokenMatch) == 2 {
		jwt = tokenMatch[1]
		path = tokenRegex.ReplaceAllString(path, "")
	} else {
		// Then try the credential from the Authorization header
		credentialMatch := keyValueRegex.FindStringSubmatch(req.Header.Get("Authorization"))
		if len(credentialMatch) == 2 {
			jwt = credentialMatch[1]
		}
	}

	var accessKeyID, secretAccessKey, sessionToken string

	// If the object store has a role ARN, try to use the jwt to assume the role
	if objectStoreInfo.RoleARN != "" && jwt != "" {
		stsClient := sts.NewFromConfig(aws.Config{
			Region: objectStoreInfo.Region,
		})
		if output, err := stsClient.AssumeRoleWithWebIdentity(ctx, &sts.AssumeRoleWithWebIdentityInput{
			RoleArn:          aws.String(objectStoreInfo.RoleARN),
			RoleSessionName:  aws.String("s3shim"),
			WebIdentityToken: aws.String(jwt),
		}); err != nil {
			if objectStoreInfo.AccessKeyID == "" {
				uchttp.Error(ctx, w, getFriendlyXMLError(nil, "BadRequest", "unable to assume role from OIDC token"), http.StatusBadRequest)
				return
			}
		} else {
			accessKeyID = *output.Credentials.AccessKeyId
			secretAccessKey = *output.Credentials.SecretAccessKey
			sessionToken = *output.Credentials.SessionToken
		}
	}

	// If the caller didn't pass in a JWT, or if the JWT passed in didn't contain valid credentials, use the object store credentials (if available)
	if accessKeyID == "" && objectStoreInfo.AccessKeyID != "" {
		accessKeyID = objectStoreInfo.AccessKeyID
		secretAccessKey, err = objectStoreInfo.SecretAccessKey.Resolve(ctx)
		if err != nil {
			uchttp.Error(ctx, w, getFriendlyXMLError(err, "InternalServerError", "failed to resolve secret access key: %s", ucerr.UserFriendlyMessage(err)), http.StatusInternalServerError)
			return
		}
	}

	// If we still have no credentials, return an error
	if accessKeyID == "" {
		uchttp.Error(ctx, w, getFriendlyXMLError(nil, "BadRequest", "invalid or no credentials provided"), http.StatusBadRequest)
		return
	}

	// Check if the user has permission to access the bucket and object
	controller := userstore.NewIdpS3ShimController(ts, p.jwtVerifier, p.cacheConfig, objectStoreInfo)
	if ok, err := controller.CheckPermission(ctx, jwt, path); err != nil {
		uchttp.Error(ctx, w, getFriendlyXMLError(err, "InternalServerError", "failed to check permission: %s", ucerr.UserFriendlyMessage(err)), http.StatusInternalServerError)
		return
	} else if !ok {
		uchttp.Error(ctx, w, getFriendlyXMLError(nil, "Forbidden", "permission denied"), http.StatusForbidden)
		return
	}

	// Create a new request based on the incoming request
	awsURL := fmt.Sprintf("https://s3.%s.amazonaws.com/%s", objectStoreInfo.Region, path)
	reqAWS, err := http.NewRequestWithContext(ctx, req.Method, awsURL, req.Body)
	if err != nil {
		uchttp.Error(ctx, w, getFriendlyXMLError(err, "BadRequest", "invalid request: %s", ucerr.UserFriendlyMessage(err)), http.StatusBadRequest)
		return
	}
	reqAWS.Host = req.URL.Host
	copyHeader(reqAWS.Header, req.Header, 0)
	t := time.Now().UTC()
	reqAWS.Header.Set("X-Amz-Date", t.Format("20060102T150405Z"))

	// Sign the request
	payload := reqAWS.Header.Get("X-Amz-Content-SHA256")
	signer := v4.NewSigner(func(signer *v4.SignerOptions) {
		signer.DisableURIPathEscaping = true
	})
	if err := signer.SignHTTP(ctx, aws.Credentials{
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		SessionToken:    sessionToken,
	}, reqAWS, payload, "s3", objectStoreInfo.Region, t); err != nil {
		uchttp.Error(ctx, w, getFriendlyXMLError(err, "InternalServerError", "failed to sign request: %s", ucerr.UserFriendlyMessage(err)), http.StatusInternalServerError)
		return
	}

	client := &http.Client{}
	resp, err := client.Do(reqAWS)
	if err != nil {
		uchttp.Error(ctx, w, getFriendlyXMLError(err, "InternalServerError", "failed to send request: %s", ucerr.UserFriendlyMessage(err)), http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			uclog.Errorf(ctx, "failed to close response body: %v", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		uchttp.Error(ctx, w, getFriendlyXMLError(err, "InternalServerError", "failed to read response body: %s", ucerr.UserFriendlyMessage(err)), http.StatusInternalServerError)
		return
	}

	transformed, err := controller.TransformData(ctx, body)
	if err != nil {
		uchttp.Error(ctx, w, getFriendlyXMLError(err, "InternalServerError", "failed to transform data: %s", ucerr.UserFriendlyMessage(err)), http.StatusInternalServerError)
		return
	}

	// Create the response headers (with new body length), then write the headers and body
	copyHeader(w.Header(), resp.Header, len(transformed))
	uclog.Verbosef(ctx, "Outgoing headers: %v", resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, bytes.NewReader(transformed))
}

// RunNewProxy starts a new s3 proxy server on the given port with the given region and credentials
func RunNewProxy(ctx context.Context, port int, tenants *tenantmap.StateMap, cacheConfig *cache.Config, jwtVerifier auth.Verifier, companyConfigStorage *companyconfig.Storage) error {
	handler := &proxy{port: port, tm: tenants, cacheConfig: cacheConfig, jwtVerifier: jwtVerifier, companyConfigStorage: companyConfigStorage}
	addr := fmt.Sprintf("0.0.0.0:%d", port)

	go func() {
		uclog.Infof(ctx, "Starting s3shim proxy server on %s", addr)
		if err := http.ListenAndServe(addr, handler); err != nil {
			uclog.Errorf(ctx, "proxy server closed with error: %v", err)
		}
	}()

	return nil
}

// NewProxy creates a new s3 proxy handler
func NewProxy(tenants *tenantmap.StateMap, cacheConfig *cache.Config, jwtVerifier auth.Verifier, companyConfigStorage *companyconfig.Storage) http.Handler {
	return &proxy{tm: tenants, cacheConfig: cacheConfig, jwtVerifier: jwtVerifier, companyConfigStorage: companyConfigStorage}
}
