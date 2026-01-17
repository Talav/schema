package schema

import (
	"fmt"
	"reflect"
)

// metadataBuilder orchestrates parsing using registered parsers.
type metadataBuilder struct {
	registry *TagParserRegistry
}

// newMetadataBuilder creates a new metadata builder.
func newMetadataBuilder(registry *TagParserRegistry) *metadataBuilder {
	return &metadataBuilder{
		registry: registry,
	}
}

// BuildStructMetadata parses the struct type and returns its metadata.
func (b *metadataBuilder) buildStructMetadata(typ reflect.Type) (*StructMetadata, error) {
	var fields []FieldMetadata
	var errs []error

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Check if unexported (skip)
		if !field.IsExported() {
			continue
		}

		// Create FieldMetadata with basic info
		fieldMetadata := FieldMetadata{
			StructFieldName: field.Name,
			Index:           i,
			Type:            field.Type,
			Embedded:        field.Anonymous,
			TagMetadata:     make(map[string]any),
		}

		// Iterate through registered parsers (by tag name) to check if field has the tag
		// Go doesn't provide a way to enumerate struct tag keys, so we check registered tags
		allParsers := b.registry.All()

		for tagName, parser := range allParsers {
			// Check if field has the tag
			tagValue, ok := field.Tag.Lookup(tagName)
			if !ok {
				// Field doesn't have this tag, try to apply default for this tag type
				if defaultFunc := b.registry.GetDefault(tagName); defaultFunc != nil {
					fieldMetadata.TagMetadata[tagName] = defaultFunc(field, i)
				}

				continue
			}

			// Field has the tag, parse it normally
			metadata, err := parser(field, i, tagValue)
			if err != nil || metadata == nil {
				errs = append(errs, fmt.Errorf("field %s: failed to parse tag %q: %w", field.Name, tagName, err))

				continue
			}

			// Store in FieldMetadata.TagMetadata[tagName] = metadata
			fieldMetadata.TagMetadata[tagName] = metadata
		}

		// Validate field has at least one TagMetadata entry
		if len(fieldMetadata.TagMetadata) == 0 {
			// Skip fields with no metadata
			continue
		}

		fields = append(fields, fieldMetadata)
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("parsing errors: %w", fmt.Errorf("%v", errs))
	}

	return NewStructMetadata(typ, fields)
}
