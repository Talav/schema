package schema

import (
	"fmt"
	"reflect"
	"slices"

	"github.com/talav/tagparser"
)

// SchemaMetadata represents metadata for schema tag fields.
type SchemaMetadata struct {
	ParamName string
	MapKey    string
	Location  ParameterLocation
	Style     Style
	Explode   bool
	Required  bool
}

const (
	optKeyLocation = "location"
	optKeyStyle    = "style"
	optKeyExplode  = "explode"
	optKeyRequired = "required"
	optValueTrue   = "true"
)

// ParameterLocation represents the location of a parameter in an OpenAPI spec.
type ParameterLocation string

const (
	// LocationQuery represents query parameters.
	LocationQuery ParameterLocation = "query"
	// LocationPath represents path parameters.
	LocationPath ParameterLocation = "path"
	// LocationHeader represents header parameters.
	LocationHeader ParameterLocation = "header"
	// LocationCookie represents cookie parameters.
	LocationCookie ParameterLocation = "cookie"
)

var validLocations = []ParameterLocation{
	LocationQuery,
	LocationPath,
	LocationHeader,
	LocationCookie,
}

// Style represents the serialization style for a parameter.
type Style string

const (
	// StyleMatrix is used for path parameters.
	// Values are prefixed with a semicolon (;) and key-value pairs are separated by an equals sign (=).
	StyleMatrix Style = "matrix"
	// StyleLabel is used for path parameters.
	// Values are prefixed with a period (.).
	StyleLabel Style = "label"
	// StyleForm is commonly used for query and cookie parameters.
	// Values are serialized as form data.
	StyleForm Style = "form"
	// StyleSimple is applicable to path and header parameters.
	// Values are serialized without any additional formatting.
	StyleSimple Style = "simple"
	// StyleSpaceDelimited is used for query parameters.
	// Array values are separated by spaces.
	StyleSpaceDelimited Style = "spaceDelimited"
	// StylePipeDelimited is used for query parameters.
	// Array values are separated by pipes (|).
	StylePipeDelimited Style = "pipeDelimited"
	// StyleDeepObject is used for query parameters.
	// Allows for complex objects to be represented in a deep object style.
	StyleDeepObject Style = "deepObject"
)

// styleGroup represents a key for grouping fields with the same style and explode value.
type styleGroup struct {
	Style   Style
	Explode bool
}

// ParseSchemaTag parses a schema tag and returns SchemaMetadata.
func ParseSchemaTag(field reflect.StructField, index int, tagValue string) (any, error) {
	tag, err := tagparser.ParseWithName(tagValue)
	if err != nil {
		return nil, fmt.Errorf("field %s: failed to parse schema tag: %w", field.Name, err)
	}

	// Use field name if tag name is empty (like JSON standard)
	paramName := tag.Name
	if paramName == "" {
		paramName = field.Name
	}

	location, style := parseLocationAndStyle(tag.Options)
	if err := isValidLocationAndStyle(location, style); err != nil {
		return nil, fmt.Errorf("field %s: %w", field.Name, err)
	}

	explode := extractBoolean(tag.Options, optKeyExplode, defaultExplode(style))
	required := extractBoolean(tag.Options, optKeyRequired, false)
	if location == LocationPath {
		required = true
	}

	return &SchemaMetadata{
		ParamName: paramName,
		MapKey:    field.Name,
		Location:  location,
		Style:     style,
		Explode:   explode,
		Required:  required,
	}, nil
}

// parseLocationAndStyle parses tag options from a map[string]string (from tagparser).
func parseLocationAndStyle(options map[string]string) (location ParameterLocation, style Style) {
	// process location first
	if options[optKeyLocation] != "" {
		location = ParameterLocation(options[optKeyLocation])
	} else {
		location = LocationQuery
	}

	// process style next, it depends on location
	if options[optKeyStyle] != "" {
		style = Style(options[optKeyStyle])
	} else {
		style = defaultStyle(location)
	}

	return location, style
}

// isValidLocationAndStyle validates style against location with field context.
func isValidLocationAndStyle(location ParameterLocation, style Style) error {
	// Validate location first
	if !isLocationValid(location) {
		return fmt.Errorf("invalid location %q", location)
	}

	// Then validate style for that location
	if !isStyleAllowed(location, style) {
		return fmt.Errorf("invalid style %q for location %q", style, location)
	}

	return nil
}

// allowedStyles returns all allowed style values for the given parameter location.
func allowedStyles(location ParameterLocation) []Style {
	switch location {
	case LocationQuery:
		return []Style{
			StyleForm,
			StyleSpaceDelimited,
			StylePipeDelimited,
			StyleDeepObject,
		}
	case LocationPath:
		return []Style{
			StyleSimple,
			StyleLabel,
			StyleMatrix,
		}
	case LocationHeader:
		return []Style{
			StyleSimple,
		}
	case LocationCookie:
		return []Style{
			StyleForm,
		}
	default:
		return nil
	}
}

// isStyleAllowed checks if the given style is allowed for the specified parameter location.
func isStyleAllowed(location ParameterLocation, style Style) bool {
	return slices.Contains(allowedStyles(location), style)
}

func isLocationValid(location ParameterLocation) bool {
	return slices.Contains(validLocations, location)
}

func defaultExplode(style Style) bool {
	return style == StyleForm || style == StyleDeepObject
}

// defaultStyle returns the default style for the given parameter location.
// Default values (based on value of in):
//   - query - form
//   - path - simple
//   - header - simple
//   - cookie - form
func defaultStyle(location ParameterLocation) Style {
	switch location {
	case LocationQuery:
		return StyleForm
	case LocationPath:
		return StyleSimple
	case LocationHeader:
		return StyleSimple
	case LocationCookie:
		return StyleForm
	default:
		return StyleForm
	}
}

// NewDefaultSchemaMetadata creates default SchemaMetadata for untagged fields.
// This function is used by the metadata builder when no explicit schema tag is found.
func DefaultSchemaMetadata(field reflect.StructField, index int) any {
	location := LocationQuery
	style := defaultStyle(location)
	explode := defaultExplode(style)

	return &SchemaMetadata{
		ParamName: field.Name,
		MapKey:    field.Name,
		Location:  location,
		Style:     style,
		Explode:   explode,
		Required:  false,
	}
}
