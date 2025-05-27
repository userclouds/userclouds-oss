package selectorconfigparser

import (
	"bufio"
	"io"
	"strings"
)

type frame struct {
	i            int
	s            string
	line, column int
}

// Lexer is a generated lexer.
type Lexer struct {
	// The lexer runs in its own goroutine, and communicates via channel 'ch'.
	ch     chan frame
	chStop chan bool
	// We record the level of nesting because the action could return, and a
	// subsequent call expects to pick up where it left off. In other words,
	// we're simulating a coroutine.
	// TODO: Support a channel-based variant that compatible with Go's yacc.
	stack []frame
	stale bool

	// The following line makes it easy for scripts to insert fields in the
	// generated code.
	// [NEX_END_OF_LEXER_STRUCT]
}

// NewLexerWithInit creates a new Lexer object, runs the given callback on it,
// then returns it.
func NewLexerWithInit(in io.Reader, initFun func(*Lexer)) *Lexer {
	yylex := new(Lexer)
	if initFun != nil {
		initFun(yylex)
	}
	yylex.ch = make(chan frame)
	yylex.chStop = make(chan bool, 1)
	var scan func(in *bufio.Reader, ch chan frame, chStop chan bool, family []dfa, line, column int)
	scan = func(in *bufio.Reader, ch chan frame, chStop chan bool, family []dfa, line, column int) {
		// Index of DFA and length of highest-precedence match so far.
		matchi, matchn := 0, -1
		var buf []rune
		n := 0
		checkAccept := func(i int, st int) bool {
			// Higher precedence match? DFAs are run in parallel, so matchn is at most len(buf), hence we may omit the length equality check.
			if family[i].acc[st] && (matchn < n || matchi > i) {
				matchi, matchn = i, n
				return true
			}
			return false
		}
		var state [][2]int
		for i := range family {
			mark := make([]bool, len(family[i].startf))
			// Every DFA starts at state 0.
			st := 0
			for {
				state = append(state, [2]int{i, st})
				mark[st] = true
				// As we're at the start of input, follow all ^ transitions and append to our list of start states.
				st = family[i].startf[st]
				if st == -1 || mark[st] {
					break
				}
				// We only check for a match after at least one transition.
				checkAccept(i, st)
			}
		}
		atEOF := false
		stopped := false
		for {
			if n == len(buf) && !atEOF {
				r, _, err := in.ReadRune()
				switch err {
				case io.EOF:
					atEOF = true
				case nil:
					buf = append(buf, r)
				default:
					panic(err)
				}
			}
			if !atEOF {
				r := buf[n]
				n++
				var nextState [][2]int
				for _, x := range state {
					x[1] = family[x[0]].f[x[1]](r)
					if x[1] == -1 {
						continue
					}
					nextState = append(nextState, x)
					checkAccept(x[0], x[1])
				}
				state = nextState
			} else {
			dollar: // Handle $.
				for _, x := range state {
					mark := make([]bool, len(family[x[0]].endf))
					for {
						mark[x[1]] = true
						x[1] = family[x[0]].endf[x[1]]
						if x[1] == -1 || mark[x[1]] {
							break
						}
						if checkAccept(x[0], x[1]) {
							// Unlike before, we can break off the search. Now that we're at the end, there's no need to maintain the state of each DFA.
							break dollar
						}
					}
				}
				state = nil
			}

			if state == nil {
				lcUpdate := func(r rune) {
					if r == '\n' {
						line++
						column = 0
					} else {
						column++
					}
				}
				// All DFAs stuck. Return last match if it exists, otherwise advance by one rune and restart all DFAs.
				if matchn == -1 {
					if len(buf) == 0 { // This can only happen at the end of input.
						break
					}
					lcUpdate(buf[0])
					buf = buf[1:]
				} else {
					text := string(buf[:matchn])
					buf = buf[matchn:]
					matchn = -1
					select {
					case ch <- frame{matchi, text, line, column}:
						{
						}
					case stopped = <-chStop:
						{
						}
					}
					if stopped {
						break
					}
					if len(family[matchi].nest) > 0 {
						scan(bufio.NewReader(strings.NewReader(text)), ch, chStop, family[matchi].nest, line, column)
					}
					if atEOF {
						break
					}
					for _, r := range text {
						lcUpdate(r)
					}
				}
				n = 0
				for i := range family {
					state = append(state, [2]int{i, 0})
				}
			}
		}
		ch <- frame{-1, "", line, column}
	}
	go scan(bufio.NewReader(in), yylex.ch, yylex.chStop, dfas, 0, 0)
	return yylex
}

