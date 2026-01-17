package schema

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:maintidx // Comprehensive table-driven test - acceptable complexity
func TestParseSchemaTag(t *testing.T) {
	tests := []struct {
		name        string
		fieldName   string
		tagValue    string
		want        *SchemaMetadata
		wantErr     bool
		errContains string
	}{
		{
			name:      "simple tag with name only",
			fieldName: "Name",
			tagValue:  "name",
			want: &SchemaMetadata{
				ParamName: "name",
				MapKey:    "Name",
				Location:  LocationQuery,
				Style:     StyleForm,
				Explode:   true,
				Required:  false,
			},
		},
		{
			name:      "empty tag name uses field name",
			fieldName: "Email",
			tagValue:  "",
			want: &SchemaMetadata{
				ParamName: "Email",
				MapKey:    "Email",
				Location:  LocationQuery,
				Style:     StyleForm,
				Explode:   true,
				Required:  false,
			},
		},
		{
			name:      "tag with location query",
			fieldName: "Filter",
			tagValue:  "filter,location=query",
			want: &SchemaMetadata{
				ParamName: "filter",
				MapKey:    "Filter",
				Location:  LocationQuery,
				Style:     StyleForm,
				Explode:   true,
				Required:  false,
			},
		},
		{
			name:      "tag with location path",
			fieldName: "ID",
			tagValue:  "id,location=path",
			want: &SchemaMetadata{
				ParamName: "id",
				MapKey:    "ID",
				Location:  LocationPath,
				Style:     StyleSimple,
				Explode:   false,
				Required:  true, // Path is always required
			},
		},
		{
			name:      "tag with location header",
			fieldName: "APIKey",
			tagValue:  "X-API-Key,location=header",
			want: &SchemaMetadata{
				ParamName: "X-API-Key",
				MapKey:    "APIKey",
				Location:  LocationHeader,
				Style:     StyleSimple,
				Explode:   false,
				Required:  false,
			},
		},
		{
			name:      "tag with location cookie",
			fieldName: "SessionID",
			tagValue:  "session_id,location=cookie",
			want: &SchemaMetadata{
				ParamName: "session_id",
				MapKey:    "SessionID",
				Location:  LocationCookie,
				Style:     StyleForm,
				Explode:   true,
				Required:  false,
			},
		},
		{
			name:      "tag with explicit style",
			fieldName: "Tags",
			tagValue:  "tags,location=query,style=spaceDelimited",
			want: &SchemaMetadata{
				ParamName: "tags",
				MapKey:    "Tags",
				Location:  LocationQuery,
				Style:     StyleSpaceDelimited,
				Explode:   false,
				Required:  false,
			},
		},
		{
			name:      "tag with explode true",
			fieldName: "Items",
			tagValue:  "items,location=query,explode=true",
			want: &SchemaMetadata{
				ParamName: "items",
				MapKey:    "Items",
				Location:  LocationQuery,
				Style:     StyleForm,
				Explode:   true,
				Required:  false,
			},
		},
		{
			name:      "tag with explode false",
			fieldName: "Items",
			tagValue:  "items,location=query,explode=false",
			want: &SchemaMetadata{
				ParamName: "items",
				MapKey:    "Items",
				Location:  LocationQuery,
				Style:     StyleForm,
				Explode:   false,
				Required:  false,
			},
		},
		{
			name:      "tag with explode flag form",
			fieldName: "Items",
			tagValue:  "items,location=query,explode",
			want: &SchemaMetadata{
				ParamName: "items",
				MapKey:    "Items",
				Location:  LocationQuery,
				Style:     StyleForm,
				Explode:   true,
				Required:  false,
			},
		},
		{
			name:      "tag with required true",
			fieldName: "Name",
			tagValue:  "name,required=true",
			want: &SchemaMetadata{
				ParamName: "name",
				MapKey:    "Name",
				Location:  LocationQuery,
				Style:     StyleForm,
				Explode:   true,
				Required:  true,
			},
		},
		{
			name:      "tag with required flag form",
			fieldName: "Name",
			tagValue:  "name,required",
			want: &SchemaMetadata{
				ParamName: "name",
				MapKey:    "Name",
				Location:  LocationQuery,
				Style:     StyleForm,
				Explode:   true,
				Required:  true,
			},
		},
		{
			name:      "path location always required even if not specified",
			fieldName: "ID",
			tagValue:  "id,location=path,required=false",
			want: &SchemaMetadata{
				ParamName: "id",
				MapKey:    "ID",
				Location:  LocationPath,
				Style:     StyleSimple,
				Explode:   false,
				Required:  true, // Path is always required, overrides explicit false
			},
		},
		{
			name:      "query with deepObject style",
			fieldName: "Filter",
			tagValue:  "filter,location=query,style=deepObject",
			want: &SchemaMetadata{
				ParamName: "filter",
				MapKey:    "Filter",
				Location:  LocationQuery,
				Style:     StyleDeepObject,
				Explode:   true, // deepObject defaults to explode=true
				Required:  false,
			},
		},
		{
			name:      "path with label style",
			fieldName: "ID",
			tagValue:  "id,location=path,style=label",
			want: &SchemaMetadata{
				ParamName: "id",
				MapKey:    "ID",
				Location:  LocationPath,
				Style:     StyleLabel,
				Explode:   false,
				Required:  true,
			},
		},
		{
			name:      "path with matrix style",
			fieldName: "ID",
			tagValue:  "id,location=path,style=matrix",
			want: &SchemaMetadata{
				ParamName: "id",
				MapKey:    "ID",
				Location:  LocationPath,
				Style:     StyleMatrix,
				Explode:   false,
				Required:  true,
			},
		},
		{
			name:      "query with pipeDelimited style",
			fieldName: "Tags",
			tagValue:  "tags,location=query,style=pipeDelimited",
			want: &SchemaMetadata{
				ParamName: "tags",
				MapKey:    "Tags",
				Location:  LocationQuery,
				Style:     StylePipeDelimited,
				Explode:   false,
				Required:  false,
			},
		},
		{
			name:      "all options combined",
			fieldName: "Filter",
			tagValue:  "filter,location=query,style=form,explode=true,required=true",
			want: &SchemaMetadata{
				ParamName: "filter",
				MapKey:    "Filter",
				Location:  LocationQuery,
				Style:     StyleForm,
				Explode:   true,
				Required:  true,
			},
		},
		{
			name:        "invalid location",
			fieldName:   "Name",
			tagValue:    "name,location=invalid",
			wantErr:     true,
			errContains: "invalid location",
		},
		{
			name:        "invalid style for query location",
			fieldName:   "Name",
			tagValue:    "name,location=query,style=simple",
			wantErr:     true,
			errContains: "invalid style",
		},
		{
			name:        "invalid style for path location",
			fieldName:   "ID",
			tagValue:    "id,location=path,style=form",
			wantErr:     true,
			errContains: "invalid style",
		},
		{
			name:        "invalid style for header location",
			fieldName:   "Key",
			tagValue:    "key,location=header,style=form",
			wantErr:     true,
			errContains: "invalid style",
		},
		{
			name:        "invalid style for cookie location",
			fieldName:   "Session",
			tagValue:    "session,location=cookie,style=simple",
			wantErr:     true,
			errContains: "invalid style",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := reflect.StructField{
				Name: tt.fieldName,
				Type: reflect.TypeFor[string](),
			}

			result, err := ParseSchemaTag(field, 0, tt.tagValue)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}

				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			meta, ok := result.(*SchemaMetadata)
			require.True(t, ok, "result should be *SchemaMetadata")

			assert.Equal(t, tt.want.ParamName, meta.ParamName)
			assert.Equal(t, tt.want.MapKey, meta.MapKey)
			assert.Equal(t, tt.want.Location, meta.Location)
			assert.Equal(t, tt.want.Style, meta.Style)
			assert.Equal(t, tt.want.Explode, meta.Explode)
			assert.Equal(t, tt.want.Required, meta.Required)
		})
	}
}

