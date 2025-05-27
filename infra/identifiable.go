package infra

import "github.com/gofrs/uuid"

// Identifiable defines an interface that lets us get id of the object
type Identifiable interface {
	GetID() uuid.UUID
}
