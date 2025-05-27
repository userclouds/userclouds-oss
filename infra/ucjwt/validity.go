package ucjwt

// Default constants for token validity
// these are defined as int64s to avoid type conversions (since that's what our JWT lib expects)
// and they are seconds, not time.Duration
const (
	DefaultValidityAccess          int64 = 60 * 60 * 24               // one day
	DefaultValidityRefresh         int64 = DefaultValidityAccess * 30 // one month
	DefaultValidityImpersonateUser int64 = 60 * 60                    // one hour
)
