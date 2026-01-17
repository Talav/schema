package schema

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestDecoder creates a decoder for testing.
func newTestDecoder() *defaultDecoder {
	metadata := NewDefaultMetadata()

	decoder := NewDecoder(metadata, "schema", "body")
	d, ok := decoder.(*defaultDecoder)
	if !ok {
		panic("NewDecoder returned unexpected type")
	}

	return d
}

func createQueryRequest(queryString string) *http.Request {
	return httptest.NewRequest(http.MethodGet, "/test?"+queryString, nil)
}

func createParamMetadata(structVal any) *StructMetadata {
	structType := reflect.TypeOf(structVal)
	registry := NewDefaultTagParserRegistry()
	builder := newMetadataBuilder(registry)
	structMeta, err := builder.buildStructMetadata(structType)
	if err != nil {
		panic(fmt.Sprintf("failed to build struct metadata: %v", err))
	}

	return structMeta
}

func TestDecoder_Decode(t *testing.T) {
	tests := []struct {
		name         string
		queryString  string
		routerParams map[string]string
		headers      map[string]string
		cookies      map[string]string
		metadata     *StructMetadata
		want         map[string]any
		wantErr      bool
	}{
		{
			name:        "query parameters only",
			queryString: "name=john&age=30",
			metadata: createParamMetadata(struct {
				Name string `schema:"name,location=query,style=form,explode=true"`
				Age  string `schema:"age,location=query,style=form,explode=true"`
			}{}),
			want: map[string]any{
				"name": "john",
				"age":  "30",
			},
		},
		{
			name:         "path parameters only",
			routerParams: map[string]string{"id": "123", "slug": "test-post"},
			metadata: createParamMetadata(struct {
				ID   string `schema:"id,location=path"`
				Slug string `schema:"slug,location=path"`
			}{}),
			want: map[string]any{
				"id":   "123",
				"slug": "test-post",
			},
		},
		{
			name:    "header parameters only",
			headers: map[string]string{"X-Request-ID": "abc123", "X-Client-Version": "1.0"},
			metadata: createParamMetadata(struct {
				RequestID     string `schema:"X-Request-ID,location=header"`
				ClientVersion string `schema:"X-Client-Version,location=header"`
			}{}),
			want: map[string]any{
				"X-Request-ID":     "abc123",
				"X-Client-Version": "1.0",
			},
		},
		{
			name:        "empty query string",
			queryString: "",
			metadata: createParamMetadata(struct {
				Name string `schema:"name,location=query,style=form,explode=true"`
			}{}),
			want: map[string]any{},
		},
	}

	decoder := newTestDecoder()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createQueryRequest(tt.queryString)

			// Add headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			// Add cookies
			for name, value := range tt.cookies {
				req.AddCookie(&http.Cookie{Name: name, Value: value})
			}

			result, err := decoder.Decode(req, tt.routerParams, tt.metadata)

			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestDecoder_DecodeQuery(t *testing.T) {
	tests := []struct {
		name        string
		queryString string
		structVal   any
		want        map[string]any
		wantErr     bool
	}{
		{
			name:        "simple query parameters",
			queryString: "name=john&age=30",
			structVal: struct {
				Name string `schema:"name,location=query,style=form,explode=true"`
				Age  string `schema:"age,location=query,style=form,explode=true"`
			}{},
			want: map[string]any{
				"name": "john",
				"age":  "30",
			},
		},
		{
			name:        "array parameter exploded",
			queryString: "ids=1&ids=2&ids=3",
			structVal: struct {
				IDs []string `schema:"ids,location=query,style=form,explode=true"`
			}{},
			want: map[string]any{
				"ids": []any{"1", "2", "3"},
			},
		},
		{
			name:        "array parameter non-exploded",
			queryString: "ids=1,2,3",
			structVal: struct {
				IDs []string `schema:"ids,location=query,style=form,explode=false"`
			}{},
			want: map[string]any{
				"ids": []any{"1", "2", "3"},
			},
		},
		{
			name:        "deep object style",
			queryString: "filter%5Btype%5D=car&filter%5Bcolor%5D=red",
			structVal: struct {
				Filter map[string]any `schema:"filter,location=query,style=deepObject,explode=true"`
			}{},
			want: map[string]any{
				"filter": map[string]any{
					"type":  "car",
					"color": "red",
				},
			},
		},
		{
			name:        "space delimited style",
			queryString: "ids=1%202%203",
			structVal: struct {
				IDs []string `schema:"ids,location=query,style=spaceDelimited"`
			}{},
			want: map[string]any{
				"ids": []any{"1", "2", "3"},
			},
		},
		{
			name:        "pipe delimited style",
			queryString: "ids=1%7C2%7C3",
			structVal: struct {
				IDs []string `schema:"ids,location=query,style=pipeDelimited"`
			}{},
			want: map[string]any{
				"ids": []any{"1", "2", "3"},
			},
		},
		{
			name:        "no query fields in metadata",
			queryString: "name=john",
			structVal: struct {
				ID string `schema:"id,location=path"`
			}{},
			want: map[string]any{},
		},
		{
			name:        "empty query string",
			queryString: "",
			structVal: struct {
				Name string `schema:"name,location=query,style=form,explode=true"`
			}{},
			want: map[string]any{},
		},
		{
			name:        "mixed styles in query",
			queryString: "name=john&filter%5Btype%5D=car",
			structVal: struct {
				Name   string         `schema:"name,location=query,style=form,explode=true"`
				Filter map[string]any `schema:"filter,location=query,style=deepObject,explode=true"`
			}{},
			want: map[string]any{
				"name": "john",
				"filter": map[string]any{
					"type": "car",
				},
			},
		},
	}

	decoder := newTestDecoder()
	metadata := NewDefaultMetadata()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			structType := reflect.TypeOf(tt.structVal)
			structMeta, err := metadata.GetStructMetadata(structType)
			require.NoError(t, err)

			req := createQueryRequest(tt.queryString)

			result, err := decoder.decodeQuery(req, structMeta)

			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestDecoder_DecodePath(t *testing.T) {
	tests := []struct {
		name         string
		routerParams map[string]string
		structType   reflect.Type
		want         map[string]any
		wantErr      bool
	}{
		{
			name: "simple path parameter",
			routerParams: map[string]string{
				"id": "123",
			},
			structType: reflect.TypeOf(struct {
				ID string `schema:"id,location=path"`
			}{}),
			want: map[string]any{
				"id": "123",
			},
		},
		{
			name: "multiple path parameters",
			routerParams: map[string]string{
				"userID": "42",
				"postID": "100",
			},
			structType: reflect.TypeOf(struct {
				UserID string `schema:"userID,location=path"`
				PostID string `schema:"postID,location=path"`
			}{}),
			want: map[string]any{
				"userID": "42",
				"postID": "100",
			},
		},
		{
			name: "label style path parameter",
			routerParams: map[string]string{
				"values": ".1.2.3",
			},
			structType: reflect.TypeOf(struct {
				Values []string `schema:"values,location=path,style=label,explode=true"`
			}{}),
			want: map[string]any{
				"values": []any{"1", "2", "3"},
			},
		},
		{
			name:         "empty router params",
			routerParams: map[string]string{},
			structType: reflect.TypeOf(struct {
				ID string `schema:"id,location=path"`
			}{}),
			want: map[string]any{
				"id": "",
			},
		},
		{
			name: "no path fields in metadata",
			routerParams: map[string]string{
				"id": "123",
			},
			structType: reflect.TypeOf(struct {
				Name string `schema:"name,location=query"`
			}{}),
			want: map[string]any{},
		},
	}

	decoder := newTestDecoder()
	metadata := NewDefaultMetadata()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			structMeta, err := metadata.GetStructMetadata(tt.structType)
			require.NoError(t, err)

			result, err := decoder.decodePath(tt.routerParams, structMeta)

			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestDecoder_DecodeHeader(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		structType reflect.Type
		want       map[string]any
		wantErr    bool
	}{
		{
			name: "simple header parameter",
			headers: map[string]string{
				"X-Request-ID": "abc123",
			},
			structType: reflect.TypeOf(struct {
				RequestID string `schema:"X-Request-ID,location=header"`
			}{}),
			want: map[string]any{
				"X-Request-ID": "abc123",
			},
		},
		{
			name: "multiple header parameters",
			headers: map[string]string{
				"X-Request-ID":     "abc123",
				"X-Client-Version": "2.0",
			},
			structType: reflect.TypeOf(struct {
				RequestID     string `schema:"X-Request-ID,location=header"`
				ClientVersion string `schema:"X-Client-Version,location=header"`
			}{}),
			want: map[string]any{
				"X-Request-ID":     "abc123",
				"X-Client-Version": "2.0",
			},
		},
		{
			name:    "missing header",
			headers: map[string]string{},
			structType: reflect.TypeOf(struct {
				RequestID string `schema:"X-Request-ID,location=header"`
			}{}),
			want: map[string]any{
				"X-Request-ID": "",
			},
		},
		{
			name: "no header fields in metadata",
			headers: map[string]string{
				"X-Request-ID": "abc123",
			},
			structType: reflect.TypeOf(struct {
				Name string `schema:"name,location=query"`
			}{}),
			want: map[string]any{},
		},
	}

	decoder := newTestDecoder()
	metadata := NewDefaultMetadata()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			structMeta, err := metadata.GetStructMetadata(tt.structType)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			result, err := decoder.decodeHeader(req, structMeta)

			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestDecoder_DecodeCookie(t *testing.T) {
	// NOTE: The current decoder implementation has a limitation where StyleForm
	// (the only allowed style for cookies per OpenAPI spec) is not supported
	// in decodeValueByStyle for single values. These tests verify behavior
	// for scenarios that work around this limitation.
	tests := []struct {
		name       string
		cookies    map[string]string
		structType reflect.Type
		want       map[string]any
		wantErr    bool
	}{
		{
			name:    "missing cookie (skipped)",
			cookies: map[string]string{},
			structType: reflect.TypeOf(struct {
				Session string `schema:"session,location=cookie"`
			}{}),
			want: map[string]any{},
		},
		{
			name: "no cookie fields in metadata",
			cookies: map[string]string{
				"session": "xyz789",
			},
			structType: reflect.TypeOf(struct {
				Name string `schema:"name,location=query"`
			}{}),
			want: map[string]any{},
		},
	}

	decoder := newTestDecoder()
	metadata := NewDefaultMetadata()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			structMeta, err := metadata.GetStructMetadata(tt.structType)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			for name, value := range tt.cookies {
				req.AddCookie(&http.Cookie{Name: name, Value: value})
			}

			result, err := decoder.decodeCookie(req, structMeta)

			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestFilterQueryValuesByFields(t *testing.T) {
	tests := []struct {
		name      string
		allValues map[string][]string
		fields    []FieldMetadata
		want      map[string][]string
	}{
		{
			name: "filters matching fields",
			allValues: map[string][]string{
				"name":    {"john"},
				"age":     {"30"},
				"unknown": {"value"},
			},
			fields: []FieldMetadata{
				{
					TagMetadata: map[string]any{
						"schema": &SchemaMetadata{ParamName: "name", MapKey: "Name"},
					},
				},
				{
					TagMetadata: map[string]any{
						"schema": &SchemaMetadata{ParamName: "age", MapKey: "Age"},
					},
				},
			},
			want: map[string][]string{
				"name": {"john"},
				"age":  {"30"},
			},
		},
		{
			name: "handles deep object notation",
			allValues: map[string][]string{
				"filter[type]":  {"car"},
				"filter[color]": {"red"},
				"other":         {"value"},
			},
			fields: []FieldMetadata{
				{
					TagMetadata: map[string]any{
						"schema": &SchemaMetadata{ParamName: "filter", MapKey: "Filter"},
					},
				},
			},
			want: map[string][]string{
				"filter[type]":  {"car"},
				"filter[color]": {"red"},
			},
		},
		{
			name: "handles dotted notation",
			allValues: map[string][]string{
				"user.name": {"john"},
				"user.age":  {"30"},
				"other":     {"value"},
			},
			fields: []FieldMetadata{
				{
					TagMetadata: map[string]any{
						"schema": &SchemaMetadata{ParamName: "user", MapKey: "User"},
					},
				},
			},
			want: map[string][]string{
				"user.name": {"john"},
				"user.age":  {"30"},
			},
		},
		{
			name: "no matching fields",
			allValues: map[string][]string{
				"unknown": {"value"},
			},
			fields: []FieldMetadata{
				{
					TagMetadata: map[string]any{
						"schema": &SchemaMetadata{ParamName: "name", MapKey: "Name"},
					},
				},
			},
			want: map[string][]string{},
		},
		{
			name:      "empty values",
			allValues: map[string][]string{},
			fields: []FieldMetadata{
				{
					TagMetadata: map[string]any{
						"schema": &SchemaMetadata{ParamName: "name", MapKey: "Name"},
					},
				},
			},
			want: map[string][]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterQueryValuesByFields(tt.allValues, tt.fields)

			assert.Equal(t, tt.want, map[string][]string(result))
		})
	}
}
