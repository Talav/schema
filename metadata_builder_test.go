package schema

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// builderMockParser returns a simple metadata map for testing.
func builderMockParser(field reflect.StructField, index int, tagValue string) (any, error) {
	return map[string]string{"tag": tagValue, "field": field.Name}, nil
}

// builderMockDefault returns a simple default metadata map for testing.
func builderMockDefault(field reflect.StructField, index int) any {
	return map[string]string{"default": "value", "field": field.Name}
}

// builderMockErrorParser always returns an error for testing error handling.
func builderMockErrorParser(field reflect.StructField, index int, tagValue string) (any, error) {
	return nil, assert.AnError
}

func TestMetadataBuilder_BuildStructMetadata_WithTags(t *testing.T) {
	// Test that builder correctly processes fields with tags
	type testStruct struct {
		Name string `custom:"name"`
		Age  int    `custom:"age"`
	}

	registry := NewTagParserRegistry(
		WithTagParser("custom", builderMockParser),
	)

	builder := newMetadataBuilder(registry)
	typ := reflect.TypeOf(testStruct{})

	result, err := builder.buildStructMetadata(typ)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Fields, 2)

	// Verify Name field
	nameField, ok := result.Field("Name")
	require.True(t, ok)
	assert.Equal(t, "Name", nameField.StructFieldName)
	assert.Equal(t, 0, nameField.Index)
	assert.True(t, nameField.HasTag("custom"))
	assert.Len(t, nameField.TagMetadata, 1)

	// Verify Age field
	ageField, ok := result.Field("Age")
	require.True(t, ok)
	assert.Equal(t, "Age", ageField.StructFieldName)
	assert.Equal(t, 1, ageField.Index)
	assert.True(t, ageField.HasTag("custom"))
	assert.Len(t, ageField.TagMetadata, 1)
}

func TestMetadataBuilder_BuildStructMetadata_WithDefaults(t *testing.T) {
	// Test that builder applies defaults when tags are missing
	type testStruct struct {
		Tagged   string `custom:"tagged"`
		Untagged string // No tag, should get default
	}

	registry := NewTagParserRegistry(
		WithTagParser("custom", builderMockParser, builderMockDefault),
	)

	builder := newMetadataBuilder(registry)
	typ := reflect.TypeOf(testStruct{})

	result, err := builder.buildStructMetadata(typ)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Fields, 2)

	// Verify tagged field has parsed metadata
	taggedField, ok := result.Field("Tagged")
	require.True(t, ok)
	assert.True(t, taggedField.HasTag("custom"))
	meta, ok := taggedField.TagMetadata["custom"]
	require.True(t, ok)
	assert.NotNil(t, meta)

	// Verify untagged field has default metadata
	untaggedField, ok := result.Field("Untagged")
	require.True(t, ok)
	assert.True(t, untaggedField.HasTag("custom"))
	defaultMeta, ok := untaggedField.TagMetadata["custom"]
	require.True(t, ok)
	assert.NotNil(t, defaultMeta)
}

func TestMetadataBuilder_BuildStructMetadata_SkipsUnexportedFields(t *testing.T) {
	// Test that builder skips unexported fields
	type testStruct struct {
		Exported   string `custom:"exported"`
		unexported string `custom:"unexported"` //nolint:unused,revive // Test field - verified that it's skipped
	}

	registry := NewTagParserRegistry(
		WithTagParser("custom", builderMockParser),
	)

	builder := newMetadataBuilder(registry)
	typ := reflect.TypeOf(testStruct{})

	result, err := builder.buildStructMetadata(typ)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Fields, 1)

	// Only exported field should be present
	field, ok := result.Field("Exported")
	require.True(t, ok)
	assert.Equal(t, "Exported", field.StructFieldName)

	// Unexported field should not be present
	_, ok = result.Field("unexported")
	assert.False(t, ok)
}

func TestMetadataBuilder_BuildStructMetadata_SkipsFieldsWithNoMetadata(t *testing.T) {
	// Test that builder skips fields with no tags and no defaults
	type testStruct struct {
		Tagged string `custom:"tagged"`
		NoTag  string // No tag, no default
		NoTag2 int    // No tag, no default
	}

	registry := NewTagParserRegistry(
		WithTagParser("custom", builderMockParser), // No default function
	)

	builder := newMetadataBuilder(registry)
	typ := reflect.TypeOf(testStruct{})

	result, err := builder.buildStructMetadata(typ)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Fields, 1)

	// Only tagged field should be present
	field, ok := result.Field("Tagged")
	require.True(t, ok)
	assert.Equal(t, "Tagged", field.StructFieldName)

	// Fields without tags or defaults should be skipped
	_, ok = result.Field("NoTag")
	assert.False(t, ok)
	_, ok = result.Field("NoTag2")
	assert.False(t, ok)
}

func TestMetadataBuilder_BuildStructMetadata_HandlesParsingErrors(t *testing.T) {
	// Test that builder collects and returns parsing errors
	type testStruct struct {
		Valid   string `custom:"valid"`
		Invalid string `custom:"invalid"`
	}

	registry := NewTagParserRegistry(
		WithTagParser("custom", builderMockErrorParser), // Always errors
	)

	builder := newMetadataBuilder(registry)
	typ := reflect.TypeOf(testStruct{})

	result, err := builder.buildStructMetadata(typ)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "parsing errors")
	assert.Contains(t, err.Error(), "Valid")
	assert.Contains(t, err.Error(), "Invalid")
}
