package gomysqlserver

import "sync"

// CredentialProvider defines an interface for auth
// interface for user credential provider
// hint: can be extended for more functionality
// =================================IMPORTANT NOTE===============================
// if the password in a third-party credential provider could be updated at runtime, we have to invalidate the caching
// for 'caching_sha2_password' by calling 'func (s *Server)InvalidateCache(string, string)'.
type CredentialProvider interface {
	// get user credential
	GetCredential(username string) (password string, found bool, err error)

	// differing auth methods
	// NB: capability is the client capability flags, and we need it since we proxy the client
	// to the target DB to check the password, and need to forward it in case behavior changes
	// (eg. CLIENT_FOUND_ROWS)
	CheckPassword(username, password string, capability uint32) (bool, error)
}

// NewInMemoryProvider instantiates an InMemoryProvider
func NewInMemoryProvider() *InMemoryProvider {
	return &InMemoryProvider{
		userPool: sync.Map{},
	}
}

// InMemoryProvider implements a in memory credential provider
type InMemoryProvider struct {
	userPool sync.Map // username -> password
}

// GetCredential gets user credential
func (m *InMemoryProvider) GetCredential(username string) (password string, found bool, err error) {
	v, ok := m.userPool.Load(username)
	if !ok {
		return "", false, nil
	}
	return v.(string), true, nil
}

// AddUser adds a user to the in memory provider
func (m *InMemoryProvider) AddUser(username, password string) {
	m.userPool.Store(username, password)
}

// Provider is an alias for InMemoryProvider
type Provider InMemoryProvider

// CheckPassword checks if the password is correct
func (m *InMemoryProvider) CheckPassword(username, password string, capability uint32) (bool, error) {
	v, ok := m.userPool.Load(username)
	if !ok {
		return false, nil
	}
	return v.(string) == password, nil
}
