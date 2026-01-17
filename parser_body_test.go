package schema

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseBodyTag(t *testing.T) {
	tests := []struct {
		name        string
		fieldName   string
		tagValue    string
		want        *BodyMetadata
		wantErr     bool
		errContains string
	}{
		{
			name:      "empty tag defaults to structured",
			fieldName: "Body",
			tagValue:  "",
			want: &BodyMetadata{
				MapKey:   "Body",
				BodyType: BodyTypeStructured,
				Required: false,
			},
		},
		{
			name:      "structured body type",
			fieldName: "Body",
			tagValue:  "structured",
			want: &BodyMetadata{
				MapKey:   "Body",
				BodyType: BodyTypeStructured,
				Required: false,
			},
		},
		{
			name:      "file body type",
			fieldName: "File",
			tagValue:  "file",
			want: &BodyMetadata{
				MapKey:   "File",
				BodyType: BodyTypeFile,
				Required: false,
			},
		},
		{
			name:      "multipart body type",
			fieldName: "Form",
			tagValue:  "multipart",
			want: &BodyMetadata{
				MapKey:   "Form",
				BodyType: BodyTypeMultipart,
				Required: false,
			},
		},
		{
			name:      "structured with required true",
			fieldName: "Body",
			tagValue:  "structured,required=true",
			want: &BodyMetadata{
				MapKey:   "Body",
				BodyType: BodyTypeStructured,
				Required: true,
			},
		},
		{
			name:      "structured with required flag form",
			fieldName: "Body",
			tagValue:  "structured,required",
			want: &BodyMetadata{
				MapKey:   "Body",
				BodyType: BodyTypeStructured,
				Required: true,
			},
		},
		{
			name:      "file with required true",
			fieldName: "File",
			tagValue:  "file,required=true",
			want: &BodyMetadata{
				MapKey:   "File",
				BodyType: BodyTypeFile,
				Required: true,
			},
		},
		{
			name:      "multipart with required true",
			fieldName: "Form",
			tagValue:  "multipart,required=true",
			want: &BodyMetadata{
				MapKey:   "Form",
				BodyType: BodyTypeMultipart,
				Required: true,
			},
		},
		{
			name:      "structured with required false",
			fieldName: "Body",
			tagValue:  "structured,required=false",
			want: &BodyMetadata{
				MapKey:   "Body",
				BodyType: BodyTypeStructured,
				Required: false,
			},
		},
		{
			name:        "invalid body type",
			fieldName:   "Body",
			tagValue:    "invalid",
			wantErr:     true,
			errContains: "invalid body type",
		},
		{
			name:        "invalid body type with error message",
			fieldName:   "Body",
			tagValue:    "json",
			wantErr:     true,
			errContains: "must be 'structured', 'file', or 'multipart'",
		},
		{
			name:      "case sensitive body type",
			fieldName: "Body",
			tagValue:  "Structured",
			wantErr:   true,
		},
		{
			name:      "file with empty required value",
			fieldName: "File",
			tagValue:  "file,required",
			want: &BodyMetadata{
				MapKey:   "File",
				BodyType: BodyTypeFile,
				Required: true,
			},
		},
		{
			name:      "multipart with empty required value",
			fieldName: "Form",
			tagValue:  "multipart,required",
			want: &BodyMetadata{
				MapKey:   "Form",
				BodyType: BodyTypeMultipart,
				Required: true,
			},
		},
		{
			name:      "long field name",
			fieldName: "VeryLongFieldNameForBody",
			tagValue:  "structured",
			want: &BodyMetadata{
				MapKey:   "VeryLongFieldNameForBody",
				BodyType: BodyTypeStructured,
				Required: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := reflect.StructField{
				Name: tt.fieldName,
				Type: reflect.TypeFor[string](),
			}

			result, err := ParseBodyTag(field, 0, tt.tagValue)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}

				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			meta, ok := result.(*BodyMetadata)
			require.True(t, ok, "result should be *BodyMetadata")

			assert.Equal(t, tt.want.MapKey, meta.MapKey)
			assert.Equal(t, tt.want.BodyType, meta.BodyType)
			assert.Equal(t, tt.want.Required, meta.Required)
		})
	}
}

