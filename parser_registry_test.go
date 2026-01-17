package schema

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockParser is a test parser function.
func mockParser(field reflect.StructField, index int, tagValue string) (any, error) {
	return map[string]string{"tag": tagValue}, nil
}

// mockDefault is a test default metadata function.
func mockDefault(field reflect.StructField, index int) any {
	return map[string]string{"default": "value"}
}

func TestNewTagParserRegistry_Empty(t *testing.T) {
	registry := NewTagParserRegistry()

	assert.NotNil(t, registry)
	assert.Nil(t, registry.Get("schema"))
	assert.Nil(t, registry.Get("body"))
	assert.Empty(t, registry.All())
}

func TestNewTagParserRegistry_SingleParser(t *testing.T) {
	registry := NewTagParserRegistry(
		WithTagParser("custom", mockParser),
	)

	parser := registry.Get("custom")
	assert.NotNil(t, parser)
	assert.Equal(t, reflect.ValueOf(mockParser).Pointer(), reflect.ValueOf(parser).Pointer())
}

func TestNewTagParserRegistry_MultipleParsers(t *testing.T) {
	registry := NewTagParserRegistry(
		WithTagParser("json", mockParser),
		WithTagParser("xml", mockParser),
		WithTagParser("yaml", mockParser),
	)

	assert.NotNil(t, registry.Get("json"))
	assert.NotNil(t, registry.Get("xml"))
	assert.NotNil(t, registry.Get("yaml"))
	assert.Nil(t, registry.Get("unknown"))
}

func TestNewTagParserRegistry_ParserWithDefault(t *testing.T) {
	registry := NewTagParserRegistry(
		WithTagParser("custom", mockParser, mockDefault),
	)

	parser := registry.Get("custom")
	assert.NotNil(t, parser)

	defaultFunc := registry.GetDefault("custom")
	assert.NotNil(t, defaultFunc)
	assert.Equal(t, reflect.ValueOf(mockDefault).Pointer(), reflect.ValueOf(defaultFunc).Pointer())
}

func TestNewTagParserRegistry_ParserWithoutDefault(t *testing.T) {
	registry := NewTagParserRegistry(
		WithTagParser("custom", mockParser),
	)

	parser := registry.Get("custom")
	assert.NotNil(t, parser)

	defaultFunc := registry.GetDefault("custom")
	assert.Nil(t, defaultFunc)
}

func TestNewTagParserRegistry_MultipleOptions(t *testing.T) {
	registry := NewTagParserRegistry(
		WithTagParser("tag1", mockParser),
		WithTagParser("tag2", mockParser, mockDefault),
		WithTagParser("tag3", mockParser),
	)

	assert.NotNil(t, registry.Get("tag1"))
	assert.NotNil(t, registry.Get("tag2"))
	assert.NotNil(t, registry.Get("tag3"))
	assert.NotNil(t, registry.GetDefault("tag2"))
	assert.Nil(t, registry.GetDefault("tag1"))
	assert.Nil(t, registry.GetDefault("tag3"))
}

func TestNewDefaultTagParserRegistry(t *testing.T) {
	registry := NewDefaultTagParserRegistry()

	// Verify schema parser is registered
	schemaParser := registry.Get("schema")
	assert.NotNil(t, schemaParser)
	assert.Equal(t, reflect.ValueOf(ParseSchemaTag).Pointer(), reflect.ValueOf(schemaParser).Pointer())

	// Verify body parser is registered
	bodyParser := registry.Get("body")
	assert.NotNil(t, bodyParser)
	assert.Equal(t, reflect.ValueOf(ParseBodyTag).Pointer(), reflect.ValueOf(bodyParser).Pointer())

	// Verify schema has default function
	schemaDefault := registry.GetDefault("schema")
	assert.NotNil(t, schemaDefault)
	assert.Equal(t, reflect.ValueOf(DefaultSchemaMetadata).Pointer(), reflect.ValueOf(schemaDefault).Pointer())

	// Verify body has no default function
	bodyDefault := registry.GetDefault("body")
	assert.Nil(t, bodyDefault)
}

