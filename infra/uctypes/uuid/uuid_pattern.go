package uuid

// UUIDPattern is a pattern suitable for regexp matching that matches a UUID.
// We accept either a UUID with no hyphens or hyphens in the expected locations.
const UUIDPattern = `[0-9a-fA-F]{32}|[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`
