package secret

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucerr"
)

// cache is an in-memory cache of secrets
type cache struct {
	secrets      map[string]cacheObject
	secretsMutex sync.RWMutex
}

type cacheObject struct {
	Secret  string
	Expires time.Time
}

var c = &cache{secrets: map[string]cacheObject{}}
var secretCacheDuration = time.Hour * 24

// UIPlaceholder is a placeholder for UIs to use when displaying secrets
var UIPlaceholder = String{location: "********"}

// EmptyString is a secret.String that is empty (for UIs mostly)
var EmptyString = String{location: ""}

// Prefix is just a type alias to make some automation easier
type Prefix string

//go:generate genconstant Prefix

// Matches is a poorly named function to check if a string starts with a prefix
func (p Prefix) Matches(s string) bool {
	return strings.HasPrefix(s, string(p))
}

// Value gets the non-prefixed value of a string
func (p Prefix) Value(s string) string {
	return strings.TrimPrefix(s, string(p))
}

// String is a string value that is potentially secured by AWS Secret Manager
type String struct {
	location string // the location, which may be the secret or a prefixed pointer
}

// NewString returns a new secret.String that is stored "correctly" according to
// the environment. Specifically in prod/staging we use AWS, but elsewhere we use Dev
func NewString(ctx context.Context, serviceName, name, secret string) (*String, error) {
	uv := universe.Current()
	switch uv {
	case universe.OnPrem: // for now we will use AWS secrets manager for on-prem, but we may want to be more flexible here down the line.
		fallthrough
	case universe.Staging:
		fallthrough
	case universe.Prod:
		return newAWSString(ctx, uv, serviceName, name, secret)

	case universe.Debug:
		fallthrough
	case universe.CI:
		fallthrough
	case universe.Test:
		fallthrough
	case universe.Container:
		fallthrough
	case universe.Dev:
		s := newDevString(secret)
		return &s, nil
	}
	return nil, ucerr.Errorf("undefined universe %v", uv)
}

func prefixForCurrentUniverse() string {
	switch universe.Current() {
	case universe.Staging:
		fallthrough
	case universe.Prod:
		return string(PrefixAWS)

	case universe.Debug:
		fallthrough
	case universe.CI:
		fallthrough
	case universe.Test:
		fallthrough
	case universe.Container:
		fallthrough
	case universe.Dev:
		return string(PrefixDevLiteral) // FIXME
	}

	return ""
}

func getSecretPath(uv universe.Universe, serviceName, name string) string {
	if !uv.IsOnPrem() {
		return fmt.Sprintf("%s/%s/%s", uv, serviceName, name)
	}
	// on-prem secrets are stored in a different path, since we want to allow customers to create IAM policies that restrict access to secrets
	// for the UC SW running in their environment
	return fmt.Sprintf("userclouds/%s/%s/%s", uv, serviceName, name)
}

// LocationFromName returns a full secret name/location with the correct universe formatting
// Prefixed with `userclouds` for our on-prem usage to allow us to namespace in customer SM
// TODO (sgarrity 8/24): we should use this naming consistently across our secrets usage
func LocationFromName(serviceName, name string) string {
	prefix := prefixForCurrentUniverse()
	// NB that prefixes end in / so we don't need to add another separator here
	return fmt.Sprintf("%s%s", prefix, getSecretPath(universe.Current(), serviceName, name))
}

// FromLocation returns a new secret.String with the specified location
func FromLocation(location string) *String {
	return &String{location: location}
}

// NewTestString returns a string that is *not* stored in AWS Secret Manager
func NewTestString(s string) String {
	return String{location: fmt.Sprintf("%s%s", PrefixDevLiteral, s)}
}

// Resolve decides if the string is a Secret Store path and resolves it,
// or returns the string unchanged otherwise.
func (s *String) Resolve(ctx context.Context) (string, error) {
	c.secretsMutex.RLock()
	co, ok := c.secrets[s.location]
	c.secretsMutex.RUnlock()
	if ok {
		// check if we need to invalidate this
		if time.Now().UTC().Before(co.Expires) {
			return co.Secret, nil
		}
	}

	var value string

	// if uri starts with AWSPrefix, try to resolve it with AWS
	// TODO: we could make this more generic with a lookup table etc down the road
	if strings.HasPrefix(s.location, string(PrefixAWS)) {
		secretName := PrefixAWS.Value(s.location)

		// Resolve using AWS Secret Manager ... note that because we don't have context
		// or anything else during config load, getSecret will use the AWS creds from env
		// vars (or fail)
		secretData, err := getAWSSecret(ctx, secretName)
		if err != nil {
			return "", ucerr.Wrap(err)
		}

		// decode AWS's JSON wrapper if necessary
		var awsSec awsSecret
		if err := json.Unmarshal([]byte(secretData), &awsSec); err == nil {
			value = awsSec.String
		} else {
			value = secretData
		}
	} else if PrefixDev.Matches(s.location) {
		secretData, err := getDevSecret(s.location)
		if err != nil {
			return "", ucerr.Wrap(err)
		}

		value = secretData
	} else if PrefixDevLiteral.Matches(s.location) {
		val := PrefixDevLiteral.Value(s.location)
		value = val
	} else if s.location == "" {
		// empty secrets are allowed (because they're required for local DB passwords)
		empty := ""
		value = empty
	} else if PrefixEnv.Matches(s.location) {
		secretData, err := getEnvVariableSecret(s.location)
		if err != nil {
			return "", ucerr.Wrap(err)
		}
		value = secretData
	} else {
		// return "", ucerr.Errorf("Unknown or missing secret.String prefix: %s", s.location)
		value = s.location
	}

	c.secretsMutex.Lock()
	c.secrets[s.location] = cacheObject{
		Secret:  value,
		Expires: time.Now().UTC().Add(secretCacheDuration),
	}
	c.secretsMutex.Unlock()

	return value, nil
}

