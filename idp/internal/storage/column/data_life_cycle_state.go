package column

import (
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/ucerr"
)

// DataLifeCycleState is an internal enum for supported data lifecycle states
type DataLifeCycleState int

// DataLifeCycleState constants
const (
	DataLifeCycleStateLive        DataLifeCycleState = 1
	DataLifeCycleStateSoftDeleted DataLifeCycleState = 2
)

// Validate implements Validateable
func (dlcs DataLifeCycleState) Validate() error {
	switch dlcs {
	case DataLifeCycleStateLive:
	case DataLifeCycleStateSoftDeleted:
	default:
		return ucerr.Friendlyf(nil, "Invalid DataLifeCycleState: %d", dlcs)
	}

	return nil
}

// DataLifeCycleStateFromClient converts a client data lifecycle state to the internal representation
func DataLifeCycleStateFromClient(dlcs userstore.DataLifeCycleState) DataLifeCycleState {
	if dlcs.GetConcrete() == userstore.DataLifeCycleStateSoftDeleted {
		return DataLifeCycleStateSoftDeleted
	}

	return DataLifeCycleStateLive
}

// ToClient converts the internal data lifecycle state to the client representation
func (dlcs DataLifeCycleState) ToClient() userstore.DataLifeCycleState {
	if dlcs == DataLifeCycleStateSoftDeleted {
		return userstore.DataLifeCycleStateSoftDeleted
	}

	return userstore.DataLifeCycleStateLive
}
