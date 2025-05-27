package pagination

// LimitMultiplier specifies a factor to multiply the limit by. If unspecified, the default limit multiplier will be used.
func LimitMultiplier(limitMultiplier int) Option {
	return optFunc(
		func(p *Paginator) {
			p.limitMultiplier = limitMultiplier
		})
}

// ResultType initializes the supported keys for a pagination query from the example result instance.
func ResultType(result PageableType) Option {
	return optFunc(
		func(p *Paginator) {
			p.setResultType(result)
		})
}
