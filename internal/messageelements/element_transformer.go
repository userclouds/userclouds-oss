package messageelements

import (
	"bytes"
	texttemplate "text/template"

	"userclouds.com/infra/ucerr"
)

// functions that implement this interface can be used for message element transformation during validation; when registering
// an element type, an appropriate transformer must be specified
type elementTransformer func(elt ElementType, parameters any, element string) (transformedElement string, err error)

var identityTransformer = func(elt ElementType, parameters any, element string) (transformedElement string, err error) {
	return element, nil
}

var templateTransformer = func(elt ElementType, parameters any, element string) (transformedElement string, err error) {
	elementTemplate, err := texttemplate.New("element").Parse(element)
	if err != nil {
		return element, ucerr.Wrap(err)
	}

	buf := &bytes.Buffer{}
	if err := elementTemplate.Execute(buf, parameters); err != nil {
		return element, ucerr.Wrap(err)
	}

	return buf.String(), nil
}
