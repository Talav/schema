package schema

import (
	"net/http"
	"reflect"

	"github.com/talav/mapstructure"
)

const (
	defaultSchemaTag = "schema"
	defaultBodyTag   = "body"
)

// Decoder interface for decoding HTTP request data to maps.
type Decoder interface {
	// Decode decodes HTTP parameters (query, header, cookie, path, body) to map.
	Decode(request *http.Request, routerParams map[string]string, metadata *StructMetadata) (map[string]any, error)
}

// Unmarshaler interface for unmarshaling maps to Go structs.
type Unmarshaler interface {
	// Unmarshal transforms map[string]any into a Go struct pointed to by result.
	Unmarshal(data map[string]any, result any) error
}

// Codec handles encoding and decoding between structs and parameter strings.
// It uses injectable decoder and unmarshaler for request handling.
type Codec struct {
	metadata    *Metadata // uses Metadata for metadata (assumes fully configured)
	unmarshaler Unmarshaler
	decoder     Decoder
}

// NewDefaultCodec creates a new Codec with default configuration.
// This is the recommended constructor for most use cases.
func NewDefaultCodec() *Codec {
	return NewCodec(NewDefaultMetadata(), mapstructure.NewDefaultUnmarshaler(), NewDefaultDecoder())
}

// NewCodec creates a new Codec with custom dependencies for advanced use cases.
func NewCodec(metadata *Metadata, unmarshaler Unmarshaler, decoder Decoder) *Codec {
	return &Codec{
		metadata:    metadata,
		unmarshaler: unmarshaler,
		decoder:     decoder,
	}
}

// DecodeRequest decodes an HTTP request into the provided struct.
// result must be a pointer to the target struct.
func (c *Codec) DecodeRequest(request *http.Request, routerParams map[string]string, result any) error {
	typ := reflect.TypeOf(result)
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	metadata, err := c.metadata.GetStructMetadata(typ)
	if err != nil {
		return err
	}

	// Decode parameters to map
	paramMap, err := c.decoder.Decode(request, routerParams, metadata)
	if err != nil {
		return err
	}

	// Unmarshal map to struct
	return c.unmarshaler.Unmarshal(paramMap, result)
}
