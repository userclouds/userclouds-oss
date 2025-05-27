package uuidarray

// this is a super simple wrapper around the pq.GenericArray type
// to allow us to use []uuid.UUID as a type in our structs and
// correctly persist them to the database as a postgres array UUID[]

import (
	"github.com/gofrs/uuid"
)

// UUIDArray is a DB wrapper type for []uuid.UUID
type UUIDArray []uuid.UUID
