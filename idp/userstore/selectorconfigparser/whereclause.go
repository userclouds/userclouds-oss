package selectorconfigparser

import (
	"strings"

	"userclouds.com/infra/ucerr"
)

type customLexer struct {
	*Lexer
	ErrorOutput string
}

func (l *customLexer) Error(s string) {
	l.ErrorOutput = s
}

// ParseWhereClause parses a where clause and returns an error if it is invalid
func ParseWhereClause(clause string) error {
	input := strings.NewReader(clause)
	cl := &customLexer{NewLexer(input), ""}
	if yyParse(cl) != 0 {
		return ucerr.Friendlyf(nil, "error parsing where clause \"%s\": %s", clause, cl.ErrorOutput)
	}
	return nil
}

// NOTE: to update the parser after changes the lexer.nex and/or parser.y, do the following:
// (we forked blynn/nex to fix the linter issues ourselves :) )
// 1) export GOPATH=/tmp/go
// 2) go get github.com/userclouds/nex
// 3) go install github.com/userclouds/nex
// 4) go install golang.org/x/tools/cmd/goyacc@master
// 5) bin/nex -o idp/userstore/selectorconfigparser/lexer.go idp/userstore/selectorconfigparser/lexer.nex
// 6) bin/goyacc -o idp/userstore/selectorconfigparser/parser.go idp/userstore/selectorconfigparser/parser.y
// 7) revert changes to go.mod and go.sum from steps 2) and 3)
// 8) touch up generated .go files from 4) and 5) to satisfy lint rules
