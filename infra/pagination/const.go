package pagination

// DefaultLimit is the default pagination limit, appropriate for a UI query
const DefaultLimit = 50

// DefaultLimitMultiplier is the default pagination limit multiplier
const DefaultLimitMultiplier = 1

// MaxLimit is (at the moment) fairly arbitrarily chosen and limits results in a single call. This protects the server/DB
// from trying to process too much data at once
const MaxLimit = 1500
