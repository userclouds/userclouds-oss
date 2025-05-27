package pagination

import (
	"fmt"
	"regexp"
	"strings"

	"userclouds.com/infra/ucerr"
)

type filterParser struct {
	filter string
	chars  []string
	index  int
	fq     *FilterQuery
}

func newFilterParser(filter string) filterParser {
	return filterParser{
		filter: filter,
		chars:  strings.Split(filter, ""),
		index:  0,
		fq:     &FilterQuery{},
	}
}

func (fp *filterParser) advance(numChars int) {
	fp.index += numChars
}

func (fp filterParser) current() string {
	return fp.chars[fp.index]
}

func (fp filterParser) makeError(s string) error {
	return ucerr.Friendlyf(nil, "filter query '%s' at index '%d' %s", fp.filter, fp.index, s)
}

func (fp filterParser) hasNext() bool {
	return fp.index < len(fp.chars)
}

func (fp *filterParser) parse() (*FilterQuery, error) {
	if len(fp.filter) == 0 {
		return nil, ucerr.New("filter query is empty")
	}

	for fp.hasNext() {
		switch fp.current() {
		case "(":
			fp.parseStartNested()
		case ")":
			if err := fp.parseEndNested(); err != nil {
				return nil, ucerr.Wrap(err)
			}
		case "'":
			if err := fp.parseLeaf(); err != nil {
				return nil, ucerr.Wrap(err)
			}
		case ",":
			if err := fp.parseLogical(); err != nil {
				return nil, ucerr.Wrap(err)
			}
		default:
			return nil, ucerr.Wrap(fp.makeError("has unexpected char"))
		}
	}

	if !fp.fq.parent.isRoot() {
		return nil, ucerr.Errorf("filter query '%s' is unbalanced", fp.filter)
	}

	return fp.fq, nil
}

var leafFilter = regexp.MustCompile(`'[^']+',[A-Z]+,'.*?[^\\]'[)]`)

func (fp *filterParser) parseKey(key string) (string, error) {
	jsonbKeyParts := strings.Split(key, "->>")
	if len(jsonbKeyParts) == 2 {
		return fmt.Sprintf("%s->>'%s'", jsonbKeyParts[0], jsonbKeyParts[1]), nil
	}
	jsonbKeyParts = strings.Split(key, "->")
	if len(jsonbKeyParts) == 2 {
		return fmt.Sprintf("%s->'%s'", jsonbKeyParts[0], jsonbKeyParts[1]), nil
	}
	if len(jsonbKeyParts) == 1 {
		return key, nil
	}
	return "", ucerr.Wrap(fp.makeError("has invalid key"))
}

func (fp *filterParser) parseLeaf() error {
	if !fp.fq.isNested() {
		return ucerr.Wrap(fp.makeError("has unexpected leaf node"))
	}

	remaining := fp.remaining()
	if !leafFilter.MatchString(remaining) {
		return ucerr.Wrap(fp.makeError("has invalid leaf node"))
	}
	leafPrefix := leafFilter.FindString(remaining)

	leafParts := strings.SplitN(strings.TrimPrefix(leafPrefix, "'"), ",", 3)
	if len(leafParts) != 3 {
		return ucerr.Wrap(fp.makeError("has invalid leaf node"))
	}
	key := strings.TrimSuffix(leafParts[0], "'")
	op := operator(leafParts[1])
	value := strings.TrimSuffix(strings.TrimPrefix(leafParts[2], "'"), "')")
	if !op.isLeafOperator() {
		return ucerr.Wrap(fp.makeError("has invalid leaf node"))
	}

	fp.fq = fp.fq.addChild(op)
	keyToAdd, err := fp.parseKey(key)
	if err != nil {
		return ucerr.Wrap(err)
	}
	fp.fq.addChild(keyToAdd)
	fp.fq.addChild(value)
	fp.advance(len(key) + len(op) + len(value) + 6)
	return nil
}

const logicalAndToken = ",AND,"
const logicalOrToken = ",OR,"

func (fp *filterParser) parseLogical() error {
	if fp.fq.isRoot() {
		return ucerr.Wrap(fp.makeError("has invalid logical operator"))
	}

	remaining := fp.remaining()
	if strings.HasPrefix(remaining, logicalAndToken) {
		fp.fq.rotateLeft(and)
		fp.advance(len(logicalAndToken))
	} else if strings.HasPrefix(remaining, logicalOrToken) {
		fp.fq.rotateLeft(or)
		fp.advance(len(logicalOrToken))
	} else {
		return ucerr.Wrap(fp.makeError("has unrecognized operator"))
	}

	return nil
}

func (fp *filterParser) parseEndNested() error {
	fp.fq = fp.fq.findAncestor()
	if fp.fq.isRoot() {
		return ucerr.Wrap(fp.makeError("has unbalanced ')"))
	}
	fp.advance(1)
	return nil
}

func (fp *filterParser) parseStartNested() {
	fp.fq = fp.fq.addChild(nested)
	fp.advance(1)
}

func (fp filterParser) remaining() string {
	return strings.Join(fp.chars[fp.index:], "")
}

// CreateFilterQuery creates a parsed filter query from a filter string
func CreateFilterQuery(s string) (*FilterQuery, error) {
	fp := newFilterParser(s)
	fq, err := fp.parse()
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return fq, nil
}