func TestDefaultSchemaMetadata(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		fieldType reflect.Type
		want      *SchemaMetadata
	}{
		{
			name:      "string field",
			fieldName: "Name",
			fieldType: reflect.TypeFor[string](),
			want: &SchemaMetadata{
				ParamName: "Name",
				MapKey:    "Name",
				Location:  LocationQuery,
				Style:     StyleForm,
				Explode:   true,
				Required:  false,
			},
		},
		{
			name:      "int field",
			fieldName: "Age",
			fieldType: reflect.TypeFor[int](),
			want: &SchemaMetadata{
				ParamName: "Age",
				MapKey:    "Age",
				Location:  LocationQuery,
				Style:     StyleForm,
				Explode:   true,
				Required:  false,
			},
		},
		{
			name:      "slice field",
			fieldName: "Tags",
			fieldType: reflect.TypeFor[[]string](),
			want: &SchemaMetadata{
				ParamName: "Tags",
				MapKey:    "Tags",
				Location:  LocationQuery,
				Style:     StyleForm,
				Explode:   true,
				Required:  false,
			},
		},
		{
			name:      "struct field",
			fieldName: "User",
			fieldType: reflect.TypeFor[struct{}](),
			want: &SchemaMetadata{
				ParamName: "User",
				MapKey:    "User",
				Location:  LocationQuery,
				Style:     StyleForm,
				Explode:   true,
				Required:  false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := reflect.StructField{
				Name: tt.fieldName,
				Type: tt.fieldType,
			}

			result := DefaultSchemaMetadata(field, 0)

			require.NotNil(t, result)

			meta, ok := result.(*SchemaMetadata)
			require.True(t, ok, "result should be *SchemaMetadata")

			assert.Equal(t, tt.want.ParamName, meta.ParamName)
			assert.Equal(t, tt.want.MapKey, meta.MapKey)
			assert.Equal(t, tt.want.Location, meta.Location)
			assert.Equal(t, tt.want.Style, meta.Style)
			assert.Equal(t, tt.want.Explode, meta.Explode)
			assert.Equal(t, tt.want.Required, meta.Required)
		})
	}
}

