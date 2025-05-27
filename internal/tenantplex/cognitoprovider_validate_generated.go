// NOTE: automatically generated file -- DO NOT EDIT

package tenantplex

import (
	"userclouds.com/infra/ucerr"
)

// Validate implements Validateable
func (o CognitoProvider) Validate() error {
	if err := o.AWSConfig.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	if o.UserPoolID == "" {
		return ucerr.Friendlyf(nil, "CognitoProvider.UserPoolID can't be empty")
	}
	return nil
}