// ResolveForUI simplifies the logic (slightly) for UIs to display secrets
func (s *String) ResolveForUI(ctx context.Context) (*String, error) {
	secret, err := s.Resolve(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	if secret == "" {
		return &EmptyString, nil
	}

	return &UIPlaceholder, nil
}

// ResolveInsecurelyForUI is currently used for login app client secrets only,
// when the user actually does need to see them in the UI (rather than just update them)
func (s *String) ResolveInsecurelyForUI(ctx context.Context) (*String, error) {
	secret, err := s.Resolve(ctx)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &String{location: secret}, nil
}

// Delete removes the secret from the secret store (if applicable) and clears the location
func (s *String) Delete(ctx context.Context) error {
	var err error
	if strings.HasPrefix(s.location, string(PrefixAWS)) {
		sec := strings.TrimPrefix(s.location, string(PrefixAWS))
		err = deleteAWSSecret(ctx, sec)
	}

	s.location = ""
	return ucerr.Wrap(err)
}

// UnmarshalYAML implements yaml.Unmarshaler
// Note like UnmarshalText we assume this is a location, and
// we'll lazily resolve it later as needed
func (s *String) UnmarshalYAML(unmarshal func(any) error) error {
	// the secret path itself is just a string, so start there
	var uri string
	if err := unmarshal(&uri); err != nil {
		return ucerr.Wrap(err)
	}
	s.location = uri
	return nil
}

// MarshalText implements encoding.TextMarshaler
// NB: we don't implement MarshalJSON because we intentionally *don't* want
// to emit a rich object here (for backcompat, and no need)
func (s String) MarshalText() ([]byte, error) {
	// we always save location since it's either the pointer we
	// want to save, or it's a copy of .value anyway
	return []byte(s.location), nil
}

// UnmarshalText implements json.Unmarshaler
// We always assume this is a location, and we'll lazily resolve
// the secret later in Resolve() as needed
func (s *String) UnmarshalText(b []byte) error {
	s.location = string(b)
	return nil
}

// String implements Stringer, specifically to obscure secrets when logged
// To actually use a secret, you need to explicitly use Resolve()
func (s String) String() string {
	return strings.Repeat("*", len(s.location))
}

// Validate implements Validateable
func (s String) Validate() error {
	// empty secrets are ok
	if s.IsEmpty() {
		return nil
	}
	if s.hasValidPrefix() {
		return nil
	}

	// passthrough secrets are ok again for this migration
	return nil

	// return ucerr.Errorf("secret.String.Validate unrecognized prefix for %s", s.location)
}

// IsEmpty checks if the secret.String location is empty
func (s String) IsEmpty() bool {
	return s.location == ""

}

// hasValidPrefix checks if the secret.String has a valid prefix
func (s String) hasValidPrefix() bool {
	for _, p := range AllPrefixes {
		if p.Matches(s.location) {
			return true
		}
	}
	return false
}

// Scan implements sql.Scanner
func (s *String) Scan(value any) error {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case string:
		s.location = v
		return nil
	default:
		return ucerr.Errorf("cannot scan %T into secret.String", value)
	}
}

// Value implements sql.Valuer
func (s String) Value() (driver.Value, error) {
	return s.location, nil
}

// UpdateFromClient updates the secret.String in-place from a client-provided value, iff necessary
func (s *String) UpdateFromClient(ctx context.Context, serviceName, name string) error {
	// if this just came back with a secret location, we're good
	// NB: this makes an assumption that client should never change the location, but that seems right
	// NBB: this also assumes that secrets are never prefixed with our prefixes, which seems ... reasonable for now?
	if s.hasValidPrefix() {
		return nil
	}

	// if it's not a valid prefix, we assume it's a secret itself
	// and we need to store it safely
	sec := s.location
	secString, err := NewString(ctx, serviceName, name, sec)
	if err != nil {
		return ucerr.Wrap(err)
	}

	*s = *secString
	return nil
}
