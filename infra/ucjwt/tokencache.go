package ucjwt

import (
	"net/url"
	"sync"

	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucerr"
)

// CachedTokenSource adds a cache around ClientCredentialsTokenSource
type CachedTokenSource struct {
	cacheAccessMutex  sync.RWMutex
	cachedToken       string
	cachedTokenSource *oidc.ClientCredentialsTokenSource
}

// NewCachedTokenSource returns a new token cache that is initialized with a token source
func NewCachedTokenSource(tokenURL string, clientID string, clientSecret string, customAudiences []string) (*CachedTokenSource, error) {
	var c CachedTokenSource
	c.cachedToken = ""
	c.cachedTokenSource = nil

	if err := c.setTokenSource(tokenURL, clientID, clientSecret, customAudiences); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &c, nil
}

// GetToken returns the bearer token or an error otherwise.
func (c *CachedTokenSource) GetToken() (string, error) {
	// Check if there is a token in the cache
	c.cacheAccessMutex.RLock()
	token := c.cachedToken
	c.cacheAccessMutex.RUnlock()
	var err error
	if token != "" {
		// If there is a token in the cache make sure it is not expired.
		if e, err := IsExpired(token); err != nil || !e {
			return token, nil
		}
	}
	// Else we either have no token or it is expired
	c.cacheAccessMutex.RLock()
	tokenSource := c.cachedTokenSource
	c.cacheAccessMutex.RUnlock()

	// If we don't have a token source create one and store it in the map
	if tokenSource == nil {
		tokenSource, err = c.getTokenSource()
		// If we can't get a token source it means we are not fully initialized so return a blank token
		if err != nil || tokenSource == nil {
			return "", nil
		}
		c.cacheAccessMutex.Lock()
		c.cachedTokenSource = tokenSource
		c.cacheAccessMutex.Unlock()
	}

	if token, err = tokenSource.GetToken(); err != nil {
		return "", ucerr.Wrap(err)
	}

	// Store the newly acquired token
	c.cacheAccessMutex.Lock()
	c.cachedToken = token
	c.cacheAccessMutex.Unlock()

	return token, nil
}

// setTokenSource allows initialization of a token source for a particular tenant
func (c *CachedTokenSource) setTokenSource(tokenURL string, clientID string, clientSecret string, customAudiences []string) error {

	var tokenSource oidc.ClientCredentialsTokenSource

	tokenEndpointURL, err := url.Parse(tokenURL)
	if err != nil {
		return ucerr.Wrap(err)
	}
	// TODO: move common routes into constants
	tokenEndpointURL.Path = "/oidc/token"

	tokenSource.TokenURL = tokenEndpointURL.String()
	tokenSource.ClientID = clientID
	tokenSource.ClientSecret = clientSecret
	tokenSource.CustomAudiences = customAudiences

	c.cacheAccessMutex.Lock()
	c.cachedTokenSource = &tokenSource
	c.cacheAccessMutex.Unlock()
	return nil
}

func (c *CachedTokenSource) getTokenSource() (*oidc.ClientCredentialsTokenSource, error) {
	return nil, nil
}
