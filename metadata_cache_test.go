package schema

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetadataCache_GetReturnsMetadata(t *testing.T) {
	type User struct {
		Name string `schema:"name"`
		Age  int    `schema:"age"`
	}

	metadata := NewDefaultMetadata()
	typ := reflect.TypeOf(User{})
	structMeta, err := metadata.GetStructMetadata(typ)

	require.NoError(t, err)
	require.NotNil(t, structMeta)
	require.Greater(t, len(structMeta.Fields), 0)
}

func TestMetadataCache_HandlesDifferentTypes(t *testing.T) {
	type User struct {
		Name string `schema:"name"`
		Age  int    `schema:"age"`
	}

	type Product struct {
		Name  string `schema:"name"`
		Price int    `schema:"price"`
	}

	metadata := NewDefaultMetadata()

	// Test different types work
	userType := reflect.TypeOf(User{})
	userMeta, err := metadata.GetStructMetadata(userType)
	require.NoError(t, err)
	require.NotNil(t, userMeta)

	productType := reflect.TypeOf(Product{})
	productMeta, err := metadata.GetStructMetadata(productType)
	require.NoError(t, err)

	// Different types should return different metadata
	assert.NotEqual(t, userMeta, productMeta)
}

func TestMetadataCache_CacheReuse(t *testing.T) {
	type User struct {
		Name string `schema:"name"`
	}

	metadata := NewDefaultMetadata()
	typ := reflect.TypeOf(User{})

	structMeta1, err1 := metadata.GetStructMetadata(typ)
	require.NoError(t, err1)

	structMeta2, err2 := metadata.GetStructMetadata(typ)
	require.NoError(t, err2)

	// Should return same pointer (cached)
	assert.Equal(t, structMeta1, structMeta2)
}
