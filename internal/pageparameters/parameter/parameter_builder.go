package parameter

// Builder is a helper class for building up a collection of parameters
type Builder struct {
	parameters []Parameter
}

// NewBuilder return a new instance of a Builder
func NewBuilder() *Builder {
	return &Builder{parameters: []Parameter{}}
}

// AddParameter will make a new parameter instance for a specified parameter
// name and its associated parameter type, and the specified value
func (b *Builder) AddParameter(n Name, v string) *Builder {
	b.parameters = append(b.parameters, MakeParameter(n, v))
	return b
}

// Build returns a copy of the collection of parameters that have been created
func (b *Builder) Build() (p []Parameter) {
	return append(p, b.parameters...)
}
