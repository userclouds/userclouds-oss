package service

// Service represents a service
type Service string

// SDK service represent SDK`
const SDK Service = "sdk"

// IsValid validates a service name
func IsValid(service Service) bool {
	if service == SDK {
		return true
	}

	return false
}
