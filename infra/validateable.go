package infra

// Validateable defines an interface that lets us validate things
type Validateable interface {
	Validate() error
}
