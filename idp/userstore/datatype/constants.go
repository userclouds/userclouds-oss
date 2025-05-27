package datatype

import (
	"github.com/gofrs/uuid"

	"userclouds.com/idp/userstore"
)

// Birthdate is a resource id for the system birthdate data type
var Birthdate = userstore.ResourceID{
	ID:   uuid.Must(uuid.FromString("76f0685b-dd42-4b3f-8c33-4c72e4eff73e")),
	Name: "birthdate",
}

// Boolean is a resource id for the system boolean data type
var Boolean = userstore.ResourceID{
	ID:   uuid.Must(uuid.FromString("e16b5ead-54db-4b42-a55f-f21907cda9e4")),
	Name: "boolean",
}

// CanonicalAddress is a resource id for the canonical address data type
var CanonicalAddress = userstore.ResourceID{
	ID:   uuid.Must(uuid.FromString("33dc5de6-94b6-4f08-94b6-e04d1f981671")),
	Name: "canonical_address",
}

// Composite is a resource id for the composite concrete data type
var Composite = userstore.ResourceID{
	ID:   uuid.Must(uuid.FromString("d81658a7-848a-4504-9c6e-5fa17f90f1a6")),
	Name: "composite",
}

// Date is a resource id for the system date data type
var Date = userstore.ResourceID{
	ID:   uuid.Must(uuid.FromString("3e2546c0-14d6-49d3-8b95-a5000bb4ad6a")),
	Name: "date",
}

// E164PhoneNumber is a resource id for the system E164 phone number data type
var E164PhoneNumber = userstore.ResourceID{
	ID:   uuid.Must(uuid.FromString("97f0ab8a-f2fd-43da-9feb-3d1f8aacc042")),
	Name: "e164_phonenumber",
}

// Email is a resource id for the system email data type
var Email = userstore.ResourceID{
	ID:   uuid.Must(uuid.FromString("8a84f041-c605-4ebf-b552-9e14f51c9e54")),
	Name: "email",
}

// Integer is a resource id for the system integer data type
var Integer = userstore.ResourceID{
	ID:   uuid.Must(uuid.FromString("22b8a1b6-e5a2-4c3c-9a99-746f0345b727")),
	Name: "integer",
}

// PhoneNumber is a resource id for the system phone number data type
var PhoneNumber = userstore.ResourceID{
	ID:   uuid.Must(uuid.FromString("ae962c31-2ca7-42e1-814b-32e6493dba82")),
	Name: "phonenumber",
}

// SSN is a resource id for the system SSN data type
var SSN = userstore.ResourceID{
	ID:   uuid.Must(uuid.FromString("fba9f9bb-b9e0-4258-9fb8-6777792dbeba")),
	Name: "ssn",
}

// String is a resource id for the system string data type
var String = userstore.ResourceID{
	ID:   uuid.Must(uuid.FromString("d26b6d52-a8d7-4c2f-9efc-394eb90a3294")),
	Name: "string",
}

// Timestamp is a resource id for the system timestamp data type
var Timestamp = userstore.ResourceID{
	ID:   uuid.Must(uuid.FromString("66a87f97-32c4-4ccc-91da-d8c880e21e5a")),
	Name: "timestamp",
}

// UUID is a resource id for the system UUID data type
var UUID = userstore.ResourceID{
	ID:   uuid.Must(uuid.FromString("d036bbba-6012-4d74-b7c4-9a2bbc09a749")),
	Name: "uuid",
}
