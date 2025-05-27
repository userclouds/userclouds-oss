package main

// TODO: there are two assumptions baked in here about how we advance the pager,
// and how we name paginated methods. In a perfect world we'd actually pull that
// code into this test, but that's way more complex so punting that for now.

type pager struct{}

func (p *pager) AdvanceCursor(fields []string) {
}

func ListObjectsPaginated() int {
	return 1
}

type storage struct {
}

func (s *storage) ListObjectsPaginated() int {
	return 1
}

func Foo() {
	p := &pager{}

	for { // want `found a naked for loop with pagination but no advance`
		ListObjectsPaginated()
	}

	s := storage{}
	for { // want `found a naked for loop with pagination but no advance`
		s.ListObjectsPaginated()
	}

	for {
		ListObjectsPaginated()
		p.AdvanceCursor(nil)
	}

	for {
		s.ListObjectsPaginated()
		p.AdvanceCursor(nil)
	}
}