func TestParseBodyTag_DifferentFieldTypes(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		fieldType reflect.Type
		tagValue  string
		want      *BodyMetadata
	}{
		{
			name:      "string field",
			fieldName: "Body",
			fieldType: reflect.TypeFor[string](),
			tagValue:  "structured",
			want: &BodyMetadata{
				MapKey:   "Body",
				BodyType: BodyTypeStructured,
				Required: false,
			},
		},
		{
			name:      "byte slice field for file",
			fieldName: "File",
			fieldType: reflect.TypeFor[[]byte](),
			tagValue:  "file",
			want: &BodyMetadata{
				MapKey:   "File",
				BodyType: BodyTypeFile,
				Required: false,
			},
		},
		{
			name:      "struct field for structured",
			fieldName: "Body",
			fieldType: reflect.TypeFor[struct{}](),
			tagValue:  "structured",
			want: &BodyMetadata{
				MapKey:   "Body",
				BodyType: BodyTypeStructured,
				Required: false,
			},
		},
		{
			name:      "interface field",
			fieldName: "Body",
			fieldType: reflect.TypeFor[any](),
			tagValue:  "structured",
			want: &BodyMetadata{
				MapKey:   "Body",
				BodyType: BodyTypeStructured,
				Required: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := reflect.StructField{
				Name: tt.fieldName,
				Type: tt.fieldType,
			}

			result, err := ParseBodyTag(field, 0, tt.tagValue)
			require.NoError(t, err)
			require.NotNil(t, result)

			meta, ok := result.(*BodyMetadata)
			require.True(t, ok)

			assert.Equal(t, tt.want.MapKey, meta.MapKey)
			assert.Equal(t, tt.want.BodyType, meta.BodyType)
			assert.Equal(t, tt.want.Required, meta.Required)
		})
	}
}

func TestParseBodyTag_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		fieldName   string
		tagValue    string
		want        *BodyMetadata
		wantErr     bool
		errContains string
	}{
		{
			name:      "empty string tag",
			fieldName: "Body",
			tagValue:  "",
			want: &BodyMetadata{
				MapKey:   "Body",
				BodyType: BodyTypeStructured,
				Required: false,
			},
		},
		{
			name:      "whitespace only tag",
			fieldName: "Body",
			tagValue:  "   ",
			want: &BodyMetadata{
				MapKey:   "Body",
				BodyType: BodyTypeStructured,
				Required: false,
			},
		},
		{
			name:      "multiple required flags",
			fieldName: "Body",
			tagValue:  "structured,required,required",
			want: &BodyMetadata{
				MapKey:   "Body",
				BodyType: BodyTypeStructured,
				Required: true,
			},
		},
		{
			name:      "required false overrides flag",
			fieldName: "Body",
			tagValue:  "structured,required,required=false",
			want: &BodyMetadata{
				MapKey:   "Body",
				BodyType: BodyTypeStructured,
				Required: false,
			},
		},
		{
			name:        "invalid body type with options",
			fieldName:   "Body",
			tagValue:    "invalid,required=true",
			wantErr:     true,
			errContains: "invalid body type",
		},
		{
			name:      "structured with unknown option",
			fieldName: "Body",
			tagValue:  "structured,unknown=value",
			want: &BodyMetadata{
				MapKey:   "Body",
				BodyType: BodyTypeStructured,
				Required: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := reflect.StructField{
				Name: tt.fieldName,
				Type: reflect.TypeFor[string](),
			}

			result, err := ParseBodyTag(field, 0, tt.tagValue)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}

				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			meta, ok := result.(*BodyMetadata)
			require.True(t, ok)

			assert.Equal(t, tt.want.MapKey, meta.MapKey)
			assert.Equal(t, tt.want.BodyType, meta.BodyType)
			assert.Equal(t, tt.want.Required, meta.Required)
		})
	}
}

func TestParseBodyType(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		want        BodyType
		wantErr     bool
		errContains string
	}{
		{
			name:    "empty string defaults to structured",
			input:   "",
			want:    BodyTypeStructured,
			wantErr: false,
		},
		{
			name:    "structured type",
			input:   "structured",
			want:    BodyTypeStructured,
			wantErr: false,
		},
		{
			name:    "file type",
			input:   "file",
			want:    BodyTypeFile,
			wantErr: false,
		},
		{
			name:    "multipart type",
			input:   "multipart",
			want:    BodyTypeMultipart,
			wantErr: false,
		},
		{
			name:        "invalid type",
			input:       "invalid",
			want:        "",
			wantErr:     true,
			errContains: "invalid body type",
		},
		{
			name:        "case sensitive - uppercase",
			input:       "STRUCTURED",
			want:        "",
			wantErr:     true,
			errContains: "invalid body type",
		},
		{
			name:        "case sensitive - mixed case",
			input:       "Structured",
			want:        "",
			wantErr:     true,
			errContains: "invalid body type",
		},
		{
			name:        "partial match",
			input:       "struct",
			want:        "",
			wantErr:     true,
			errContains: "invalid body type",
		},
		{
			name:        "json type (invalid)",
			input:       "json",
			want:        "",
			wantErr:     true,
			errContains: "must be 'structured', 'file', or 'multipart'",
		},
		{
			name:        "xml type (invalid)",
			input:       "xml",
			want:        "",
			wantErr:     true,
			errContains: "invalid body type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseBodyType(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestBodyTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		bodyType BodyType
		wantStr  string
	}{
		{
			name:     "BodyTypeStructured",
			bodyType: BodyTypeStructured,
			wantStr:  "structured",
		},
		{
			name:     "BodyTypeFile",
			bodyType: BodyTypeFile,
			wantStr:  "file",
		},
		{
			name:     "BodyTypeMultipart",
			bodyType: BodyTypeMultipart,
			wantStr:  "multipart",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantStr, string(tt.bodyType))
		})
	}
}
