// NOTE: automatically generated file -- DO NOT EDIT

package storage

import "userclouds.com/infra/ucerr"

// MarshalText implements encoding.TextMarshaler (for JSON)
func (t MFAChallengeState) MarshalText() ([]byte, error) {
	switch t {
	case MFAChallengeStateBadChallenge:
		return []byte("badchallenge"), nil
	case MFAChallengeStateExpired:
		return []byte("expired"), nil
	case MFAChallengeStateIssued:
		return []byte("issued"), nil
	case MFAChallengeStateNoChallenge:
		return []byte("nochallenge"), nil
	default:
		return nil, ucerr.Friendlyf(nil, "unknown MFAChallengeState value '%d'", t)
	}
}

// UnmarshalText implements encoding.TextMarshaler (for JSON)
func (t *MFAChallengeState) UnmarshalText(b []byte) error {
	s := string(b)
	switch s {
	case "badchallenge":
		*t = MFAChallengeStateBadChallenge
	case "expired":
		*t = MFAChallengeStateExpired
	case "issued":
		*t = MFAChallengeStateIssued
	case "nochallenge":
		*t = MFAChallengeStateNoChallenge
	default:
		return ucerr.Friendlyf(nil, "unknown MFAChallengeState value '%s'", s)
	}
	return nil
}

// Validate implements Validateable
func (t *MFAChallengeState) Validate() error {
	switch *t {
	case MFAChallengeStateBadChallenge:
		return nil
	case MFAChallengeStateExpired:
		return nil
	case MFAChallengeStateIssued:
		return nil
	case MFAChallengeStateNoChallenge:
		return nil
	default:
		return ucerr.Friendlyf(nil, "unknown MFAChallengeState value '%d'", *t)
	}
}

// Enum implements Enum
func (t MFAChallengeState) Enum() []any {
	return []any{
		"badchallenge",
		"expired",
		"issued",
		"nochallenge",
	}
}

// AllMFAChallengeStates is a slice of all MFAChallengeState values
var AllMFAChallengeStates = []MFAChallengeState{
	MFAChallengeStateBadChallenge,
	MFAChallengeStateExpired,
	MFAChallengeStateIssued,
	MFAChallengeStateNoChallenge,
}

// just here for easier debugging
func (t MFAChallengeState) String() string {
	bs, err := t.MarshalText()
	if err != nil {
		return err.Error()
	}
	return string(bs)
}
