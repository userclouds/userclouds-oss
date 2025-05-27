package codegen

import (
	"fmt"
	"slices"
	"strings"
)

// Imports is a simple utility to manage codegen imports
type Imports struct {
	golang   map[string]struct{}
	external map[string]struct{}
	uc       map[string]struct{}
}

// NewImports creates a new Imports struct
func NewImports() *Imports {
	p := &Imports{}
	p.golang = make(map[string]struct{})
	p.external = make(map[string]struct{})
	p.uc = make(map[string]struct{})
	return p
}

// Copy creates a copy of the Imports struct
func (p *Imports) Copy() *Imports {
	n := NewImports()
	for k := range p.golang {
		n.golang[k] = struct{}{}
	}
	for k := range p.external {
		n.external[k] = struct{}{}
	}
	for k := range p.uc {
		n.uc[k] = struct{}{}
	}
	return n
}

// Add adds an import package
func (p *Imports) Add(i string) {
	i = strings.TrimSpace(i)
	i = strings.Trim(i, `"`)
	i = fmt.Sprintf("\t\"%s\"", i)
	if strings.Contains(i, "userclouds.com") {
		p.uc[i] = struct{}{}
	} else if strings.Contains(i, ".") {
		p.external[i] = struct{}{}
	} else {
		p.golang[i] = struct{}{}
	}
}

// String implements Stringer
func (p *Imports) String() string {
	// NB: ordering is important here
	sections := [][]string{
		stringSorter(p.golang),
		stringSorter(p.external),
		stringSorter(p.uc),
	}

	var nonEmptySections []string
	for _, s := range sections {
		if len(s) > 0 {
			nonEmptySections = append(nonEmptySections, strings.Join(s, "\n"))
		}
	}

	return strings.Join(nonEmptySections, "\n\n")
}

func stringSorter(m map[string]struct{}) []string {
	var s []string
	for k := range m {
		s = append(s, k)
	}
	slices.Sort(s)
	return s
}
