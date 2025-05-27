package timestamparray

// this is a simple wrapper around the pq.GenericArray type
// to allow us to use []time.Time as a type in our structs and
// correctly persist them to the database as a postgres array time.Time[]

import (
	"time"
)

// TimestampArray is a DB wrapper type for []time.Time
type TimestampArray []time.Time