func TestTagParserRegistry_Get_ExistingParser(t *testing.T) {
	registry := NewTagParserRegistry(
		WithTagParser("test", mockParser),
	)

	parser := registry.Get("test")
	assert.NotNil(t, parser)
	assert.Equal(t, reflect.ValueOf(mockParser).Pointer(), reflect.ValueOf(parser).Pointer())
}

func TestTagParserRegistry_Get_NonExistentParser(t *testing.T) {
	registry := NewTagParserRegistry()

	parser := registry.Get("nonexistent")
	assert.Nil(t, parser)
}

func TestTagParserRegistry_Get_MultipleRetrievals(t *testing.T) {
	registry := NewTagParserRegistry(
		WithTagParser("test", mockParser),
	)

	parser1 := registry.Get("test")
	parser2 := registry.Get("test")

	assert.Equal(t, reflect.ValueOf(parser1).Pointer(), reflect.ValueOf(parser2).Pointer())
}

func TestTagParserRegistry_GetDefault_ExistingDefault(t *testing.T) {
	registry := NewTagParserRegistry(
		WithTagParser("test", mockParser, mockDefault),
	)

	defaultFunc := registry.GetDefault("test")
	assert.NotNil(t, defaultFunc)
	assert.Equal(t, reflect.ValueOf(mockDefault).Pointer(), reflect.ValueOf(defaultFunc).Pointer())
}

func TestTagParserRegistry_GetDefault_NoDefault(t *testing.T) {
	registry := NewTagParserRegistry(
		WithTagParser("test", mockParser),
	)

	defaultFunc := registry.GetDefault("test")
	assert.Nil(t, defaultFunc)
}

func TestTagParserRegistry_GetDefault_NonExistentTag(t *testing.T) {
	registry := NewTagParserRegistry()

	defaultFunc := registry.GetDefault("nonexistent")
	assert.Nil(t, defaultFunc)
}

func TestTagParserRegistry_All_EmptyRegistry(t *testing.T) {
	registry := NewTagParserRegistry()

	all := registry.All()
	assert.NotNil(t, all)
	assert.Empty(t, all)
}

func TestTagParserRegistry_All_SingleParser(t *testing.T) {
	registry := NewTagParserRegistry(
		WithTagParser("test", mockParser),
	)

	all := registry.All()
	require.Len(t, all, 1)
	assert.NotNil(t, all["test"])
	assert.Equal(t, reflect.ValueOf(mockParser).Pointer(), reflect.ValueOf(all["test"]).Pointer())
}

func TestTagParserRegistry_All_MultipleParsers(t *testing.T) {
	registry := NewTagParserRegistry(
		WithTagParser("tag1", mockParser),
		WithTagParser("tag2", mockParser),
		WithTagParser("tag3", mockParser),
	)

	all := registry.All()
	require.Len(t, all, 3)
	assert.NotNil(t, all["tag1"])
	assert.NotNil(t, all["tag2"])
	assert.NotNil(t, all["tag3"])
}

func TestTagParserRegistry_All_Independence(t *testing.T) {
	registry := NewTagParserRegistry(
		WithTagParser("test", mockParser),
	)

	all := registry.All()
	all["new"] = mockParser

	// Original registry should be unchanged
	assert.Nil(t, registry.Get("new"))
	assert.Len(t, registry.All(), 1)
}

func TestTagParserRegistry_All_MapKeys(t *testing.T) {
	registry := NewTagParserRegistry(
		WithTagParser("json", mockParser),
		WithTagParser("xml", mockParser),
		WithTagParser("yaml", mockParser),
	)

	all := registry.All()
	assert.Contains(t, all, "json")
	assert.Contains(t, all, "xml")
	assert.Contains(t, all, "yaml")
	assert.NotContains(t, all, "unknown")
}

