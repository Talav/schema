package schema

import (
	"maps"
	"reflect"
)

// TagParserFunc is a function type for parsing struct tags into metadata.
type TagParserFunc func(field reflect.StructField, index int, tagValue string) (any, error)

// DefaultMetadataFunc creates default metadata for untagged fields.
type DefaultMetadataFunc func(field reflect.StructField, index int) any

// TagParserRegistry manages registered tag parsers with explicit tag name mapping.
// It is immutable after construction.
type TagParserRegistry struct {
	parsers  map[string]TagParserFunc
	defaults map[string]DefaultMetadataFunc
}

// TagParserRegistryOption configures a TagParserRegistry during construction.
type TagParserRegistryOption func(parsers map[string]TagParserFunc, defaults map[string]DefaultMetadataFunc)

// WithTagParser registers a parser with an explicit tag name.
// If parser is nil, it is skipped. If tag already exists, it is overridden.
// An optional default metadata function can be provided as a third parameter.
func WithTagParser(tagName string, parser TagParserFunc, defaultFunc ...DefaultMetadataFunc) TagParserRegistryOption {
	return func(parsers map[string]TagParserFunc, defaults map[string]DefaultMetadataFunc) {
		if parser == nil || tagName == "" {
			return
		}
		parsers[tagName] = parser

		// If a default function is provided, register it
		if len(defaultFunc) > 0 && defaultFunc[0] != nil {
			defaults[tagName] = defaultFunc[0]
		}
	}
}

// NewTagParserRegistry creates a new immutable tag parser registry with the given options.
func NewTagParserRegistry(opts ...TagParserRegistryOption) *TagParserRegistry {
	parsers := make(map[string]TagParserFunc)
	defaults := make(map[string]DefaultMetadataFunc)

	for _, opt := range opts {
		opt(parsers, defaults)
	}

	return &TagParserRegistry{
		parsers:  parsers,
		defaults: defaults,
	}
}

// NewDefaultTagParserRegistry creates a new tag parser registry with default parsers (schema and body).
func NewDefaultTagParserRegistry() *TagParserRegistry {
	return NewTagParserRegistry(
		WithTagParser("schema", ParseSchemaTag, DefaultSchemaMetadata),
		WithTagParser("body", ParseBodyTag),
	)
}

// Get returns the parser for the given tag name, or nil if not found.
func (r *TagParserRegistry) Get(tagName string) TagParserFunc {
	return r.parsers[tagName]
}

// All returns a copy of all registered parsers.
func (r *TagParserRegistry) All() map[string]TagParserFunc {
	result := make(map[string]TagParserFunc, len(r.parsers))
	maps.Copy(result, r.parsers)

	return result
}

// GetDefault returns the default metadata factory for the given tag name, or nil if not found.
func (r *TagParserRegistry) GetDefault(tagName string) DefaultMetadataFunc {
	return r.defaults[tagName]
}
