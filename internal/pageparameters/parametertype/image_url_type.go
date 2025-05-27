package parametertype

import (
	"userclouds.com/internal/ucimage"
)

// ImageURL is a parameter type representing a URL for retrieving a stored image
const ImageURL Type = "image_url"

func init() {
	imageURLPattern := ucimage.MustGenerateURLRegexp(ucimage.Logo)
	validator := func(v string) bool {
		return v == "" || imageURLPattern.MatchString(v)
	}

	if err := registerParameterType(ImageURL, validator); err != nil {
		panic(err)
	}
}