func TestTagParserRegistry_All_MapValues(t *testing.T) {
	registry := NewTagParserRegistry(
		WithTagParser("test", mockParser),
	)

	all := registry.All()
	assert.Equal(t, reflect.ValueOf(mockParser).Pointer(), reflect.ValueOf(all["test"]).Pointer())
}

func TestWithTagParser_BasicRegistration(t *testing.T) {
	registry := NewTagParserRegistry(
		WithTagParser("custom", mockParser),
	)

	parser := registry.Get("custom")
	assert.NotNil(t, parser)
}

func TestWithTagParser_WithDefault(t *testing.T) {
	registry := NewTagParserRegistry(
		WithTagParser("custom", mockParser, mockDefault),
	)

	parser := registry.Get("custom")
	defaultFunc := registry.GetDefault("custom")

	assert.NotNil(t, parser)
	assert.NotNil(t, defaultFunc)
}

func TestWithTagParser_OverrideExisting(t *testing.T) {
	parser1 := func(field reflect.StructField, index int, tagValue string) (any, error) {
		return "parser1", nil
	}
	parser2 := func(field reflect.StructField, index int, tagValue string) (any, error) {
		return "parser2", nil
	}

	registry := NewTagParserRegistry(
		WithTagParser("test", parser1),
		WithTagParser("test", parser2),
	)

	parser := registry.Get("test")
	assert.NotNil(t, parser)
	assert.Equal(t, reflect.ValueOf(parser2).Pointer(), reflect.ValueOf(parser).Pointer())
}

func TestWithTagParser_OverrideWithDefault(t *testing.T) {
	default1 := func(field reflect.StructField, index int) any {
		return "default11"
	}
	default2 := func(field reflect.StructField, index int) any {
		return "default2"
	}

	registry := NewTagParserRegistry(
		WithTagParser("test", mockParser, default1),
		WithTagParser("test", mockParser, default2),
	)

	defaultFunc := registry.GetDefault("test")
	assert.NotNil(t, defaultFunc)
	assert.Equal(t, reflect.ValueOf(default2).Pointer(), reflect.ValueOf(defaultFunc).Pointer())
}

func TestWithTagParser_KeepDefaultWhenNotProvided(t *testing.T) {
	registry := NewTagParserRegistry(
		WithTagParser("test", mockParser, mockDefault),
		WithTagParser("test", mockParser), // Register without default - should keep existing
	)

	// Default should still exist because we didn't explicitly remove it
	defaultFunc := registry.GetDefault("test")
	assert.NotNil(t, defaultFunc)
	assert.Equal(t, reflect.ValueOf(mockDefault).Pointer(), reflect.ValueOf(defaultFunc).Pointer())
}

func TestWithTagParser_MultipleDefaults(t *testing.T) {
	default1 := func(field reflect.StructField, index int) any {
		return "default1"
	}
	default2 := func(field reflect.StructField, index int) any {
		return "default2"
	}

	registry := NewTagParserRegistry(
		WithTagParser("tag1", mockParser, default1),
		WithTagParser("tag2", mockParser, default2),
	)

	assert.NotNil(t, registry.GetDefault("tag1"))
	assert.NotNil(t, registry.GetDefault("tag2"))
	assert.Equal(t, reflect.ValueOf(default1).Pointer(), reflect.ValueOf(registry.GetDefault("tag1")).Pointer())
	assert.Equal(t, reflect.ValueOf(default2).Pointer(), reflect.ValueOf(registry.GetDefault("tag2")).Pointer())
}