type dfa struct {
	acc          []bool           // Accepting states.
	f            []func(rune) int // Transitions.
	startf, endf []int            // Transitions at start and end of input.
	nest         []dfa
}

var dfas = []dfa{
	// {[a-zA-Z0-9_-]+}(->>'[a-zA-Z0-9_-]+')?
	{[]bool{false, false, false, true, false, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 45:
				return -1
			case 62:
				return -1
			case 95:
				return -1
			case 123:
				return 1
			case 125:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 90:
				return -1
			case 97 <= r && r <= 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 45:
				return 2
			case 62:
				return -1
			case 95:
				return 2
			case 123:
				return -1
			case 125:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 2
			case 65 <= r && r <= 90:
				return 2
			case 97 <= r && r <= 122:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 45:
				return 2
			case 62:
				return -1
			case 95:
				return 2
			case 123:
				return -1
			case 125:
				return 3
			}
			switch {
			case 48 <= r && r <= 57:
				return 2
			case 65 <= r && r <= 90:
				return 2
			case 97 <= r && r <= 122:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 45:
				return 4
			case 62:
				return -1
			case 95:
				return -1
			case 123:
				return -1
			case 125:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 90:
				return -1
			case 97 <= r && r <= 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 45:
				return -1
			case 62:
				return 5
			case 95:
				return -1
			case 123:
				return -1
			case 125:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 90:
				return -1
			case 97 <= r && r <= 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 45:
				return -1
			case 62:
				return 6
			case 95:
				return -1
			case 123:
				return -1
			case 125:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 90:
				return -1
			case 97 <= r && r <= 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return 7
			case 45:
				return -1
			case 62:
				return -1
			case 95:
				return -1
			case 123:
				return -1
			case 125:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 90:
				return -1
			case 97 <= r && r <= 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 45:
				return 8
			case 62:
				return -1
			case 95:
				return 8
			case 123:
				return -1
			case 125:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 8
			case 65 <= r && r <= 90:
				return 8
			case 97 <= r && r <= 122:
				return 8
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return 9
			case 45:
				return 8
			case 62:
				return -1
			case 95:
				return 8
			case 123:
				return -1
			case 125:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 8
			case 65 <= r && r <= 90:
				return 8
			case 97 <= r && r <= 122:
				return 8
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 45:
				return -1
			case 62:
				return -1
			case 95:
				return -1
			case 123:
				return -1
			case 125:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			case 65 <= r && r <= 90:
				return -1
			case 97 <= r && r <= 122:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// ABS|abs|CHAR_LENGTH|char_length|CHARACTER_LENGTH|character_length|LOWER|lower|UPPER|upper
	{[]bool{false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, true, false, true, false, false, false, true, false, false, false, true, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, true, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return 1
			case 66:
				return -1
			case 67:
				return 2
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return 3
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return 4
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return 5
			case 98:
				return -1
			case 99:
				return 6
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return 7
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return 8
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return 71
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return 49
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return 45
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return 41
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return 39
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return 17
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return 13
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return 9
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return 10
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return 11
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return 12
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return 14
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return 15
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return 16
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return 18
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return 19
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return 20
			case 97:
				return 21
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return 33
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return 22
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return 23
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return 24
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return 25
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return 26
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return 27
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return 28
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return 29
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return 30
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return 31
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return 32
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return 34
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return 35
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return 36
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return 37
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return 38
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return 40
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return 42
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return 43
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return 44
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return 46
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return 47
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return 48
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 50
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return 51
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 52
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return 53
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return 60
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return 54
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return 55
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return 56
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return 57
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return 58
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return 59
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return 61
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return 62
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return 63
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return 64
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return 65
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return 66
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return 67
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return 68
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return 69
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return 70
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return 72
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 66:
				return -1
			case 67:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 72:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 87:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 98:
				return -1
			case 99:
				return -1
			case 101:
				return -1
			case 103:
				return -1
			case 104:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// ,
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 44:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 44:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// '(day|dow|doy|epoch|hour|microseconds|milliseconds|minute|month|second|timezone|week|year)'
	{[]bool{false, false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 39:
				return 1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return 2
			case 101:
				return 3
			case 104:
				return 4
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return 5
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return 6
			case 116:
				return 7
			case 117:
				return -1
			case 119:
				return 8
			case 121:
				return 9
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return 65
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return 66
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return 61
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return 58
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return 29
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return 30
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 24
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return 17
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 14
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 10
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return 11
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return 12
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return 13
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 15
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return 16
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return 13
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return 18
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 19
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return 20
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return 21
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return 22
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 23
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return 13
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return 25
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return 26
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return 27
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return 28
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return 13
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return 34
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return 35
			case 109:
				return -1
			case 110:
				return 36
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return 31
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return 32
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return 33
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return 13
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return 49
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return 40
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return 37
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return 38
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 39
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return 13
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return 41
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return 42
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 43
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return 44
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return 45
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return 46
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return 47
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return 48
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return 13
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return 50
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return 51
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 52
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return 53
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return 54
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return 55
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return 56
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return 57
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return 13
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return 59
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return 60
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return 13
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return 62
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return 63
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return 64
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return 13
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return 69
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return 67
			case 121:
				return 68
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return 13
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return 13
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return 13
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 104:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			case 109:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			case 119:
				return -1
			case 121:
				return -1
			case 122:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// DATE_PART|date_part|DATE_TRUNC|date_trunc
	{[]bool{false, false, false, false, false, false, false, false, false, false, false, false, true, false, false, true, false, false, false, false, false, false, false, false, false, true, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return 1
			case 69:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return 2
			case 101:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 16
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 95:
				return -1
			case 97:
				return 3
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return 4
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return 5
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 95:
				return 6
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 112:
				return 7
			case 114:
				return -1
			case 116:
				return 8
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 95:
				return -1
			case 97:
				return 13
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return 9
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return 10
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return 11
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 99:
				return 12
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return 14
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return 15
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return 17
			case 85:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return 18
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 95:
				return 19
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 80:
				return 20
			case 82:
				return -1
			case 84:
				return 21
			case 85:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 26
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return 22
			case 84:
				return -1
			case 85:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return 23
			case 95:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return 24
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return 25
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return 27
			case 84:
				return -1
			case 85:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return 28
			case 85:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 67:
				return -1
			case 68:
				return -1
			case 69:
				return -1
			case 78:
				return -1
			case 80:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 95:
				return -1
			case 97:
				return -1
			case 99:
				return -1
			case 100:
				return -1
			case 101:
				return -1
			case 110:
				return -1
			case 112:
				return -1
			case 114:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// DIV|div|MOD|mod
	{[]bool{false, false, false, false, false, false, true, false, true, false, true, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 68:
				return 1
			case 73:
				return -1
			case 77:
				return 2
			case 79:
				return -1
			case 86:
				return -1
			case 100:
				return 3
			case 105:
				return -1
			case 109:
				return 4
			case 111:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 73:
				return 11
			case 77:
				return -1
			case 79:
				return -1
			case 86:
				return -1
			case 100:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 111:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 79:
				return 9
			case 86:
				return -1
			case 100:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 111:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 79:
				return -1
			case 86:
				return -1
			case 100:
				return -1
			case 105:
				return 7
			case 109:
				return -1
			case 111:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 79:
				return -1
			case 86:
				return -1
			case 100:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 111:
				return 5
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 79:
				return -1
			case 86:
				return -1
			case 100:
				return 6
			case 105:
				return -1
			case 109:
				return -1
			case 111:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 79:
				return -1
			case 86:
				return -1
			case 100:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 111:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 79:
				return -1
			case 86:
				return -1
			case 100:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 111:
				return -1
			case 118:
				return 8
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 79:
				return -1
			case 86:
				return -1
			case 100:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 111:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return 10
			case 73:
				return -1
			case 77:
				return -1
			case 79:
				return -1
			case 86:
				return -1
			case 100:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 111:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 79:
				return -1
			case 86:
				return -1
			case 100:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 111:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 79:
				return -1
			case 86:
				return 12
			case 100:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 111:
				return -1
			case 118:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 68:
				return -1
			case 73:
				return -1
			case 77:
				return -1
			case 79:
				return -1
			case 86:
				return -1
			case 100:
				return -1
			case 105:
				return -1
			case 109:
				return -1
			case 111:
				return -1
			case 118:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// =|<=|>=|<|>|!=| LIKE | like | ILIKE | ilike
	{[]bool{false, false, false, true, true, true, true, true, true, false, false, false, false, false, false, false, true, false, false, false, false, true, false, false, false, true, false, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 32:
				return 1
			case 33:
				return 2
			case 60:
				return 3
			case 61:
				return 4
			case 62:
				return 5
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 33:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 69:
				return -1
			case 73:
				return 9
			case 75:
				return -1
			case 76:
				return 10
			case 101:
				return -1
			case 105:
				return 11
			case 107:
				return -1
			case 108:
				return 12
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 33:
				return -1
			case 60:
				return -1
			case 61:
				return 8
			case 62:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 33:
				return -1
			case 60:
				return -1
			case 61:
				return 7
			case 62:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 33:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 33:
				return -1
			case 60:
				return -1
			case 61:
				return 6
			case 62:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 33:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 33:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 33:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 33:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 76:
				return 26
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 33:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 69:
				return -1
			case 73:
				return 22
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 33:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return 17
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 33:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return 13
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 33:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return 14
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 33:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return 15
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return 16
			case 33:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 33:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 33:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return 18
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 33:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return 19
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 33:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return 20
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return 21
			case 33:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 33:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 33:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return 23
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 33:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 69:
				return 24
			case 73:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return 25
			case 33:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 33:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 33:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 69:
				return -1
			case 73:
				return 27
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 33:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return 28
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 33:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 69:
				return 29
			case 73:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return 30
			case 33:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 33:
				return -1
			case 60:
				return -1
			case 61:
				return -1
			case 62:
				return -1
			case 69:
				return -1
			case 73:
				return -1
			case 75:
				return -1
			case 76:
				return -1
			case 101:
				return -1
			case 105:
				return -1
			case 107:
				return -1
			case 108:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// IS|is
	{[]bool{false, false, false, true, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 73:
				return 1
			case 83:
				return -1
			case 105:
				return 2
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 83:
				return 4
			case 105:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 83:
				return -1
			case 105:
				return -1
			case 115:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 83:
				return -1
			case 105:
				return -1
			case 115:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 73:
				return -1
			case 83:
				return -1
			case 105:
				return -1
			case 115:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1}, nil},

	// NOT|not|!
	{[]bool{false, true, false, false, false, true, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 33:
				return 1
			case 78:
				return 2
			case 79:
				return -1
			case 84:
				return -1
			case 110:
				return 3
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 33:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 33:
				return -1
			case 78:
				return -1
			case 79:
				return 6
			case 84:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 33:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 110:
				return -1
			case 111:
				return 4
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 33:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return 5
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 33:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 33:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return 7
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 33:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 84:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 116:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// NULL|null
	{[]bool{false, false, false, false, false, true, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 76:
				return -1
			case 78:
				return 1
			case 85:
				return -1
			case 108:
				return -1
			case 110:
				return 2
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return -1
			case 78:
				return -1
			case 85:
				return 6
			case 108:
				return -1
			case 110:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return -1
			case 78:
				return -1
			case 85:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 117:
				return 3
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return -1
			case 78:
				return -1
			case 85:
				return -1
			case 108:
				return 4
			case 110:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return -1
			case 78:
				return -1
			case 85:
				return -1
			case 108:
				return 5
			case 110:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return -1
			case 78:
				return -1
			case 85:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return 7
			case 78:
				return -1
			case 85:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return 8
			case 78:
				return -1
			case 85:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 76:
				return -1
			case 78:
				return -1
			case 85:
				return -1
			case 108:
				return -1
			case 110:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// \?
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 63:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 63:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// (FALSE|false|TRUE|true|1::BOOL(EAN)?|0::BOOL(EAN)?)(::BOOL(EAN)?)?
	{[]bool{false, false, false, false, false, false, false, false, false, true, false, false, false, false, false, true, false, false, true, false, false, false, true, false, false, true, false, false, false, true, false, false, false, false, false, true, false, false, true, false, false, false, false, false, true, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 48:
				return 1
			case 49:
				return 2
			case 58:
				return -1
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return 3
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return 4
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return 5
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return 6
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return 39
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return 30
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return 26
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return 23
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return 19
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return 7
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return 8
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return 9
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return 10
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return 11
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return -1
			case 66:
				return 12
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return 13
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return 14
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return 15
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return 16
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return 17
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return 18
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return 20
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return 21
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return 22
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return 10
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return 24
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return 25
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return 10
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return 27
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return 28
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return 29
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return 10
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return 31
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return -1
			case 66:
				return 32
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return 33
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return 34
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return 35
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return 10
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return 36
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return 37
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return 38
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return 10
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return 40
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return -1
			case 66:
				return 41
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return 42
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return 43
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return 44
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return 10
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return 45
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return 46
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return -1
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return 47
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 48:
				return -1
			case 49:
				return -1
			case 58:
				return 10
			case 65:
				return -1
			case 66:
				return -1
			case 69:
				return -1
			case 70:
				return -1
			case 76:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 83:
				return -1
			case 84:
				return -1
			case 85:
				return -1
			case 97:
				return -1
			case 101:
				return -1
			case 102:
				return -1
			case 108:
				return -1
			case 114:
				return -1
			case 115:
				return -1
			case 116:
				return -1
			case 117:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// [-+]?[0-9]+(::INT(EGER)?)?
	{[]bool{false, false, true, false, false, false, false, true, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 43:
				return 1
			case 45:
				return 1
			case 58:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 58:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 58:
				return 3
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return 2
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 58:
				return 4
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 58:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return 5
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 58:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return 6
			case 82:
				return -1
			case 84:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 58:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return 7
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 58:
				return -1
			case 69:
				return 8
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 58:
				return -1
			case 69:
				return -1
			case 71:
				return 9
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 58:
				return -1
			case 69:
				return 10
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 58:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return 11
			case 84:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 43:
				return -1
			case 45:
				return -1
			case 58:
				return -1
			case 69:
				return -1
			case 71:
				return -1
			case 73:
				return -1
			case 78:
				return -1
			case 82:
				return -1
			case 84:
				return -1
			}
			switch {
			case 48 <= r && r <= 57:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// ARRAY|array
	{[]bool{false, false, false, false, false, false, true, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return 1
			case 82:
				return -1
			case 89:
				return -1
			case 97:
				return 2
			case 114:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 82:
				return 7
			case 89:
				return -1
			case 97:
				return -1
			case 114:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 82:
				return -1
			case 89:
				return -1
			case 97:
				return -1
			case 114:
				return 3
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 82:
				return -1
			case 89:
				return -1
			case 97:
				return -1
			case 114:
				return 4
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 82:
				return -1
			case 89:
				return -1
			case 97:
				return 5
			case 114:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 82:
				return -1
			case 89:
				return -1
			case 97:
				return -1
			case 114:
				return -1
			case 121:
				return 6
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 82:
				return -1
			case 89:
				return -1
			case 97:
				return -1
			case 114:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 82:
				return 8
			case 89:
				return -1
			case 97:
				return -1
			case 114:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return 9
			case 82:
				return -1
			case 89:
				return -1
			case 97:
				return -1
			case 114:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 82:
				return -1
			case 89:
				return 10
			case 97:
				return -1
			case 114:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 82:
				return -1
			case 89:
				return -1
			case 97:
				return -1
			case 114:
				return -1
			case 121:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// \[
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 91:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 91:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \]
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 93:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 93:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// '([^']|”)+'(::[A-Z]+)?
	{[]bool{false, false, false, false, true, false, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 39:
				return 1
			case 58:
				return -1
			}
			switch {
			case 65 <= r && r <= 90:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return 2
			case 58:
				return 3
			}
			switch {
			case 65 <= r && r <= 90:
				return 3
			}
			return 3
		},
		func(r rune) int {
			switch r {
			case 39:
				return 5
			case 58:
				return -1
			}
			switch {
			case 65 <= r && r <= 90:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return 4
			case 58:
				return 3
			}
			switch {
			case 65 <= r && r <= 90:
				return 3
			}
			return 3
		},
		func(r rune) int {
			switch r {
			case 39:
				return 5
			case 58:
				return 6
			}
			switch {
			case 65 <= r && r <= 90:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return 4
			case 58:
				return 3
			}
			switch {
			case 65 <= r && r <= 90:
				return 3
			}
			return 3
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 58:
				return 7
			}
			switch {
			case 65 <= r && r <= 90:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 58:
				return -1
			}
			switch {
			case 65 <= r && r <= 90:
				return 8
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 39:
				return -1
			case 58:
				return -1
			}
			switch {
			case 65 <= r && r <= 90:
				return 8
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// ANY|any
	{[]bool{false, false, false, false, true, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 65:
				return 1
			case 78:
				return -1
			case 89:
				return -1
			case 97:
				return 2
			case 110:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 78:
				return 5
			case 89:
				return -1
			case 97:
				return -1
			case 110:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 78:
				return -1
			case 89:
				return -1
			case 97:
				return -1
			case 110:
				return 3
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 78:
				return -1
			case 89:
				return -1
			case 97:
				return -1
			case 110:
				return -1
			case 121:
				return 4
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 78:
				return -1
			case 89:
				return -1
			case 97:
				return -1
			case 110:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 78:
				return -1
			case 89:
				return 6
			case 97:
				return -1
			case 110:
				return -1
			case 121:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 65:
				return -1
			case 78:
				return -1
			case 89:
				return -1
			case 97:
				return -1
			case 110:
				return -1
			case 121:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1}, nil},

	// AND | and | OR | or
	{[]bool{false, false, false, false, false, false, false, true, false, false, true, false, true, false, false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 32:
				return 1
			case 65:
				return -1
			case 68:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 65:
				return 2
			case 68:
				return -1
			case 78:
				return -1
			case 79:
				return 3
			case 82:
				return -1
			case 97:
				return 4
			case 100:
				return -1
			case 110:
				return -1
			case 111:
				return 5
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 65:
				return -1
			case 68:
				return -1
			case 78:
				return 13
			case 79:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 65:
				return -1
			case 68:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return 11
			case 97:
				return -1
			case 100:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 65:
				return -1
			case 68:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 110:
				return 8
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 65:
				return -1
			case 68:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 114:
				return 6
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return 7
			case 65:
				return -1
			case 68:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 65:
				return -1
			case 68:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 65:
				return -1
			case 68:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 100:
				return 9
			case 110:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return 10
			case 65:
				return -1
			case 68:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 65:
				return -1
			case 68:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return 12
			case 65:
				return -1
			case 68:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 65:
				return -1
			case 68:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 65:
				return -1
			case 68:
				return 14
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return 15
			case 65:
				return -1
			case 68:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 32:
				return -1
			case 65:
				return -1
			case 68:
				return -1
			case 78:
				return -1
			case 79:
				return -1
			case 82:
				return -1
			case 97:
				return -1
			case 100:
				return -1
			case 110:
				return -1
			case 111:
				return -1
			case 114:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, []int{ /* End-of-input transitions */ -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, nil},

	// \(
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 40:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 40:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// \)
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 41:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 41:
				return -1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// [ \t\n\f\r]+
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			switch r {
			case 9:
				return 1
			case 10:
				return 1
			case 12:
				return 1
			case 13:
				return 1
			case 32:
				return 1
			}
			return -1
		},
		func(r rune) int {
			switch r {
			case 9:
				return 1
			case 10:
				return 1
			case 12:
				return 1
			case 13:
				return 1
			case 32:
				return 1
			}
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},

	// .
	{[]bool{false, true}, []func(rune) int{ // Transitions
		func(r rune) int {
			return 1
		},
		func(r rune) int {
			return -1
		},
	}, []int{ /* Start-of-input transitions */ -1, -1}, []int{ /* End-of-input transitions */ -1, -1}, nil},
}

// NewLexer creates a new default Lexer.
func NewLexer(in io.Reader) *Lexer {
	return NewLexerWithInit(in, nil)
}

// Stop stops the lexer.
func (yylex *Lexer) Stop() {
	yylex.chStop <- true
}

// Text returns the matched text.
func (yylex *Lexer) Text() string {
	return yylex.stack[len(yylex.stack)-1].s
}

// Line returns the current line number.
// The first line is 0.
func (yylex *Lexer) Line() int {
	if len(yylex.stack) == 0 {
		return 0
	}
	return yylex.stack[len(yylex.stack)-1].line
}

// Column returns the current column number.
// The first column is 0.
func (yylex *Lexer) Column() int {
	if len(yylex.stack) == 0 {
		return 0
	}
	return yylex.stack[len(yylex.stack)-1].column
}

func (yylex *Lexer) next(lvl int) int {
	if lvl == len(yylex.stack) {
		l, c := 0, 0
		if lvl > 0 {
			l, c = yylex.stack[lvl-1].line, yylex.stack[lvl-1].column
		}
		yylex.stack = append(yylex.stack, frame{0, "", l, c})
	}
	if lvl == len(yylex.stack)-1 {
		p := &yylex.stack[lvl]
		*p = <-yylex.ch
		yylex.stale = false
	} else {
		yylex.stale = true
	}
	return yylex.stack[lvl].i
}
func (yylex *Lexer) pop() {
	yylex.stack = yylex.stack[:len(yylex.stack)-1]
}
func (yylex Lexer) Error(e string) {
	panic(e)
}

// Lex runs the lexer. Always returns 0.
// When the -s option is given, this function is not generated;
// instead, the NN_FUN macro runs the lexer.
func (yylex *Lexer) Lex(lval *yySymType) int {
OUTER0:
	for {
		switch yylex.next(0) {
		case 0:
			{
				return COLUMN_IDENTIFIER
			}
		case 1:
			{
				return COLUMN_OPERATOR
			}
		case 2:
			{
				return COMMA
			}
		case 3:
			{
				return DATE_ARGUMENT
			}
		case 4:
			{
				return DATE_OPERATOR
			}
		case 5:
			{
				return NUMBER_PART_OPERATOR
			}
		case 6:
			{
				return OPERATOR
			}
		case 7:
			{
				return IS
			}
		case 8:
			{
				return NOT
			}
		case 9:
			{
				return NULL
			}
		case 10:
			{
				return VALUE_PLACEHOLDER
			}
		case 11:
			{
				return BOOL_VALUE
			}
		case 12:
			{
				return INT_VALUE
			}
		case 13:
			{
				return ARRAY_OPERATOR
			}
		case 14:
			{
				return LEFT_BRACKET
			}
		case 15:
			{
				return RIGHT_BRACKET
			}
		case 16:
			{
				return QUOTED_VALUE
			}
		case 17:
			{
				return ANY
			}
		case 18:
			{
				return CONJUNCTION
			}
		case 19:
			{
				return LEFT_PARENTHESIS
			}
		case 20:
			{
				return RIGHT_PARENTHESIS
			}
		case 21:
			{ /* eat up whitespace */
			}
		case 22:
			{
				return UNKNOWN
			}
		default:
			break OUTER0
		}
		continue
	}
	yylex.pop()

	return 0
}
