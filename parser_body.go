package schema

import (
	"fmt"
	"reflect"

	"github.com/talav/tagparser"
)

// BodyMetadata represents metadata for body tag fields.
type BodyMetadata struct {
	MapKey   string
	BodyType BodyType
	Required bool
}

// BodyType represents the type of request body.
type BodyType string

const (
	BodyTypeStructured BodyType = "structured" // JSON, XML
	BodyTypeFile       BodyType = "file"       // File upload
	BodyTypeMultipart  BodyType = "multipart"  // Multipart form
)

// ParseBodyTag parses a body tag and returns BodyMetadata.
func ParseBodyTag(field reflect.StructField, index int, tagValue string) (any, error) {
	tag, err := tagparser.ParseWithName(tagValue)
	if err != nil {
		return nil, fmt.Errorf("field %s: failed to parse body tag: %w", field.Name, err)
	}

	bodyType, err := parseBodyType(tag.Name)
	if err != nil {
		return nil, fmt.Errorf("field %s: %w", field.Name, err)
	}

	required := extractBoolean(tag.Options, optKeyRequired, false)

	return &BodyMetadata{
		MapKey:   field.Name,
		BodyType: bodyType,
		Required: required,
	}, nil
}

// parseBodyType parses the body type from the tag name.
func parseBodyType(bodyTypeStr string) (BodyType, error) {
	switch bodyTypeStr {
	case "", "structured":
		return BodyTypeStructured, nil
	case "file":
		return BodyTypeFile, nil
	case "multipart":
		return BodyTypeMultipart, nil
	default:
		return "", fmt.Errorf("invalid body type %q (must be 'structured', 'file', or 'multipart')", bodyTypeStr)
	}
}
