package main

type base struct{}

func (b base) Validate() error {
	return nil
}

type embed struct {
	base
}

type explicit struct {
	base
}

func (e explicit) Validate() error {
	return e.base.Validate()
}

type container struct {
	e  embed
	e2 explicit

	es  []embed
	e2s []explicit
}

func main() {
	var b base
	b.Validate()

	var e embed
	e.Validate() // want `Validate is calling an embedded method`

	var e2 explicit
	e2.Validate()

	var c container
	// TODO: these test cases don't work (MethodSet is 0-len)
	// c.e.Validate()     // w `Validate is calling an embedded method`
	// c.es[0].Validate() // w `Validate is calling an embedded method`
	c.e2.Validate()
	c.e2s[0].Validate()
}
