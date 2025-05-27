package ucaws

import (
	"strings"
)

// these constants are copied from github.com/aws/aws-sdk-go-v2/aws/arn
const (
	arnDelimiter = ":"
	arnSections  = 6
	arnPrefix    = "arn:"
)

// IsValidAwsARN returns whether the given arn is valid
func IsValidAwsARN(arn string) bool {
	if !strings.HasPrefix(arn, arnPrefix) {
		return false
	}
	sections := strings.SplitN(arn, arnDelimiter, arnSections)
	return len(sections) == arnSections
}
