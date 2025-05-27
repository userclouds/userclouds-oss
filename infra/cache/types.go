package cache

// Key is the type storing the cache key name. It is a string but is a separate type to avoid bugs related to mixing up strings.
type Key string

// Sentinel is the type storing in the cache marker for in progress operation
type Sentinel string

// SentinelType captures the type of the sentinel for different operations
type SentinelType string