func TestTagParserRegistry_Integration_CreateGetAll(t *testing.T) {
	registry := NewTagParserRegistry(
		WithTagParser("json", mockParser),
		WithTagParser("xml", mockParser),
	)

	// Get individual parsers
	jsonParser := registry.Get("json")
	xmlParser := registry.Get("xml")

	assert.NotNil(t, jsonParser)
	assert.NotNil(t, xmlParser)

	// Get all parsers
	all := registry.All()
	require.Len(t, all, 2)
	assert.NotNil(t, all["json"])
	assert.NotNil(t, all["xml"])
}

func TestTagParserRegistry_Integration_DefaultWorkflow(t *testing.T) {
	registry := NewTagParserRegistry(
		WithTagParser("tag1", mockParser, mockDefault),
		WithTagParser("tag2", mockParser),
	)

	// Get defaults
	default1 := registry.GetDefault("tag1")
	default2 := registry.GetDefault("tag2")

	assert.NotNil(t, default1)
	assert.Nil(t, default2)
}

func TestTagParserRegistry_Integration_OverrideWorkflow(t *testing.T) {
	parser1 := func(field reflect.StructField, index int, tagValue string) (any, error) {
		return "old", nil
	}
	parser2 := func(field reflect.StructField, index int, tagValue string) (any, error) {
		return "new", nil
	}

	registry := NewTagParserRegistry(
		WithTagParser("test", parser1),
		WithTagParser("test", parser2),
	)

	parser := registry.Get("test")
	assert.Equal(t, reflect.ValueOf(parser2).Pointer(), reflect.ValueOf(parser).Pointer())
}

func TestTagParserRegistry_Integration_MixedRegistration(t *testing.T) {
	default1 := func(field reflect.StructField, index int) any {
		return "default1"
	}

	registry := NewTagParserRegistry(
		WithTagParser("with_default", mockParser, default1),
		WithTagParser("without_default", mockParser),
	)

	assert.NotNil(t, registry.Get("with_default"))
	assert.NotNil(t, registry.Get("without_default"))
	assert.NotNil(t, registry.GetDefault("with_default"))
	assert.Nil(t, registry.GetDefault("without_default"))
}

func TestTagParserRegistry_CustomTagParser(t *testing.T) {
	customParser := func(field reflect.StructField, index int, tagValue string) (any, error) {
		return map[string]string{"custom": tagValue}, nil
	}

	registry := NewTagParserRegistry(
		WithTagParser("json", customParser),
	)

	parser := registry.Get("json")
	assert.NotNil(t, parser)
	assert.Equal(t, reflect.ValueOf(customParser).Pointer(), reflect.ValueOf(parser).Pointer())
}

func TestTagParserRegistry_MultipleCustomTags(t *testing.T) {
	jsonParser := func(field reflect.StructField, index int, tagValue string) (any, error) {
		return "json", nil
	}
	xmlParser := func(field reflect.StructField, index int, tagValue string) (any, error) {
		return "xml", nil
	}
	yamlParser := func(field reflect.StructField, index int, tagValue string) (any, error) {
		return "yaml", nil
	}

	registry := NewTagParserRegistry(
		WithTagParser("json", jsonParser),
		WithTagParser("xml", xmlParser),
		WithTagParser("yaml", yamlParser),
	)

	assert.NotNil(t, registry.Get("json"))
	assert.NotNil(t, registry.Get("xml"))
	assert.NotNil(t, registry.Get("yaml"))

	all := registry.All()
	require.Len(t, all, 3)
}

func TestTagParserRegistry_ManyParsers(t *testing.T) {
	opts := make([]TagParserRegistryOption, 0, 15)
	for i := 0; i < 15; i++ {
		tagName := string(rune('a' + i))
		opts = append(opts, WithTagParser(tagName, mockParser))
	}

	registry := NewTagParserRegistry(opts...)

	all := registry.All()
	require.Len(t, all, 15)

	for i := 0; i < 15; i++ {
		tagName := string(rune('a' + i))
		assert.NotNil(t, registry.Get(tagName))
		assert.Contains(t, all, tagName)
	}
}
