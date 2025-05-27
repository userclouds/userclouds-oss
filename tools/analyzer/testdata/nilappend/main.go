package nilappend

func test() {
	// Should flag - unnecessary nil check before append
	var opts []string
	if opts == nil { // want `unnecessary nil check: append can handle nil slices`
		opts = make([]string, 0)
	}
	opts = append(opts, "foo")

	// Should flag - nil check with reversed condition
	var opts2 []string
	if nil == opts2 { // want `unnecessary nil check: append can handle nil slices`
		opts2 = make([]string, 0)
	}
	opts2 = append(opts2, "foo")

	// Should not flag - no nil check
	var good1 []string
	good1 = append(good1, "foo")

	// Should not flag - nil check but no make
	var good2 []string
	if good2 == nil {
		good2 = append(good2, "foo")
	}

	// Should not flag - nil check but no append
	var good3 []string
	if good3 == nil {
		good3 = make([]string, 0)
	}
	good3 = good3

	// Should not flag - different variable
	var good4 []string
	if good4 == nil {
		other := make([]string, 0)
		_ = other
	}
	good4 = append(good4, "foo")
}