func TestParseSchemaTag_DefaultValues(t *testing.T) {
	tests := []struct {
		name             string
		tagValue         string
		expectedLocation ParameterLocation
		expectedStyle    Style
		expectedExplode  bool
	}{
		{
			name:             "no location defaults to query",
			tagValue:         "name",
			expectedLocation: LocationQuery,
			expectedStyle:    StyleForm,
			expectedExplode:  true,
		},
		{
			name:             "query location defaults to form style",
			tagValue:         "name,location=query",
			expectedLocation: LocationQuery,
			expectedStyle:    StyleForm,
			expectedExplode:  true,
		},
		{
			name:             "path location defaults to simple style",
			tagValue:         "id,location=path",
			expectedLocation: LocationPath,
			expectedStyle:    StyleSimple,
			expectedExplode:  false,
		},
		{
			name:             "header location defaults to simple style",
			tagValue:         "key,location=header",
			expectedLocation: LocationHeader,
			expectedStyle:    StyleSimple,
			expectedExplode:  false,
		},
		{
			name:             "cookie location defaults to form style",
			tagValue:         "session,location=cookie",
			expectedLocation: LocationCookie,
			expectedStyle:    StyleForm,
			expectedExplode:  true,
		},
		{
			name:             "form style defaults to explode true",
			tagValue:         "items,location=query,style=form",
			expectedLocation: LocationQuery,
			expectedStyle:    StyleForm,
			expectedExplode:  true,
		},
		{
			name:             "deepObject style defaults to explode true",
			tagValue:         "filter,location=query,style=deepObject",
			expectedLocation: LocationQuery,
			expectedStyle:    StyleDeepObject,
			expectedExplode:  true,
		},
		{
			name:             "spaceDelimited style defaults to explode false",
			tagValue:         "tags,location=query,style=spaceDelimited",
			expectedLocation: LocationQuery,
			expectedStyle:    StyleSpaceDelimited,
			expectedExplode:  false,
		},
		{
			name:             "pipeDelimited style defaults to explode false",
			tagValue:         "tags,location=query,style=pipeDelimited",
			expectedLocation: LocationQuery,
			expectedStyle:    StylePipeDelimited,
			expectedExplode:  false,
		},
		{
			name:             "simple style defaults to explode false",
			tagValue:         "id,location=path,style=simple",
			expectedLocation: LocationPath,
			expectedStyle:    StyleSimple,
			expectedExplode:  false,
		},
		{
			name:             "label style defaults to explode false",
			tagValue:         "id,location=path,style=label",
			expectedLocation: LocationPath,
			expectedStyle:    StyleLabel,
			expectedExplode:  false,
		},
		{
			name:             "matrix style defaults to explode false",
			tagValue:         "id,location=path,style=matrix",
			expectedLocation: LocationPath,
			expectedStyle:    StyleMatrix,
			expectedExplode:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := reflect.StructField{
				Name: "TestField",
				Type: reflect.TypeFor[string](),
			}

			result, err := ParseSchemaTag(field, 0, tt.tagValue)
			require.NoError(t, err)

			meta, ok := result.(*SchemaMetadata)
			require.True(t, ok)

			assert.Equal(t, tt.expectedLocation, meta.Location)
			assert.Equal(t, tt.expectedStyle, meta.Style)
			assert.Equal(t, tt.expectedExplode, meta.Explode)
		})
	}
}

