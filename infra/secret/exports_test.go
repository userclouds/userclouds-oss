package secret

// NewStringWithLocation is only exported in testing builds
func NewStringWithLocation(location string) *String {
	return &String{location: location}
}

// ResetCache is only exported in testing builds
func ResetCache() {
	c.secretsMutex.Lock()
	c.secrets = map[string]cacheObject{}
	c.secretsMutex.Unlock()
}
