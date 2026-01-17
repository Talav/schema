package schema

import (
	"errors"
	"fmt"
	"reflect"
)

// Metadata is a separate component for metadata operations (tag parsing, metadata building, caching).
// It serves use cases beyond Codec:
// - Validation metadata (for validators)
// - OpenAPI schema generation
// - Introspection and tooling.
type Metadata struct {
	cache *metadataCache
}

// NewMetadata creates a new Metadata with the given registry.
func NewMetadata(registry *TagParserRegistry) *Metadata {
	// Create builder with registry
	builder := newMetadataBuilder(registry)

	// Create internal cache with builder
	cache := newMetadataCache(builder)

	return &Metadata{
		cache: cache,
	}
}

// NewDefaultMetadata creates a new Metadata with default parsers (schema and body).
func NewDefaultMetadata() *Metadata {
	registry := NewDefaultTagParserRegistry()

	return NewMetadata(registry)
}

// GetStructMetadata retrieves or builds struct metadata for the given type.
func (m *Metadata) GetStructMetadata(typ reflect.Type) (*StructMetadata, error) {
	return m.cache.get(typ)
}

// GetTagMetadata is a package-level generic function for type-safe access to tag metadata.
// Usage: GetTagMetadata[*SchemaMetadata](field, "schema").
func GetTagMetadata[T any](f *FieldMetadata, tagName string) (T, bool) {
	var zero T
	if f == nil || f.TagMetadata == nil {
		return zero, false
	}
	if meta, ok := f.TagMetadata[tagName]; ok {
		if typed, ok := meta.(T); ok {
			return typed, true
		}
	}

	return zero, false
}

// HasTag checks if field has a specific tag.
func (f *FieldMetadata) HasTag(tagName string) bool {
	if f.TagMetadata == nil {
		return false
	}
	_, ok := f.TagMetadata[tagName]

	return ok
}

type StructMetadata struct {
	Type         reflect.Type
	Fields       []FieldMetadata
	fieldsByName map[string]*FieldMetadata
}

// NewStructMetadata creates a new struct metadata from a type and fields.
// This is useful for tests and when you already have FieldMetadata built.
func NewStructMetadata(typ reflect.Type, fields []FieldMetadata) (*StructMetadata, error) {
	// Validate all fields and collect errors
	var errs []error
	for i, field := range fields {
		if err := validateField(field); err != nil {
			errs = append(errs, fmt.Errorf("field[%d] %q: %w", i, field.StructFieldName, err))
		}
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("validation failed: %w", errors.Join(errs...))
	}

	// Build map for O(1) lookup by StructFieldName
	fieldsByName := make(map[string]*FieldMetadata, len(fields))
	for i := range fields {
		fieldsByName[fields[i].StructFieldName] = &fields[i]
	}

	return &StructMetadata{
		Type:         typ,
		Fields:       fields,
		fieldsByName: fieldsByName,
	}, nil
}

// Field returns FieldMetadata by field name.
func (m *StructMetadata) Field(fieldName string) (*FieldMetadata, bool) {
	field, exists := m.fieldsByName[fieldName]
	if !exists {
		return nil, false
	}

	return field, true
}

// FieldMetadata represents a cached struct field metadata.
// It can represent both parameter fields (schema tag) and body fields (body tag).
type FieldMetadata struct {
	// StructFieldName is the name of the struct field in Go source code.
	StructFieldName string
	// Index is the field index in the struct (used for reflection-based field access).
	Index int
	// Embedded indicates whether this field is an embedded/anonymous struct field.
	Embedded bool
	// Type is the reflect.Type of the field.
	Type reflect.Type

	// Tag-specific metadata: tag name -> metadata object
	// A field can have multiple tags (e.g., schema + validate)
	TagMetadata map[string]any // "schema" -> *SchemaMetadata, "body" -> *BodyMetadata, etc.
}

// validateField validates a single FieldMetadata and returns an error if invalid.
func validateField(field FieldMetadata) error {
	var errs []error

	if field.StructFieldName == "" {
		errs = append(errs, fmt.Errorf("structFieldName cannot be empty"))
	}

	if field.Type == nil {
		errs = append(errs, fmt.Errorf("type cannot be nil"))
	}

	if field.Index < 0 {
		errs = append(errs, fmt.Errorf("index must be non-negative, got %d", field.Index))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