func TestParseSchemaTag_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		fieldName   string
		tagValue    string
		want        *SchemaMetadata
		wantErr     bool
		errContains string
	}{
		{
			name:      "tag with comma in name",
			fieldName: "Name",
			tagValue:  "name,value",
			want: &SchemaMetadata{
				ParamName: "name",
				MapKey:    "Name",
				Location:  LocationQuery,
				Style:     StyleForm,
				Explode:   true,
				Required:  false,
			},
		},
		{
			name:      "tag with multiple options",
			fieldName: "Filter",
			tagValue:  "filter,location=query,style=form,explode=true,required=true",
			want: &SchemaMetadata{
				ParamName: "filter",
				MapKey:    "Filter",
				Location:  LocationQuery,
				Style:     StyleForm,
				Explode:   true,
				Required:  true,
			},
		},
		{
			name:      "tag with empty value options",
			fieldName: "Name",
			tagValue:  "name,required,explode",
			want: &SchemaMetadata{
				ParamName: "name",
				MapKey:    "Name",
				Location:  LocationQuery,
				Style:     StyleForm,
				Explode:   true,
				Required:  true,
			},
		},
		{
			name:      "tag with false values",
			fieldName: "Name",
			tagValue:  "name,required=false,explode=false",
			want: &SchemaMetadata{
				ParamName: "name",
				MapKey:    "Name",
				Location:  LocationQuery,
				Style:     StyleForm,
				Explode:   false,
				Required:  false,
			},
		},
		{
			name:      "long field name",
			fieldName: "VeryLongFieldNameForTesting",
			tagValue:  "short",
			want: &SchemaMetadata{
				ParamName: "short",
				MapKey:    "VeryLongFieldNameForTesting",
				Location:  LocationQuery,
				Style:     StyleForm,
				Explode:   true,
				Required:  false,
			},
		},
		{
			name:      "special characters in param name",
			fieldName: "APIKey",
			tagValue:  "X-API-Key,location=header",
			want: &SchemaMetadata{
				ParamName: "X-API-Key",
				MapKey:    "APIKey",
				Location:  LocationHeader,
				Style:     StyleSimple,
				Explode:   false,
				Required:  false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := reflect.StructField{
				Name: tt.fieldName,
				Type: reflect.TypeFor[string](),
			}

			result, err := ParseSchemaTag(field, 0, tt.tagValue)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}

				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			meta, ok := result.(*SchemaMetadata)
			require.True(t, ok)

			assert.Equal(t, tt.want.ParamName, meta.ParamName)
			assert.Equal(t, tt.want.MapKey, meta.MapKey)
			assert.Equal(t, tt.want.Location, meta.Location)
			assert.Equal(t, tt.want.Style, meta.Style)
			assert.Equal(t, tt.want.Explode, meta.Explode)
			assert.Equal(t, tt.want.Required, meta.Required)
		})
	}
}
