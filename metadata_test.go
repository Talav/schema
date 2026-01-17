package schema

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateField(t *testing.T) {
	typ := reflect.TypeOf("")

	t.Run("valid field", func(t *testing.T) {
		field := FieldMetadata{
			StructFieldName: "Name",
			Type:            typ,
			Index:           0,
		}
		err := validateField(field)
		assert.NoError(t, err)
	})

	t.Run("empty StructFieldName", func(t *testing.T) {
		field := FieldMetadata{
			StructFieldName: "",
			Type:            typ,
			Index:           0,
		}
		err := validateField(field)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "structFieldName cannot be empty")
	})

	t.Run("nil Type", func(t *testing.T) {
		field := FieldMetadata{
			StructFieldName: "Name",
			Type:            nil,
			Index:           0,
		}
		err := validateField(field)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "type cannot be nil")
	})

	t.Run("negative Index", func(t *testing.T) {
		field := FieldMetadata{
			StructFieldName: "Name",
			Type:            typ,
			Index:           -1,
		}
		err := validateField(field)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "index must be non-negative")
	})

	t.Run("multiple errors", func(t *testing.T) {
		field := FieldMetadata{
			StructFieldName: "",
			Type:            nil,
			Index:           -1,
		}
		err := validateField(field)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "structFieldName cannot be empty")
		assert.Contains(t, err.Error(), "type cannot be nil")
		assert.Contains(t, err.Error(), "index must be non-negative")
	})
}
