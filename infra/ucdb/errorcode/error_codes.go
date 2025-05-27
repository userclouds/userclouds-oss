package errorcode

import "github.com/lib/pq"

// InvalidTextRepresentation returns the postgres error code for an invalid_text_representation error
func InvalidTextRepresentation() pq.ErrorCode {
	return pq.ErrorCode("22P02")
}

// InvalidDateTimeFormat returns the postgres error code for an invalid_datetime_format error
func InvalidDateTimeFormat() pq.ErrorCode {
	return pq.ErrorCode("22007")
}

// UniqueViolation returns the postgres error code for a unique_violation
func UniqueViolation() pq.ErrorCode {
	return pq.ErrorCode("23505")
}

// TransactionCommit returns the postgres error code for a transaction commit error
func TransactionCommit() pq.ErrorCode {
	return pq.ErrorCode("40001")
}
