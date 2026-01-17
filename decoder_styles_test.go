package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecoder_DecodeFormStyle(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    map[string]any
		wantErr bool
	}{
		{
			name: "single key-value",
			data: "name=john",
			want: map[string]any{
				"name": "john",
			},
		},
		{
			name: "multiple key-values",
			data: "name=john&age=30&city=NYC",
			want: map[string]any{
				"name": "john",
				"age":  "30",
				"city": "NYC",
			},
		},
		{
			name: "comma separated values",
			data: "ids=1,2,3",
			want: map[string]any{
				"ids": []any{"1", "2", "3"},
			},
		},
		{
			name: "repeated keys (explode)",
			data: "ids=1&ids=2&ids=3",
			want: map[string]any{
				"ids": []any{"1", "2", "3"},
			},
		},
		{
			name: "dotted notation (nested)",
			data: "filter.type=car&filter.color=red",
			want: map[string]any{
				"filter": map[string]any{
					"type":  "car",
					"color": "red",
				},
			},
		},
		{
			name: "deeply nested dotted notation",
			data: "user.profile.name=john&user.profile.age=30",
			want: map[string]any{
				"user": map[string]any{
					"profile": map[string]any{
						"name": "john",
						"age":  "30",
					},
				},
			},
		},
		{
			name: "empty string",
			data: "",
			want: map[string]any{},
		},
		{
			name: "empty value",
			data: "name=",
			want: map[string]any{},
		},
		{
			name: "URL encoded values",
			data: "name=John%20Doe&email=john%40example.com",
			want: map[string]any{
				"name":  "John Doe",
				"email": "john@example.com",
			},
		},
		{
			name: "special characters",
			data: "query=hello+world&tag=%23golang",
			want: map[string]any{
				"query": "hello world",
				"tag":   "#golang",
			},
		},
	}

	decoder := newTestDecoder()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := decoder.decodeFormStyle(tt.data)

			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestDecoder_DecodeSimpleStyle(t *testing.T) {
	tests := []struct {
		name string
		data string
		want any
	}{
		{
			name: "simple value",
			data: "john",
			want: "john",
		},
		{
			name: "empty string",
			data: "",
			want: "",
		},
		{
			name: "value with special chars",
			data: "hello-world_123",
			want: "hello-world_123",
		},
		{
			name: "comma separated (returned as-is)",
			data: "1,2,3",
			want: "1,2,3",
		},
	}

	decoder := newTestDecoder()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := decoder.decodeSimpleStyle(tt.data)

			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestDecoder_DecodeLabelStyle(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		explode bool
		want    any
	}{
		{
			name:    "single value non-exploded",
			data:    ".red",
			explode: false,
			want:    "red",
		},
		{
			name:    "array non-exploded (comma-separated)",
			data:    ".1,2,3",
			explode: false,
			want:    []any{"1", "2", "3"},
		},
		{
			name:    "object non-exploded (comma-separated key-value pairs)",
			data:    ".x,1024,y,768",
			explode: false,
			want: map[string]any{
				"x": "1024",
				"y": "768",
			},
		},
		{
			name:    "array exploded (period-separated)",
			data:    ".1.2.3",
			explode: true,
			want:    []any{"1", "2", "3"},
		},
		{
			name:    "object exploded (period-separated key-value pairs)",
			data:    ".x.1024.y.768",
			explode: true,
			want: map[string]any{
				"x": "1024",
				"y": "768",
			},
		},
		{
			name:    "empty string non-exploded",
			data:    ".",
			explode: false,
			want:    nil,
		},
		{
			name:    "single value exploded",
			data:    ".red",
			explode: true,
			want:    "red",
		},
		{
			name:    "without leading dot (handled)",
			data:    "red",
			explode: false,
			want:    "red",
		},
	}

	decoder := newTestDecoder()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := decoder.decodeLabelStyle(tt.data, tt.explode)

			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestDecoder_DecodeMatrixStyle(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		explode bool
		want    map[string]any
		wantErr bool
	}{
		{
			name:    "single key-value non-exploded",
			data:    ";color=blue",
			explode: false,
			want: map[string]any{
				"color": "blue",
			},
		},
		{
			name:    "array non-exploded (comma-separated)",
			data:    ";ids=1,2,3",
			explode: false,
			want: map[string]any{
				"ids": []any{"1", "2", "3"},
			},
		},
		{
			name:    "array exploded (repeated keys)",
			data:    ";ids=1;ids=2;ids=3",
			explode: true,
			want: map[string]any{
				"ids": []any{"1", "2", "3"},
			},
		},
		{
			name:    "multiple keys non-exploded",
			data:    ";color=blue;size=large",
			explode: false,
			want: map[string]any{
				"color": "blue",
				"size":  "large",
			},
		},
		{
			name:    "multiple keys exploded",
			data:    ";color=blue;size=large",
			explode: true,
			want: map[string]any{
				"color": []any{"blue"},
				"size":  []any{"large"},
			},
		},
		{
			name:    "empty string",
			data:    ";",
			explode: false,
			want:    map[string]any{},
		},
		{
			name:    "without leading semicolon",
			data:    "color=blue",
			explode: false,
			want: map[string]any{
				"color": "blue",
			},
		},
		{
			name:    "invalid format - no equals sign",
			data:    ";color",
			explode: false,
			want:    nil,
			wantErr: true,
		},
	}

	decoder := newTestDecoder()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := decoder.decodeMatrixStyle(tt.data, tt.explode)

			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestDecoder_DecodeDelimitedStyles(t *testing.T) {
	decoder := newTestDecoder()

	// Define decoder functions to test
	decoders := []struct {
		name      string
		decodeFn  func(string) (map[string]any, error)
		separator string
		tests     []struct {
			name    string
			data    string
			want    map[string]any
			wantErr bool
		}
	}{
		{
			name:      "space_delimited",
			decodeFn:  func(q string) (map[string]any, error) { return decoder.decodeSpaceDelimited(q) },
			separator: " ",
			tests: []struct {
				name    string
				data    string
				want    map[string]any
				wantErr bool
			}{
				{
					name: "single key with separated values",
					data: "ids=1%202%203",
					want: map[string]any{"ids": []any{"1", "2", "3"}},
				},
				{
					name: "single value",
					data: "ids=1",
					want: map[string]any{"ids": "1"},
				},
				{
					name: "multiple keys",
					data: "ids=1%202&names=john%20jane",
					want: map[string]any{"ids": []any{"1", "2"}, "names": []any{"john", "jane"}},
				},
				{
					name: "empty string",
					data: "",
					want: map[string]any{},
				},
			},
		},
		{
			name:      "pipe_delimited",
			decodeFn:  func(q string) (map[string]any, error) { return decoder.decodePipeDelimited(q) },
			separator: "|",
			tests: []struct {
				name    string
				data    string
				want    map[string]any
				wantErr bool
			}{
				{
					name: "single key with separated values",
					data: "ids=1%7C2%7C3",
					want: map[string]any{"ids": []any{"1", "2", "3"}},
				},
				{
					name: "single value",
					data: "ids=1",
					want: map[string]any{"ids": "1"},
				},
				{
					name: "multiple keys",
					data: "ids=1%7C2&names=john%7Cjane",
					want: map[string]any{"ids": []any{"1", "2"}, "names": []any{"john", "jane"}},
				},
				{
					name: "empty string",
					data: "",
					want: map[string]any{},
				},
			},
		},
	}

	for _, dec := range decoders {
		t.Run(dec.name, func(t *testing.T) {
			for _, tt := range dec.tests {
				t.Run(tt.name, func(t *testing.T) {
					result, err := dec.decodeFn(tt.data)

					if tt.wantErr {
						require.Error(t, err)

						return
					}

					require.NoError(t, err)
					assert.Equal(t, tt.want, result)
				})
			}
		})
	}
}

func TestDecoder_DecodeDeepObject(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    map[string]any
		wantErr bool
	}{
		{
			name: "simple deep object",
			data: "filter%5Btype%5D=car",
			want: map[string]any{
				"filter": map[string]any{
					"type": "car",
				},
			},
		},
		{
			name: "multiple nested properties",
			data: "filter%5Btype%5D=car&filter%5Bcolor%5D=red",
			want: map[string]any{
				"filter": map[string]any{
					"type":  "car",
					"color": "red",
				},
			},
		},
		{
			name: "deeply nested",
			data: "user%5Bprofile%5D%5Baddress%5D%5Bcity%5D=NYC",
			want: map[string]any{
				"user": map[string]any{
					"profile": map[string]any{
						"address": map[string]any{
							"city": "NYC",
						},
					},
				},
			},
		},
		{
			name: "multiple values for same key",
			data: "ids%5B%5D=1&ids%5B%5D=2&ids%5B%5D=3",
			want: map[string]any{
				"ids": map[string]any{
					"": []any{"1", "2", "3"},
				},
			},
		},
		{
			name: "mixed bracket and regular keys",
			data: "filter%5Btype%5D=car&sort=name",
			want: map[string]any{
				"filter": map[string]any{
					"type": "car",
				},
				"sort": "name",
			},
		},
		{
			name: "empty string",
			data: "",
			want: map[string]any{},
		},
		{
			name: "regular key-value (no brackets)",
			data: "name=john",
			want: map[string]any{
				"name": "john",
			},
		},
	}

	decoder := newTestDecoder()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := decoder.decodeDeepObject(tt.data)

			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestDecoder_ProcessFormValue(t *testing.T) {
	tests := []struct {
		name     string
		valSlice []string
		want     any
	}{
		{
			name:     "empty slice",
			valSlice: []string{},
			want:     nil,
		},
		{
			name:     "single value",
			valSlice: []string{"john"},
			want:     "john",
		},
		{
			name:     "single empty value",
			valSlice: []string{""},
			want:     nil,
		},
		{
			name:     "multiple values",
			valSlice: []string{"john", "jane"},
			want:     []any{"john", "jane"},
		},
		{
			name:     "comma separated in single value",
			valSlice: []string{"1,2,3"},
			want:     []any{"1", "2", "3"},
		},
		{
			name:     "value with comma at end",
			valSlice: []string{"a,b,"},
			want:     []any{"a", "b"},
		},
	}

	decoder := newTestDecoder()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := decoder.processFormValue(tt.valSlice)

			assert.Equal(t, tt.want, result)
		})
	}
}

