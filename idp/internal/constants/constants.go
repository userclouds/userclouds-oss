package constants

import (
	"regexp"

	"github.com/gofrs/uuid"
)

// GetUserAccessorID is the ID of the accessor "get user"
var GetUserAccessorID = uuid.Must(uuid.FromString("28bf0486-9eea-4db5-ba40-5cef12dd48db"))

// UpdateUserMutatorID is the ID of the mutator "update user"
var UpdateUserMutatorID = uuid.Must(uuid.FromString("45c07eb8-f042-4cc8-b50e-3e6eb7b34bdc"))

// OperationalPurposeID is the UUID of the operational purpose
var OperationalPurposeID = uuid.Must(uuid.FromString("7f55f479-3822-4976-a8a9-b789d5c6f152"))

// AnalyticsPurposeID is the UUID of the analytics purpose
var AnalyticsPurposeID = uuid.Must(uuid.FromString("1bc65251-1dc3-4993-9d30-92e2593a18ef"))

// MarketingPurposeID is the UUID of the marketing purpose
var MarketingPurposeID = uuid.Must(uuid.FromString("bc8e77f0-3104-4844-8a8c-c791908f947b"))

// SupportPurposeID is the UUID of the support purpose
var SupportPurposeID = uuid.Must(uuid.FromString("8c88cd01-6001-4553-b003-87559f439061"))

// SecurityPurposeID is the UUID of the security purpose
var SecurityPurposeID = uuid.Must(uuid.FromString("3f929a5c-0a3e-4e36-b911-25ff43000bf9"))

// ReferencedColumnRE is a regular expression that matches a referenced column in a user selector config
var ReferencedColumnRE = regexp.MustCompile(`\{([a-zA-Z0-9_-]+)\}(->>'[a-zA-Z0-9_-]+')?`) // keep in sync with lexer.nex
