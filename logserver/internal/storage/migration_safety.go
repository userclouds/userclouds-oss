package storage

// This file is a hack until I can fix logserver's db usage patterns
// we let genorm run to generate column mappings to inform our migration safety

// MetricsRowPlex is a placeholder
type MetricsRowPlex struct {
	MetricsRow
}

//go:generate genorm --columnlistonly MetricsRowPlex metrics_plex logdb

// MetricsRowAuthz is a placeholder
type MetricsRowAuthz struct {
	MetricsRow
}

//go:generate genorm --columnlistonly MetricsRowAuthz metrics_authz logdb

// MetricsRowIDP is a placeholder
type MetricsRowIDP struct {
	MetricsRow
}

//go:generate genorm --columnlistonly MetricsRowIDP metrics_idp logdb

// MetricsRowConsole is a placeholder
type MetricsRowConsole struct {
	MetricsRow
}

//go:generate genorm --columnlistonly MetricsRowConsole metrics_console logdb

// MetricsRowTokenizer is a placeholder
type MetricsRowTokenizer struct {
	MetricsRow
}

//go:generate genorm --columnlistonly MetricsRowTokenizer metrics_tokenizer logdb

// MetricsRowCheckAttribute is a placeholder
type MetricsRowCheckAttribute struct {
	MetricsRow
}

//go:generate genorm --columnlistonly MetricsRowCheckAttribute metrics_checkattribute logdb
