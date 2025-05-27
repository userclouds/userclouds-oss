package ucimage

import (
	"fmt"
	"strings"
)

// ImageFormat represents the format of an image
type ImageFormat string

// The supported image formats
const (
	GIF  ImageFormat = "gif"
	JPEG ImageFormat = "jpeg"
	PNG  ImageFormat = "png"
)

// ImageType represents the type of an image
type ImageType string

// The supported image types
const (
	Logo ImageType = "logo"
)

// ImageConstraints represents height, width, size, and format
// restrictions for an image. These restrictions are specified
// by image type
type ImageConstraints struct {
	MaxHeight        int
	MaxSize          int64
	MaxWidth         int
	MinHeight        int
	MinSize          int64
	MinWidth         int
	SupportedFormats map[ImageFormat]struct{}
	// Calculated based on SupportedFormats
	SupportedFormatRegex   string
	SupportedFormatMessage string
}

// NewImageConstraints returns a new instance of ImageConstraints
func NewImageConstraints(minHeight, maxHeight, minWidth, maxWidth int, minSize, maxSize int64, supportedFormats ...ImageFormat) ImageConstraints {
	formatsMap := make(map[ImageFormat]struct{})
	for _, imgFmt := range supportedFormats {
		formatsMap[imgFmt] = struct{}{}
	}

	return ImageConstraints{
		MinHeight:              minHeight,
		MaxHeight:              maxHeight,
		MinWidth:               minWidth,
		MaxWidth:               maxWidth,
		MinSize:                minSize,
		MaxSize:                maxSize,
		SupportedFormats:       formatsMap,
		SupportedFormatMessage: getSupportedFormatString(supportedFormats, ", ", ", and "),
		SupportedFormatRegex:   fmt.Sprintf("(%v)", getSupportedFormatString(supportedFormats, "|", "")),
	}
}

func getSupportedFormatString(supportedFormats []ImageFormat, sep string, lastSeparator string) string {
	sf := make([]string, 0, len(supportedFormats))
	for _, imgFmt := range supportedFormats {
		sf = append(sf, string(imgFmt))
	}
	if lastSeparator != "" {
		dm := strings.Join(sf[:len(sf)-1], sep)
		return strings.Join([]string{dm, sf[len(sf)-1]}, lastSeparator)
	}
	return strings.Join(sf, sep)
}
