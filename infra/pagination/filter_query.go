package pagination

const nested = 1

type operator string

const (
	and operator = "AND"
	eq  operator = "EQ"
	ge  operator = "GE"
	gt  operator = "GT"
	le  operator = "LE"
	il  operator = "IL"
	has operator = "HAS"
	lk  operator = "LK"
	lt  operator = "LT"
	ne  operator = "NE"
	nl  operator = "NL"
	or  operator = "OR"
)

func (o operator) isArrayOperator() bool {
	return o == has
}

func (o operator) isComparisonOperator() bool {
	switch o {
	case eq, ge, gt, le, lt, ne:
		return true
	}

	return false
}

func (o operator) isPatternOperator() bool {
	switch o {
	case il, lk, nl:
		return true
	}

	return false
}

func (o operator) isLogicalOperator() bool {
	switch o {
	case and, or:
	default:
		return false
	}

	return true
}

func (o operator) isLeafOperator() bool {
	return o.isComparisonOperator() || o.isPatternOperator() || o.isArrayOperator()
}

func (o operator) queryString() string {
	switch o {
	case eq:
		return "="
	case gt:
		return ">"
	case lt:
		return "<"
	case ge:
		return ">="
	case has:
		return "@>"
	case le:
		return "<="
	case ne:
		return "!="
	case il:
		return "ILIKE"
	case lk:
		return "LIKE"
	case nl:
		return "NOT LIKE"
	case and:
		return "AND"
	case or:
		return "OR"
	}

	return ""
}

// FilterQuery represents a parsed query tree for a filter query
type FilterQuery struct {
	value      any
	parent     *FilterQuery
	leftChild  *FilterQuery
	rightChild *FilterQuery
}

func (fq *FilterQuery) addChild(value any) *FilterQuery {
	if fq.leftChild == nil {
		fq.leftChild = &FilterQuery{
			value:  value,
			parent: fq,
		}
		return fq.leftChild
	}
	fq.rightChild = &FilterQuery{
		value:  value,
		parent: fq,
	}
	return fq.rightChild
}

func (fq *FilterQuery) findAncestor() *FilterQuery {
	for !fq.isRoot() {
		fq = fq.parent
		if fq.isNested() {
			return fq
		}
	}

	return fq
}

func (fq *FilterQuery) isNested() bool {
	return fq.value == nested
}

func (fq *FilterQuery) isRoot() bool {
	return fq.parent == nil
}

func (fq *FilterQuery) rotateLeft(value any) {
	leftChild := &FilterQuery{
		value:      fq.value,
		parent:     fq,
		leftChild:  fq.leftChild,
		rightChild: fq.rightChild,
	}
	fq.value = value
	fq.leftChild = leftChild
	fq.rightChild = nil
}
