# schema

[![Go Reference](https://pkg.go.dev/badge/github.com/talav/schema.svg)](https://pkg.go.dev/github.com/talav/schema)
[![Go Report Card](https://goreportcard.com/badge/github.com/talav/schema)](https://goreportcard.com/report/github.com/talav/schema)
[![CI](https://github.com/talav/schema/actions/workflows/schema-ci.yml/badge.svg)](https://github.com/talav/schema/actions)
[![codecov](https://codecov.io/gh/talav/schema/graph/badge.svg)](https://codecov.io/gh/talav/schema)

A Go package for decoding HTTP requests into Go structs with full support for OpenAPI 3.0 parameter serialization. Handles query, path, header, cookie parameters and request bodies (JSON, XML, forms, multipart, files) in a single unified API. 

Decouples business logic from HTTP request handling. Request types are defined with struct tags. The package extracts query parameters, path variables, headers, cookies, and request bodies, converting them into typed Go structs with OpenAPI 3.0 compliance.


## Architecture

The package consists of four main components working together:

1. **Codec** - High-level API that orchestrates the decoding pipeline
2. **Metadata** - Parses and caches struct tag metadata for performance
3. **Decoder** - Extracts parameters from HTTP requests into maps
4. **Unmarshaler** - Converts maps to typed structs (default: mapstructure)

**Thread Safety:** All components are safe for concurrent use. Create a codec once at startup and reuse it across all requests for optimal performance.

## Features

- Full OpenAPI 3.0 parameter serialization support
- All parameter locations: query, path, header, cookie
- All serialization styles: form, simple, matrix, label, spaceDelimited, pipeDelimited, deepObject
- Explode parameter support
- Request body decoding: JSON, XML, URL-encoded forms, multipart forms, file uploads
- Struct tag-based configuration
- Metadata caching for performance
- Extensible architecture (custom decoders/unmarshalers)

## Installation

```bash
go get github.com/talav/schema
```

## Quick Start

```go
package main

import (
    "net/http"
    "github.com/talav/schema"
)

type CreateUserRequest struct {
    // Query parameter
    Version string `schema:"version,location=query"`
    // Path parameter (from router)
    OrgID string `schema:"org_id,location=path"`
    // Header parameter
    APIKey string `schema:"X-Api-Key,location=header"`
    // Request body
    Body struct {
        Name  string `schema:"name"`
        Email string `schema:"email"`
    } `body:"structured"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    codec := schema.NewDefaultCodec()
    
    // Router params come from your router (chi, gorilla, etc.)
    routerParams := map[string]string{"org_id": "123"}
    
    var req CreateUserRequest
    if err := codec.DecodeRequest(r, routerParams, &req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Use req.Version, req.OrgID, req.APIKey, req.Body.Name, req.Body.Email
}
```

## Basic Usage

### Simple Decoding

The simplest way to decode an HTTP request:

```go
type UserRequest struct {
    Name string `schema:"name"`
    Age  int    `schema:"age"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    codec := schema.NewDefaultCodec()

    var req UserRequest
    if err := codec.DecodeRequest(r, nil, &req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // req.Name and req.Age are now populated from query parameters
    fmt.Printf("User: %s, Age: %d\n", req.Name, req.Age)
}
```

### Type Conversion

The underlying mapstructure library handles automatic type conversion:

```go
type Config struct {
    Host     string  `schema:"host"`
    Port     int     `schema:"port"`
    Enabled  bool    `schema:"enabled"`
    Timeout  float64 `schema:"timeout"`
}

// All of these work automatically:
// GET /config?host=localhost&port=8080&enabled=true&timeout=30.5
data := map[string]any{
    "host":    "localhost",  // string → string
    "port":    "8080",       // string → int
    "enabled": "true",       // string → bool
    "timeout": 30,           // int → float64
}

var config Config
codec.DecodeRequest(r, nil, &config)
```

**Supported type conversions:**

| Target Type | Accepted Input Types | Example |
|-------------|---------------------|---------|
| `string` | string, bool, int, uint, float, []byte | `42` → `"42"` |
| `bool` | bool, int, uint, float, string | `"true"`, `1` → `true` |
| `int`, `int8`...`int64` | int, uint, float, bool, string | `"42"` → `42` |
| `uint`, `uint8`...`uint64` | int, uint, float, bool, string | `"42"` → `uint(42)` |
| `float32`, `float64` | int, uint, float, bool, string | `"3.14"` → `3.14` |

### Default Values

Use the `default` tag to set default values for missing fields:

```go
type APIConfig struct {
    Host    string `schema:"host" default:"localhost"`
    Port    int    `schema:"port" default:"8080"`
    Debug   bool   `schema:"debug" default:"false"`
    Timeout int    `schema:"timeout" default:"30"`
}

// Only host is provided: GET /config?host=api.example.com
// Result: Host="api.example.com", Port=8080, Debug=false, Timeout=30
```

## Struct Tags

### Parameter Tag (`schema`)

The `schema` tag configures how fields are extracted from HTTP request parameters.

```go
type Request struct {
    // Basic: field name used as parameter name, defaults to query location
    Name string `schema:"name"`
    
    // With location
    ID string `schema:"id,location=path"`
    
    // With style and explode
    IDs []string `schema:"ids,location=query,style=form,explode=true"`
    
    // Required field
    Token string `schema:"token,location=header,required"`
    
    // Full specification
    Filter map[string]string `schema:"filter,location=query,style=deepObject,explode=true,required=true"`
}
```

**Tag Options:**

| Option | Values | Default | Description |
|--------|--------|---------|-------------|
| `location` | `query`, `path`, `header`, `cookie` | `query` | Parameter location |
| `style` | See styles table | Location default | Serialization style |
| `explode` | `true`, `false` | Style default | Explode arrays/objects |
| `required` | `true`, `false` | `false` (`true` for path) | Mark as required |

**Skip a field:**

```go
type Request struct {
    Internal string `schema:"-"` // Skipped during decoding
}
```

### Body Tag (`body`)

The `body` tag configures how the request body is decoded.

```go
type Request struct {
    // JSON or XML body (auto-detected from Content-Type)
    Body UserData `body:"structured"`
    
    // File upload (raw bytes)
    File []byte `body:"file"`
    
    // Multipart form
    Form FormData `body:"multipart"`
}
```

## Request Body Types

The schema package supports three main body types, each designed for different use cases. Body type detection is automatic based on both the `body` tag and the `Content-Type` header.

### Structured Data (`body:"structured"`)

**Purpose:** Decode structured data (JSON, XML, forms) into Go structs
**Content-Types:** `application/json`, `application/xml`, `text/xml`, `application/x-www-form-urlencoded`
**Use Case:** REST APIs, form submissions, data exchange

**Automatic Content-Type Detection:**
- `application/json` → JSON unmarshaling
- `application/xml`, `text/xml` → XML unmarshaling with full struct tag support
- `application/x-www-form-urlencoded` → Form data parsing
- *Fallback:* JSON (if content-type is unrecognized)

**Examples:**

```go
// JSON API endpoint
type CreateUserRequest struct {
    Body struct {
        Name  string `schema:"name"`
        Email string `schema:"email"`
        Age   int    `schema:"age"`
    } `body:"structured"`
}

// POST /users
// Content-Type: application/json
// {"name": "Alice", "email": "alice@example.com", "age": 30}
```

```go
// XML API endpoint
type XMLImportRequest struct {
    Body struct {
        Name  string `xml:"name"`
        Email string `xml:"email"`
    } `body:"structured"`
}

// POST /import
// Content-Type: application/xml
// <user><name>Alice</name><email>alice@example.com</email></user>
// Result: Body.Name == "Alice", Body.Email == "alice@example.com"
```

```go
// Form submission
type LoginRequest struct {
    Body struct {
        Username string `schema:"username"`
        Password string `schema:"password"`
        Remember bool   `schema:"remember"`
    } `body:"structured"`
}

// POST /login
// Content-Type: application/x-www-form-urlencoded
// username=alice&password=secret&remember=true
```

### Raw Files (`body:"file"`)

**Purpose:** Handle raw binary data uploads
**Content-Types:** `application/octet-stream`, any binary content
**Use Case:** File uploads, binary data processing, streaming content

**Two Consumption Patterns:**

```go
// Option 1: Load into memory (small files)
type SmallFileUpload struct {
    Filename string `schema:"filename,query"`  // Metadata in query
    Body     []byte `body:"file"`              // File content as bytes
}

// Option 2: Stream processing (large files)
type LargeFileUpload struct {
    Filename string        `schema:"filename,query"`
    Body     io.ReadCloser `body:"file"`  // Streaming reader
}

func handler(w http.ResponseWriter, r *http.Request) {
    var upload LargeFileUpload
    codec.DecodeRequest(r, nil, &upload)
    defer upload.Body.Close() // Always close!

    // Stream to disk/network
    file, _ := os.Create("/tmp/" + upload.Filename)
    io.Copy(file, upload.Body)
    file.Close()
}
```

**Memory Considerations:**
- `[]byte` loads entire file into memory - use only for small files (< 10MB)
- `io.ReadCloser` streams data - suitable for any size

**Content-Type Detection:**
Any content-type works with `body:"file"` - the package ignores content-type for file uploads and treats the body as raw bytes.

### Multipart Forms (`body:"multipart"`)

**Purpose:** Handle forms with both text fields and file uploads
**Content-Types:** `multipart/form-data` (required)
**Use Case:** HTML forms with file uploads, complex form submissions

**Structure:**
Multipart forms require a struct where each field corresponds to a form field. File fields use `io.ReadCloser`, text fields use regular types.

```go
type DocumentUploadRequest struct {
    Body struct {
        // Text fields
        Title       string `schema:"title"`
        Description string `schema:"description"`
        Category    string `schema:"category"`

        // File fields
        Document    io.ReadCloser `schema:"document"`    // Single file
        Attachments []io.ReadCloser `schema:"attachments"` // Multiple files

        // Optional: File metadata (if needed)
        DocumentName string `schema:"document_name"`
    } `body:"multipart"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    var req DocumentUploadRequest
    codec.DecodeRequest(r, nil, &req)

    // Always close file readers
    defer func() {
        if req.Body.Document != nil {
            req.Body.Document.Close()
        }
        for _, f := range req.Body.Attachments {
            f.Close()
        }
    }()

    // Process form data...
}
```

**Form Field Types:**
- **Text fields:** `string`, `int`, `bool`, etc. (automatically converted)
- **Single files:** `io.ReadCloser`
- **Multiple files:** `[]io.ReadCloser`
- **Optional files:** Use pointers `[]*io.ReadCloser` (nil if not uploaded)

### Body Type Selection Guide

| Use Case | Body Type | Example |
|----------|-----------|---------|
| **REST API with JSON** | `structured` | User creation/update APIs |
| **XML web service** | `structured` | Legacy system integration |
| **HTML form submission** | `structured` | Login, contact forms |
| **Single file upload** | `file` | Avatar, document upload |
| **Large file streaming** | `file` + `io.ReadCloser` | Video, backup uploads |
| **Form + file upload** | `multipart` | Document submission with metadata |
| **Multiple file upload** | `multipart` | Photo gallery, batch uploads |

### Advanced Usage

**Combining with query parameters:**
```go
type AdvancedRequest struct {
    // Query parameters
    APIKey  string `schema:"key,header"`
    Version string `schema:"v,query"`

    // Body data
    Body struct {
        Name  string `schema:"name"`
        Email string `schema:"email"`
    } `body:"structured"`
}
// GET /users?v=2
// Header: X-API-Key: abc123
// Body: {"name": "Alice", "email": "alice@example.com"}
```

**Nested body structs:**
```go
type NestedRequest struct {
    Body struct {
        User struct {
            Profile struct {
                Name  string `schema:"name"`
                Age   int    `schema:"age"`
            } `schema:"profile"`
            Settings map[string]bool `schema:"settings"`
        } `schema:"user"`
    } `body:"structured"`
}
// Body: {"user": {"profile": {"name": "Alice", "age": 30}, "settings": {"notifications": true}}}
```

### Nested Structs

Nested structs are handled automatically:

```go
type Address struct {
    City    string `schema:"city"`
    Country string `schema:"country"`
}

type Person struct {
    Name    string  `schema:"name"`
    Address Address `schema:"address"`
}

// Handles: POST /person
// Body: {"name": "Alice", "address": {"city": "NYC", "country": "USA"}}
```

### Embedded Structs

Embedded structs support both promoted and named field access:

```go
type Timestamps struct {
    CreatedAt string `schema:"created_at"`
    UpdatedAt string `schema:"updated_at"`
}

type User struct {
    Timestamps        // Embedded - fields promoted to parent
    Name       string `schema:"name"`
}

// Option 1: Promoted fields (flat structure)
// POST /user
// Body: {"name": "Alice", "created_at": "2024-01-01", "updated_at": "2024-01-02"}

// Option 2: Named embedded access (nested)
// Body: {"name": "Alice", "Timestamps": {"created_at": "2024-01-01", "updated_at": "2024-01-02"}}

// Both work!
```

### Pointers and Slices

```go
type Config struct {
    Tags    []string `schema:"tags"`     // Query: ?tags=go&tags=api
    Count   *int     `schema:"count"`    // nil if missing, *42 if present
    Data    []byte   `schema:"data"`     // Converts from string or []any
}

// Query: ?tags=go&tags=api&count=42&data=SGVsbG8=  (base64)
```

**⚠️ Detecting missing fields:**

By default, missing fields get zero values. Use pointers to distinguish missing from zero:

```go
type Request struct {
    Port    *int  `schema:"port"`    // nil if missing
    Enabled *bool `schema:"enabled"` // nil if missing
}

// If port not provided: req.Port == nil
// If port=0 provided: req.Port != nil && *req.Port == 0
```

## Parameter Locations

The schema package implements parameter serialization according to the [OpenAPI 3.1 Specification](https://spec.openapis.org/oas/v3.1.0#parameter-serialization). This ensures consistent behavior with OpenAPI-compliant APIs and frameworks, supporting all standard parameter locations with their respective serialization styles and explode behaviors.

| Location | Description | Default Style |
|----------|-------------|---------------|
| `query` | Query string parameters (`?name=value`) | `form` |
| `path` | Path parameters from router (`/users/{id}`) | `simple` |
| `header` | HTTP headers (`X-Api-Key: value`) | `simple` |
| `cookie` | Cookies (`Cookie: session=abc`) | `form` |

## Serialization Styles

### Query Parameters

| Style | Example | Description |
|-------|---------|-------------|
| `form` (default) | `?ids=1&ids=2` (explode) or `?ids=1,2` | Standard form encoding |
| `spaceDelimited` | `?ids=1%202%203` | Space-separated values |
| `pipeDelimited` | `?ids=1\|2\|3` | Pipe-separated values |
| `deepObject` | `?filter[type]=car&filter[color]=red` | Nested object notation |

### Path Parameters

| Style | Example | Description |
|-------|---------|-------------|
| `simple` (default) | `1,2,3` | Comma-separated |
| `label` | `.1.2.3` | Period-prefixed |
| `matrix` | `;ids=1;ids=2` | Semicolon-prefixed key-value |

### Header & Cookie Parameters

| Location | Style | Description |
|----------|-------|-------------|
| `header` | `simple` only | Comma-separated values |
| `cookie` | `form` only | Standard cookie format |

## Explode Parameter

Controls how arrays and objects are serialized:

**`explode=true` (default for form/deepObject):**
```
Arrays:  ?ids=1&ids=2&ids=3
Objects: ?filter[type]=car&filter[color]=red (deepObject)
         ?type=car&color=red (form)
```

**`explode=false`:**
```
Arrays:  ?ids=1,2,3
Objects: ?filter=type,car,color,red
```

## Usage Examples

### Query Parameters with Different Styles

```go
type SearchRequest struct {
    // Form style (default) - ?tags=go&tags=api or ?tags=go,api
    Tags []string `schema:"tags,location=query,style=form"`
    
    // Space delimited - ?ids=1%202%203
    IDs []int `schema:"ids,location=query,style=spaceDelimited"`
    
    // Pipe delimited - ?colors=red|green|blue
    Colors []string `schema:"colors,location=query,style=pipeDelimited"`
    
    // Deep object - ?filter[status]=active&filter[type]=user
    Filter struct {
        Status string `schema:"status"`
        Type   string `schema:"type"`
    } `schema:"filter,location=query,style=deepObject"`
}
```

### Path Parameters

```go
type GetUserRequest struct {
    // Simple style (default) - /users/123
    UserID string `schema:"user_id,location=path"`
    
    // Label style - /resources/.1.2.3
    Values []string `schema:"values,location=path,style=label,explode=true"`
    
    // Matrix style - /items;id=1;id=2
    IDs []string `schema:"ids,location=path,style=matrix,explode=true"`
}
```

### Headers and Cookies

```go
type AuthenticatedRequest struct {
    // Header parameter
    Authorization string `schema:"Authorization,location=header"`
    RequestID     string `schema:"X-Request-ID,location=header"`
    
    // Cookie parameter
    SessionToken string `schema:"session,location=cookie"`
}
```

### JSON Body

```go
type CreatePostRequest struct {
    // Query param for API version
    Version string `schema:"v,location=query"`
    
    // JSON body
    Body struct {
        Title   string   `schema:"title"`
        Content string   `schema:"content"`
        Tags    []string `schema:"tags"`
    } `body:"structured"`
}

// Handles: POST /posts?v=2
// Body: {"title": "Hello", "content": "World", "tags": ["go", "api"]}
```

### XML Body

```go
type XMLRequest struct {
    Body struct {
        Name  string `xml:"name"`
        Email string `xml:"email"`
    } `body:"structured"`
}

// POST /users (Content-Type: application/xml)
// Body: <user><name>John</name><email>john@example.com</email></user>
//
// Result: Body.Name == "John", Body.Email == "john@example.com"
```

**Note:** XML parsing uses Go's standard `xml.Unmarshal` with full support for XML struct tags. The target field must be a struct, slice, or string - `map[string]interface{}` is not supported since XML has a defined structure.

### URL-Encoded Form Body

```go
type FormRequest struct {
    Body struct {
        Username string `schema:"username"`
        Password string `schema:"password"`
    } `body:"structured"`
}

// Handles: POST /login (Content-Type: application/x-www-form-urlencoded)
// Body: username=john&password=secret
```

### File Upload

```go
// As bytes
type FileUploadRequest struct {
    Body []byte `body:"file"`
}

// As reader (for streaming large files)
type StreamingUploadRequest struct {
    Body io.ReadCloser `body:"file"`
}

// With query parameters
type VersionedUploadRequest struct {
    Version string `schema:"version,location=query"`
    Body    []byte `body:"file"`
}
```

### Multipart Form with Files

```go
type DocumentUploadRequest struct {
    Body struct {
        Title       string        `schema:"title"`
        Description string        `schema:"description"`
        Document    io.ReadCloser `schema:"document"`    // Single file
        Attachments []io.ReadCloser `schema:"attachments"` // Multiple files
    } `body:"multipart"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    codec := schema.NewDefaultCodec()
    
    var req DocumentUploadRequest
    if err := codec.DecodeRequest(r, nil, &req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Don't forget to close file readers
    if req.Body.Document != nil {
        defer req.Body.Document.Close()
    }
    for _, f := range req.Body.Attachments {
        defer f.Close()
    }
    
    // Process files...
}
```

### Mixed Parameters and Body

```go
type CompleteRequest struct {
    // Path parameter
    ResourceID string `schema:"id,location=path"`
    
    // Query parameters
    Format   string `schema:"format,location=query"`
    Page     int    `schema:"page,location=query"`
    PageSize int    `schema:"page_size,location=query"`
    
    // Header parameters
    Authorization string `schema:"Authorization,location=header"`
    RequestID     string `schema:"X-Request-ID,location=header"`
    
    // Cookie parameters
    SessionID string `schema:"session_id,location=cookie"`
    
    // Body
    Body struct {
        Action string            `schema:"action"`
        Data   map[string]string `schema:"data"`
    } `body:"structured"`
}
```

## API Reference

### Codec

The `Codec` is the main entry point for decoding HTTP requests.

#### `NewDefaultCodec() *Codec`

Creates a new Codec with default configuration. This is the recommended constructor for most use cases.

```go
codec := schema.NewDefaultCodec()
```

#### `NewCodec(metadata *Metadata, unmarshaler Unmarshaler, decoder Decoder) *Codec`

Creates a new Codec with custom dependencies for advanced use cases. Use this when you need full control over the decoding pipeline.

```go
// Create custom components
metadata := schema.NewMetadata(customRegistry)
decoder := schema.NewDecoder(metadata, "schema", "body")
unmarshaler := myCustomUnmarshaler{}

// Create codec with custom dependencies
codec := schema.NewCodec(metadata, unmarshaler, decoder)
```

#### `codec.DecodeRequest(request *http.Request, routerParams map[string]string, result any) error`

Decodes an HTTP request into the provided struct.

- `request`: The HTTP request to decode
- `routerParams`: Path parameters from your router (can be `nil`)
- `result`: Pointer to the target struct

```go
var req MyRequest
err := codec.DecodeRequest(r, routerParams, &req)
```

### Decoder Interface

```go
type Decoder interface {
    Decode(request *http.Request, routerParams map[string]string, metadata *StructMetadata) (map[string]any, error)
}
```

Implement this interface for custom decoding logic:

```go
type MyDecoder struct{}

func (d *MyDecoder) Decode(request *http.Request, routerParams map[string]string, metadata *StructMetadata) (map[string]any, error) {
    // Custom decoding logic
    return map[string]any{"key": "value"}, nil
}

// Use with custom codec
metadata := schema.NewDefaultMetadata()
unmarshaler := mapstructure.NewDefaultUnmarshaler()
codec := schema.NewCodec(metadata, unmarshaler, &MyDecoder{})
```

### Unmarshaler Interface

```go
type Unmarshaler interface {
    Unmarshal(data map[string]any, result any) error
}
```

The default unmarshaler uses `mapstructure`. Implement this interface for custom unmarshaling:

```go
type MyUnmarshaler struct{}

func (u *MyUnmarshaler) Unmarshal(data map[string]any, result any) error {
    // Custom unmarshaling logic
    return nil
}

// Use with custom codec
metadata := schema.NewDefaultMetadata()
decoder := schema.NewDefaultDecoder()
codec := schema.NewCodec(metadata, &MyUnmarshaler{}, decoder)
```

### Metadata

Manages struct metadata parsing and caching.

```go
// Default metadata with schema and body tag parsers
metadata := schema.NewDefaultMetadata()

// Custom metadata with custom tag parsers
registry := schema.NewTagParserRegistry(
    schema.WithTagParser("schema", schema.ParseSchemaTag, schema.DefaultSchemaMetadata),
    schema.WithTagParser("body", schema.ParseBodyTag),
    schema.WithTagParser("custom", myCustomParser),
)
metadata := schema.NewMetadata(registry)
```

### TagParserRegistry

Manages registration of custom tag parsers for extensibility.

```go
// Create registry with custom parsers
registry := schema.NewTagParserRegistry(
    schema.WithTagParser("schema", schema.ParseSchemaTag, schema.DefaultSchemaMetadata),
    schema.WithTagParser("validate", myValidateParser),
)

// Use with metadata
metadata := schema.NewMetadata(registry)
```

### Exported Types

#### Core Types

```go
// Codec - Main entry point for HTTP request decoding
type Codec struct {
    metadata    *Metadata
    unmarshaler Unmarshaler
    decoder     Decoder
}

// Metadata - Manages struct metadata parsing and caching
type Metadata struct {
    cache *metadataCache
}

// StructMetadata - Cached metadata for a struct type
type StructMetadata struct {
    Type         reflect.Type
    Fields       []FieldMetadata
    fieldsByName map[string]*FieldMetadata
}

// FieldMetadata - Metadata for individual struct fields
type FieldMetadata struct {
    StructFieldName string
    Index           int
    Embedded        bool
    Type            reflect.Type
    TagMetadata     map[string]any // "schema" -> *SchemaMetadata, "body" -> *BodyMetadata
}
```

#### Parameter Types

```go
// ParameterLocation - Where parameters come from
type ParameterLocation string

const (
    LocationQuery  ParameterLocation = "query"
    LocationPath   ParameterLocation = "path"
    LocationHeader ParameterLocation = "header"
    LocationCookie ParameterLocation = "cookie"
)

// Style - How arrays/objects are serialized
type Style string

const (
    StyleForm          Style = "form"
    StyleSimple        Style = "simple"
    StyleMatrix        Style = "matrix"
    StyleLabel         Style = "label"
    StyleSpaceDelimited Style = "spaceDelimited"
    StylePipeDelimited Style = "pipeDelimited"
    StyleDeepObject    Style = "deepObject"
)

// BodyType - Type of request body
type BodyType string

const (
    BodyTypeStructured BodyType = "structured"
    BodyTypeFile       BodyType = "file"
    BodyTypeMultipart  BodyType = "multipart"
)
```

#### Tag Metadata Types

```go
// SchemaMetadata - Parsed schema tag metadata
type SchemaMetadata struct {
    ParamName string
    MapKey    string
    Location  ParameterLocation
    Style     Style
    Explode   bool
    Required  bool
}

// BodyMetadata - Parsed body tag metadata
type BodyMetadata struct {
    MapKey   string
    BodyType BodyType
    Required bool
}
```

### Constructors

#### Codec Constructors

```go
// NewDefaultCodec creates a codec with default configuration
// Recommended for most use cases
func NewDefaultCodec() *Codec

// NewCodec creates a codec with custom dependencies
// Use for advanced customization
func NewCodec(metadata *Metadata, unmarshaler Unmarshaler, decoder Decoder) *Codec
```

#### Metadata Constructors

```go
// NewDefaultMetadata creates metadata with default tag parsers
func NewDefaultMetadata() *Metadata

// NewMetadata creates metadata with custom tag parser registry
func NewMetadata(registry *TagParserRegistry) *Metadata
```

#### Decoder Constructors

```go
// NewDecoder creates a decoder with custom tag names
func NewDecoder(metadata *Metadata, schemaTag string, bodyTag string) Decoder

// NewDefaultDecoder creates a decoder with default tag names ("schema", "body")
func NewDefaultDecoder() Decoder
```

#### Tag Parser Registry

```go
// NewTagParserRegistry creates a registry with specified parsers
func NewTagParserRegistry(opts ...TagParserRegistryOption) *TagParserRegistry

// NewDefaultTagParserRegistry creates registry with schema and body parsers
func NewDefaultTagParserRegistry() *TagParserRegistry

// WithTagParser adds a parser to the registry
func WithTagParser(tagName string, parser TagParserFunc, defaultFunc ...DefaultMetadataFunc) TagParserRegistryOption
```

### Methods

#### Codec Methods

```go
// DecodeRequest decodes an HTTP request into a struct
// result must be a pointer to the target struct
func (c *Codec) DecodeRequest(request *http.Request, routerParams map[string]string, result any) error
```

#### Metadata Methods

```go
// GetStructMetadata retrieves or builds cached struct metadata
func (m *Metadata) GetStructMetadata(typ reflect.Type) (*StructMetadata, error)
```

#### StructMetadata Methods

```go
// Field returns FieldMetadata by field name
func (m *StructMetadata) Field(fieldName string) (*FieldMetadata, bool)
```

#### Utility Functions

```go
// GetTagMetadata safely extracts typed metadata from field tags
func GetTagMetadata[T any](f *FieldMetadata, tagName string) (T, bool)

// HasTag checks if field has a specific tag
func (f *FieldMetadata) HasTag(tagName string) bool
```

### Tag Parser Functions

#### Schema Tag Functions

```go
// ParseSchemaTag parses schema tag into SchemaMetadata
func ParseSchemaTag(field reflect.StructField, index int, tagValue string) (any, error)

// DefaultSchemaMetadata creates default metadata for untagged fields
func DefaultSchemaMetadata(field reflect.StructField, index int) any
```

#### Body Tag Functions

```go
// ParseBodyTag parses body tag into BodyMetadata
func ParseBodyTag(field reflect.StructField, index int, tagValue string) (any, error)
```

### Interfaces

```go
// Decoder interface for HTTP request decoding
type Decoder interface {
    Decode(request *http.Request, routerParams map[string]string, metadata *StructMetadata) (map[string]any, error)
}

// Unmarshaler interface for map-to-struct conversion
type Unmarshaler interface {
    Unmarshal(data map[string]any, result any) error
}

// TagParserFunc function signature for parsing struct tags
type TagParserFunc func(field reflect.StructField, index int, tagValue string) (any, error)

// DefaultMetadataFunc creates default metadata for untagged fields
type DefaultMetadataFunc func(field reflect.StructField, index int) any
```

## Style/Location Compatibility

| Location | Allowed Styles | Default Style | Default Explode |
|----------|---------------|---------------|-----------------|
| query | form, spaceDelimited, pipeDelimited, deepObject | form | true (form, deepObject), false (others) |
| path | simple, label, matrix | simple | false |
| header | simple | simple | false |
| cookie | form | form | true |

## Default Behaviors

### Fields Without Tags

**Important:** Fields without `schema` or `body` tags are automatically treated as query parameters with form style:

```go
type Request struct {
    Name string // Equivalent to: `schema:"Name,location=query,style=form,explode=true"`
}
```

To skip a field entirely from decoding, use `schema:"-"`:

```go
type Request struct {
    Internal string `schema:"-"` // Not decoded
}
```

### Path Parameters

Path parameters are automatically marked as required:

```go
type Request struct {
    ID string `schema:"id,location=path"` // Always required
}
```

### Body Content-Type Detection

The body decoder automatically detects content type:

- `application/json` → JSON decoding
- `application/xml`, `text/xml` → XML decoding
- `application/x-www-form-urlencoded` → Form decoding
- `multipart/form-data` → Multipart form decoding (requires `body:"multipart"`)
- `application/octet-stream` → Raw file bytes (requires `body:"file"`)

## Performance

The library is optimized for production use with efficient caching and minimal allocations.

### Key Optimizations

- ✅ **Struct metadata caching** - Reflection done once per type, cached globally
- ✅ **Shared codec instances** - Reuse codecs across requests for cache hits
- ✅ **Efficient body parsing** - Streaming parsers for large content
- ✅ **Zero-copy operations** - Where possible, avoids unnecessary allocations

### Metadata Caching

Struct metadata is cached per type. The first decode for a struct type parses and caches the metadata; subsequent decodes reuse the cache.

**Best Practice:** Create the codec once at application startup and reuse it:

```go
// ✅ GOOD: Create once, reuse across all requests
var codec = schema.NewDefaultCodec()

func handler(w http.ResponseWriter, r *http.Request) {
    var req MyRequest
    codec.DecodeRequest(r, nil, &req) // Cache hit after first request
}
```

**Avoid:** Creating a new codec per request:

```go
// ❌ BAD: No cache benefit, slower
func handler(w http.ResponseWriter, r *http.Request) {
    codec := schema.NewDefaultCodec() // Creates new cache every time
    var req MyRequest
    codec.DecodeRequest(r, nil, &req)
}
```

### Sharing Cache

Share metadata cache between multiple codecs:

```go
// Create shared metadata instance
metadata := schema.NewDefaultMetadata()

// Create custom components that share the metadata
decoder1 := schema.NewDecoder(metadata, "schema", "body")
decoder2 := schema.NewDecoder(metadata, "schema", "body")

// Both codecs will use the same metadata cache
codec1 := schema.NewCodec(metadata, unmarshaler1, decoder1)
codec2 := schema.NewCodec(metadata, unmarshaler2, decoder2)
```

For most use cases, a single codec instance is sufficient and recommended:

```go
// Single codec shared across all handlers (recommended)
var codec = schema.NewDefaultCodec()
```

## Thread Safety

**Safe for concurrent use:** All main types are safe for concurrent access after creation.

### Concurrent Usage

```go
// ✅ Safe: Shared codec across goroutines/requests
var codec = schema.NewDefaultCodec()

func handler1(w http.ResponseWriter, r *http.Request) {
    var req Request1
    codec.DecodeRequest(r, nil, &req) // Concurrent safe
}

func handler2(w http.ResponseWriter, r *http.Request) {
    var req Request2
    codec.DecodeRequest(r, nil, &req) // Concurrent safe
}
```

### Metadata Cache

The internal metadata cache uses thread-safe operations:
- First decode of a struct type: Cache miss (parses and caches)
- Subsequent decodes: Cache hit (thread-safe read)

### Best Practices

1. **Create codecs at startup**, not per request
2. **Share codec instances** across handlers/goroutines
3. **No mutexes needed** - internal synchronization handled

## Error Handling

The `DecodeRequest` method can return errors in several scenarios:

### Common Error Types

```go
var req MyRequest
err := codec.DecodeRequest(r, routerParams, &req)
if err != nil {
    // Possible errors:
    // - HTTP request parsing errors (malformed query strings, multipart forms)
    // - Body decoding errors (invalid JSON/XML, unsupported content types)
    // - Parameter validation errors (missing required fields, invalid styles)
    // - Type conversion errors (string to int, etc.)
    http.Error(w, err.Error(), http.StatusBadRequest)
    return
}
```

### Error Scenarios

| Scenario | Example Request | Error Message |
|----------|----------------|---------------|
| Invalid JSON body | `POST / {"name": invalid}` | `failed to unmarshal JSON` |
| Malformed query | `GET /?name=%ZZ` | `failed to parse query string` |
| Missing required path param | Path param not in routerParams | `missing required path parameter` |
| Type conversion | `?count=abc` for int field | `cannot convert string to int` |
| Invalid multipart | Corrupt boundary | `failed to parse multipart form` |

### Best Practices

1. **Always check errors** from `DecodeRequest`
2. **Return HTTP 400** for client errors (malformed requests)
3. **Return HTTP 500** only for server configuration errors
4. **Log errors** for debugging but sanitize before showing to users
5. **Validate business logic separately** after successful decoding

## Testing

### Running Tests

```bash
# Run all tests
go test -v ./...

# Run with race detector
go test -race ./...

# Run with coverage
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Test Categories

- ✅ **Unit tests** - Core functionality (90%+ coverage)
- ✅ **Integration tests** - HTTP request/response cycles
- ✅ **Edge case tests** - Malformed inputs, boundary conditions
- ✅ **Concurrent tests** - Thread safety validation

## Limitations and Best Practices

### Missing Fields vs Zero Values

⚠️ **Important:** By default, missing fields receive Go zero values:

```go
type Request struct {
    Count int `schema:"count"` // Missing → 0 (zero value, not explicitly set!)
}

// GET /?name=test (no count parameter)
// req.Count == 0 (zero value, not missing!)
```

**Solutions:**

1. **Use pointers** to detect missing fields:
   ```go
   type Request struct {
       Count *int `schema:"count"` // nil if missing, *42 if present
   }
   ```

2. **Use default tags** for explicit defaults:
   ```go
   type Request struct {
       Count int `schema:"count" default:"10"`
   }
   ```

3. **Validate after unmarshaling** if fields are required

### Type Safety Considerations

The library performs **best-effort type conversion**:

```go
type Request struct {
    Count int `schema:"count"`
}

// These all work (maybe not what you want):
// ?count=42     → req.Count = 42 ✓
// ?count=3.14   → req.Count = 3 (truncates) ⚠️
// ?count=abc    → error (can't convert) ✓
```

### File Upload Memory Usage

⚠️ **Multipart uploads with large files** are loaded into memory:

```go
type Upload struct {
    File []byte `body:"file"` // Entire file in memory!
}
```

**For large files:** Use `io.ReadCloser` and stream processing:

```go
type Upload struct {
    File io.ReadCloser `body:"file"` // Streams file content
}

func handler(w http.ResponseWriter, r *http.Request) {
    var upload Upload
    codec.DecodeRequest(r, nil, &upload)
    defer upload.File.Close() // Don't forget to close!

    // Stream processing
    io.Copy(destination, upload.File)
}
```

### Content-Type Detection

Body decoding automatically detects content type, but **explicit configuration is recommended**:

```go
// ✅ Explicit (recommended)
type Request struct {
    Data User `body:"structured"` // Always expects structured data
}

// ❌ Implicit (fragile)
type Request struct {
    Data User // May fail if content-type is wrong
}
```

### Router Integration

**Path parameters must be provided by your router:**

```go
// Chi router
func handler(w http.ResponseWriter, r *http.Request) {
    routerParams := map[string]string{
        "user_id": chi.URLParam(r, "user_id"),
    }
    codec.DecodeRequest(r, routerParams, &req)
}

// Gorilla Mux
func handler(w http.ResponseWriter, r *http.Request) {
    codec.DecodeRequest(r, mux.Vars(r), &req)
}
```