func TestDecoder_DecodeDelimited(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		sep     string
		want    map[string]any
		wantErr bool
	}{
		{
			name:  "pipe delimiter",
			query: "ids=1%7C2%7C3",
			sep:   "|",
			want: map[string]any{
				"ids": []any{"1", "2", "3"},
			},
		},
		{
			name:  "space delimiter",
			query: "ids=1%202%203",
			sep:   " ",
			want: map[string]any{
				"ids": []any{"1", "2", "3"},
			},
		},
		{
			name:  "custom delimiter",
			query: "ids=1%3B2%3B3",
			sep:   ";",
			want: map[string]any{
				"ids": []any{"1", "2", "3"},
			},
		},
		{
			name:  "single value (no delimiter)",
			query: "ids=1",
			sep:   "|",
			want: map[string]any{
				"ids": "1",
			},
		},
		{
			name:  "empty query",
			query: "",
			sep:   "|",
			want:  map[string]any{},
		},
		{
			name:  "empty value",
			query: "ids=",
			sep:   "|",
			want:  map[string]any{},
		},
	}

	decoder := newTestDecoder()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := decoder.decodeDelimited(tt.query, tt.sep)

			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestDecoder_DecodeValueByStyle(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		style   Style
		explode bool
		want    any
		wantErr bool
	}{
		{
			name:    "simple style",
			value:   "john",
			style:   StyleSimple,
			explode: false,
			want:    "john",
		},
		{
			name:    "label style non-exploded",
			value:   ".red",
			style:   StyleLabel,
			explode: false,
			want:    "red",
		},
		{
			name:    "label style exploded array",
			value:   ".1.2.3",
			style:   StyleLabel,
			explode: true,
			want:    []any{"1", "2", "3"},
		},
		{
			name:    "invalid style for single value - form",
			value:   "test",
			style:   StyleForm,
			explode: false,
			wantErr: true,
		},
		{
			name:    "invalid style for single value - matrix",
			value:   "test",
			style:   StyleMatrix,
			explode: false,
			wantErr: true,
		},
		{
			name:    "invalid style for single value - deepObject",
			value:   "test",
			style:   StyleDeepObject,
			explode: false,
			wantErr: true,
		},
		{
			name:    "unknown style",
			value:   "test",
			style:   "unknown",
			explode: false,
			wantErr: true,
		},
	}

	decoder := newTestDecoder()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := decoder.decodeValueByStyle(tt.value, tt.style, tt.explode)

			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestDecoder_DecodeByStyle(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		style   Style
		explode bool
		want    map[string]any
		wantErr bool
	}{
		{
			name:    "form style",
			value:   "name=john&age=30",
			style:   StyleForm,
			explode: true,
			want: map[string]any{
				"name": "john",
				"age":  "30",
			},
		},
		{
			name:    "matrix style",
			value:   ";color=blue;size=large",
			style:   StyleMatrix,
			explode: false,
			want: map[string]any{
				"color": "blue",
				"size":  "large",
			},
		},
		{
			name:    "space delimited",
			value:   "ids=1%202%203",
			style:   StyleSpaceDelimited,
			explode: false,
			want: map[string]any{
				"ids": []any{"1", "2", "3"},
			},
		},
		{
			name:    "pipe delimited",
			value:   "ids=1%7C2%7C3",
			style:   StylePipeDelimited,
			explode: false,
			want: map[string]any{
				"ids": []any{"1", "2", "3"},
			},
		},
		{
			name:    "deep object",
			value:   "filter%5Btype%5D=car",
			style:   StyleDeepObject,
			explode: true,
			want: map[string]any{
				"filter": map[string]any{
					"type": "car",
				},
			},
		},
		{
			name:    "invalid style for map - simple",
			value:   "test",
			style:   StyleSimple,
			explode: false,
			wantErr: true,
		},
		{
			name:    "invalid style for map - label",
			value:   ".test",
			style:   StyleLabel,
			explode: false,
			wantErr: true,
		},
		{
			name:    "unknown style",
			value:   "test",
			style:   "unknown",
			explode: false,
			wantErr: true,
		},
	}

	decoder := newTestDecoder()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := decoder.decodeByStyle(tt.value, tt.style, tt.explode)

			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestDecoder_StyleDecoders_UsesTagNames(t *testing.T) {
	decoder := newTestDecoder()

	tests := []struct {
		name     string
		data     string
		style    Style
		explode  bool
		decoder  func(string) (map[string]any, error)
		expected map[string]any
	}{
		{
			name:  "form style - uses tag names (not field names)",
			data:  "user_name=John&age=30&email_address=john@example.com",
			style: StyleForm,
			decoder: func(d string) (map[string]any, error) {
				return decoder.decodeFormStyle(d)
			},
			expected: map[string]any{
				"user_name":     "John",             // Tag name, not "UserName"
				"age":           "30",               // Tag name, not "UserAge"
				"email_address": "john@example.com", // Tag name, not "Email"
			},
		},
		{
			name:    "matrix style - uses tag names (not field names)",
			data:    ";user_name=John;age=30",
			style:   StyleMatrix,
			explode: true,
			decoder: func(d string) (map[string]any, error) {
				return decoder.decodeMatrixStyle(d, true)
			},
			expected: map[string]any{
				"user_name": []any{"John"}, // Tag name, not "UserName" (explode=true creates arrays)
				"age":       []any{"30"},   // Tag name, not "UserAge"
			},
		},
		{
			name:  "space delimited - uses tag names (not field names)",
			data:  "user_name=1 2 3&age=30 40",
			style: StyleSpaceDelimited,
			decoder: func(d string) (map[string]any, error) {
				return decoder.decodeSpaceDelimited(d)
			},
			expected: map[string]any{
				"user_name": []any{"1", "2", "3"}, // Tag name, not "UserName"
				"age":       []any{"30", "40"},    // Tag name, not "UserAge"
			},
		},
		{
			name:  "pipe delimited - uses tag names (not field names)",
			data:  "user_name=1|2|3&age=30|40",
			style: StylePipeDelimited,
			decoder: func(d string) (map[string]any, error) {
				return decoder.decodePipeDelimited(d)
			},
			expected: map[string]any{
				"user_name": []any{"1", "2", "3"}, // Tag name, not "UserName"
				"age":       []any{"30", "40"},    // Tag name, not "UserAge"
			},
		},
		{
			name:  "deep object - uses tag names in bracket notation",
			data:  "user_name[first]=John&user_name[last]=Doe&age[value]=30",
			style: StyleDeepObject,
			decoder: func(d string) (map[string]any, error) {
				return decoder.decodeDeepObject(d)
			},
			expected: map[string]any{
				"user_name": map[string]any{ // Tag name, not "UserName"
					"first": "John",
					"last":  "Doe",
				},
				"age": map[string]any{ // Tag name, not "UserAge"
					"value": "30",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.decoder(tt.data)
			require.NoError(t, err)

			// Verify that keys are tag names (ParamName), not field names (MapKey)
			for expectedKey, expectedValue := range tt.expected {
				actualValue, exists := result[expectedKey]
				assert.True(t, exists, "Expected key %q (tag name) should exist in result", expectedKey)
				assert.Equal(t, expectedValue, actualValue, "Value for key %q should match", expectedKey)
			}

			// Verify that field names (MapKey) are NOT in the result
			fieldNames := []string{"UserName", "UserAge", "Email"}
			for _, fieldName := range fieldNames {
				_, exists := result[fieldName]
				assert.False(t, exists, "Field name %q should NOT be in result (should use tag name instead)", fieldName)
			}
		})
	}
}

// TestDecoder_StyleDecoders_UnknownFields tests that unknown fields
// are included (filtering happens at higher level, not in style decoders).
func TestDecoder_StyleDecoders_UnknownFields(t *testing.T) {
	decoder := newTestDecoder()

	tests := []struct {
		name     string
		data     string
		decoder  func(string) (map[string]any, error)
		expected map[string]any
	}{
		{
			name: "form style - all fields included (no filtering at style level)",
			data: "user_name=John&unknown_field=value&another_unknown=test",
			decoder: func(d string) (map[string]any, error) {
				return decoder.decodeFormStyle(d)
			},
			expected: map[string]any{
				"user_name":       "John",  // Tag name, not "UserName"
				"unknown_field":   "value", // Unknown fields are not filtered at style decoder level
				"another_unknown": "test",  // Unknown fields are not filtered at style decoder level
			},
		},
		{
			name: "form style - nested unknown objects included",
			data: "user_name=John&nested.unknown=value",
			decoder: func(d string) (map[string]any, error) {
				return decoder.decodeFormStyle(d)
			},
			expected: map[string]any{
				"user_name": "John", // Tag name, not "UserName"
				"nested": map[string]any{
					"unknown": "value", // Nested unknown fields are included
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.decoder(tt.data)
			require.NoError(t, err)

			for expectedKey, expectedValue := range tt.expected {
				actualValue, exists := result[expectedKey]
				assert.True(t, exists, "Expected key %q should exist in result", expectedKey)
				assert.Equal(t, expectedValue, actualValue, "Value for key %q should match", expectedKey)
			}
		})
	}
}

// TestDecoder_StyleDecoders_NestedStructures tests that nested structures
// use tag names at all levels (not field names).
func TestDecoder_StyleDecoders_NestedStructures(t *testing.T) {
	decoder := newTestDecoder()

	tests := []struct {
		name     string
		data     string
		decoder  func(string) (map[string]any, error)
		expected map[string]any
	}{
		{
			name: "form style - nested dotted notation uses tag names",
			data: "filter.type1=car&filter.color=red&user.name=John&user.age=30",
			decoder: func(d string) (map[string]any, error) {
				return decoder.decodeFormStyle(d)
			},
			expected: map[string]any{
				"filter": map[string]any{
					"type1": "car", // Tag name, not "Type"
					"color": "red", // Tag name, not "Color"
				},
				"user": map[string]any{
					"name": "John", // Tag name, not "Name"
					"age":  "30",   // Tag name, not "Age"
				},
			},
		},
		{
			name: "deep object - nested bracket notation uses tag names",
			data: "filter[type1]=car&filter[color]=red&user[name]=John&user[age]=30",
			decoder: func(d string) (map[string]any, error) {
				return decoder.decodeDeepObject(d)
			},
			expected: map[string]any{
				"filter": map[string]any{
					"type1": "car", // Tag name, not "Type"
					"color": "red", // Tag name, not "Color"
				},
				"user": map[string]any{
					"name": "John", // Tag name, not "Name"
					"age":  "30",   // Tag name, not "Age"
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.decoder(tt.data)
			require.NoError(t, err)

			// Verify nested structure with tag names (not field names)
			assertNestedMap(t, result, tt.expected, "")

			// Verify that field names are NOT present at any level
			assertNoFieldNames(t, result, []string{"Filter", "Type", "Color", "User", "Name", "Age"})
		})
	}
}

// assertNestedMap recursively asserts that nested maps match expected structure.
func assertNestedMap(t *testing.T, actual, expected map[string]any, path string) {
	t.Helper()
	for key, expectedValue := range expected {
		fullPath := key
		if path != "" {
			fullPath = path + "." + key
		}

		actualValue, exists := actual[key]
		assert.True(t, exists, "Expected key %q should exist at path %q", key, path)

		if expectedMap, ok := expectedValue.(map[string]any); ok {
			actualMap, ok := actualValue.(map[string]any)
			assert.True(t, ok, "Value at %q should be a map, got %T", fullPath, actualValue)
			assertNestedMap(t, actualMap, expectedMap, fullPath)
		} else {
			assert.Equal(t, expectedValue, actualValue, "Value at %q should match", fullPath)
		}
	}
}

// assertNoFieldNames recursively checks that field names are not present in the map.
func assertNoFieldNames(t *testing.T, m map[string]any, fieldNames []string) {
	t.Helper()
	for key, value := range m {
		for _, fieldName := range fieldNames {
			assert.NotEqual(t, fieldName, key, "Field name %q should not be present as key", fieldName)
		}

		// Recursively check nested maps
		if nestedMap, ok := value.(map[string]any); ok {
			assertNoFieldNames(t, nestedMap, fieldNames)
		}
	}
}
