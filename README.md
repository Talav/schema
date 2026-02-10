# Schema

[![tag](https://img.shields.io/github/tag/talav/schema.svg)](https://github.com/talav/schema/tag)
[![Go Reference](https://pkg.go.dev/badge/github.com/talav/schema.svg)](https://pkg.go.dev/github.com/talav/schema)
[![Go Report Card](https://goreportcard.com/badge/github.com/talav/schema)](https://goreportcard.com/report/github.com/talav/schema)
[![CI](https://github.com/talav/schema/actions/workflows/schema-ci.yml/badge.svg)](https://github.com/talav/schema/actions)
[![codecov](https://codecov.io/gh/talav/schema/graph/badge.svg)](https://codecov.io/gh/talav/schema)
[![License](https://img.shields.io/github/license/talav/tagparser)](./LICENSE)

Decode HTTP requests into Go structs with OpenAPI 3.0/3.1 compliance. Define your request structure with struct tags, and Schema handles the rest.

## Features

- **OpenAPI 3.0/3.1 Compliant** - Full support for all parameter serialization styles and locations
- **Unified API** - Query, path, header, cookie parameters and request bodies in one call
- **Multiple Body Formats** - JSON, XML, forms, multipart, file uploads
- **Performance Optimized** - Metadata caching, zero allocations where possible
- **Extensible** - Custom decoders, unmarshalers, and tag parsers
- **Type Safe** - Automatic type conversion with validation

## Installation

```bash
go get github.com/talav/schema
```

## Quick Start

Define your request structure and decode in one line:

```go
package main

import (
    "fmt"
    "net/http"
    "github.com/talav/schema"
)

type CreateUserRequest struct {
    // Path parameter (from router)
    OrgID string `schema:"org_id,location=path"`
    
    // Query parameter
    Version string `schema:"version" default:"v1"`
    
    // Header parameter
    APIKey string `schema:"X-API-Key,location=header"`
    
    // Request body (JSON, XML, or form - auto-detected)
    Body struct {
        Name  string `schema:"name"`
        Email string `schema:"email"`
        Age   int    `schema:"age"`
    } `body:"structured"`
}

func main() {
    // Create codec once, reuse across all requests
    codec := schema.NewDefaultCodec()
    
    http.HandleFunc("/orgs/{org_id}/users", func(w http.ResponseWriter, r *http.Request) {
        // Router params come from your router (chi, gorilla, etc.)
        routerParams := map[string]string{"org_id": "123"}
        
        var req CreateUserRequest
        if err := codec.DecodeRequest(r, routerParams, &req); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        
        // Request is decoded and validated
        fmt.Fprintf(w, "Creating user %s in org %s\n", req.Body.Name, req.OrgID)
    })
    
    http.ListenAndServe(":8080", nil)
}
```

**Try it:**
```bash
curl -X POST http://localhost:8080/orgs/acme/users?version=v2 \
  -H "X-API-Key: secret" \
  -H "Content-Type: application/json" \
  -d '{"name": "Alice", "email": "alice@example.com", "age": 30}'
```

## Documentation

**[Full Documentation](https://talav.github.io/schema)**

### Getting Started
- [Quick Start Guide](https://talav.github.io/schema/getting-started/quick-start/) - Build your first handler
- [Core Concepts](https://talav.github.io/schema/getting-started/concepts/) - Understand how it works

### User Guides
- [Parameters](https://talav.github.io/schema/guides/parameters/) - Query, path, header, cookie parameters
- [Request Bodies](https://talav.github.io/schema/guides/request-bodies/) - JSON, XML, forms, file uploads
- [Serialization Styles](https://talav.github.io/schema/guides/serialization/) - OpenAPI styles and explode
- [Type Conversion](https://talav.github.io/schema/guides/type-conversion/) - Automatic type handling

### Advanced
- [Extensibility](https://talav.github.io/schema/advanced/extensibility/) - Custom decoders and unmarshalers
- [Custom Tag Parsers](https://talav.github.io/schema/advanced/custom-tags/) - Extend the metadata system

### API Reference
- [pkg.go.dev](https://pkg.go.dev/github.com/talav/schema) - Complete API documentation

## Key Concepts

### Parameter Locations

Parameters can be extracted from any part of the HTTP request:

```go
type Request struct {
    ID      string `schema:"id,location=path"`        // /users/{id}
    Search  string `schema:"q,location=query"`        // ?q=golang
    APIKey  string `schema:"X-API-Key,location=header"` // Header: X-API-Key
    Session string `schema:"session,location=cookie"`   // Cookie: session=xyz
}
```

[Learn more about parameters →](https://talav.github.io/schema/guides/parameters/)

### Request Bodies

Support for all common body formats with automatic content-type detection:

```go
// JSON, XML, or URL-encoded forms
type JSONRequest struct {
    Body User `body:"structured"`
}

// File upload (small files)
type FileUpload struct {
    File []byte `body:"file"`
}

// File streaming (large files)
type StreamUpload struct {
    File io.ReadCloser `body:"file"`
}

// Multipart forms with files
type FormUpload struct {
    Body struct {
        Title    string        `schema:"title"`
        Document io.ReadCloser `schema:"document"`
    } `body:"multipart"`
}
```

[Learn more about request bodies →](https://talav.github.io/schema/guides/request-bodies/)

### OpenAPI Serialization Styles

Full support for OpenAPI 3.0/3.1 parameter serialization:

```go
type Request struct {
    // Form style (default): ?ids=1&ids=2 or ?ids=1,2,3
    IDs []int `schema:"ids,style=form"`
    
    // Deep object: ?filter[status]=active&filter[type]=user
    Filter struct {
        Status string `schema:"status"`
        Type   string `schema:"type"`
    } `schema:"filter,style=deepObject"`
    
    // Space delimited: ?tags=go%20api%20http
    Tags []string `schema:"tags,style=spaceDelimited"`
}
```

[Learn more about serialization →](https://talav.github.io/schema/guides/serialization/)

## Architecture

Schema consists of four main components:

1. **Codec** - High-level API orchestrating the pipeline
2. **Metadata** - Parses and caches struct tags (fast lookups)
3. **Decoder** - Extracts HTTP parameters into maps
4. **Unmarshaler** - Converts maps to typed structs (uses [mapstructure](https://github.com/talav/mapstructure))


## Examples

### With Chi Router

```go
import (
    "github.com/go-chi/chi/v5"
    "github.com/talav/schema"
)

func main() {
    codec := schema.NewDefaultCodec()
    r := chi.NewRouter()
    
    r.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
        type Request struct {
            ID   string `schema:"id,location=path"`
            Page int    `schema:"page" default:"1"`
        }
        
        var req Request
        if err := codec.DecodeRequest(r, chi.RouteContext(r.Context()).URLParams, &req); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        
        // Use req.ID and req.Page
    })
    
    http.ListenAndServe(":8080", r)
}
```

### File Upload with Streaming

```go
type UploadRequest struct {
    Filename string        `schema:"filename"`
    File     io.ReadCloser `body:"file"` // Streams large files
}

func handler(w http.ResponseWriter, r *http.Request) {
    var req UploadRequest
    if err := codec.DecodeRequest(r, nil, &req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    defer req.File.Close()
    
    // Stream to disk
    out, _ := os.Create("/uploads/" + req.Filename)
    io.Copy(out, req.File)
    out.Close()
}
```

### Multipart Form with Files

```go
type FormRequest struct {
    Body struct {
        Title       string        `schema:"title"`
        Description string        `schema:"description"`
        Document    io.ReadCloser `schema:"document"`
    } `body:"multipart"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    var req FormRequest
    if err := codec.DecodeRequest(r, nil, &req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    defer req.Body.Document.Close()
    
    // Process form + file
}
```

## Extensibility

Customize behavior with your own decoders and unmarshalers:

```go
// Custom decoder (e.g., add logging)
type LoggingDecoder struct {
    decoder schema.Decoder
}

func (d *LoggingDecoder) Decode(...) (map[string]any, error) {
    log.Println("Decoding request")
    return d.decoder.Decode(...)
}

// Custom unmarshaler (e.g., add validation)
type ValidatingUnmarshaler struct {
    unmarshaler schema.Unmarshaler
}

func (u *ValidatingUnmarshaler) Unmarshal(data map[string]any, result any) error {
    if err := u.unmarshaler.Unmarshal(data, result); err != nil {
        return err
    }
    // Add validation logic
    return validate(result)
}

// Use custom components
metadata := schema.NewDefaultMetadata()
decoder := NewLoggingDecoder(schema.NewDefaultDecoder())
unmarshaler := NewValidatingUnmarshaler(mapstructure.NewDefaultUnmarshaler())
codec := schema.NewCodec(metadata, unmarshaler, decoder)
```

[Learn more about extensibility →](https://talav.github.io/schema/advanced/extensibility/)

## Testing

```bash
# Run all tests
go test ./...

# With race detector
go test -race ./...

# With coverage
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing`)
5. Open a Pull Request

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

Built on top of:
- [mapstructure](https://github.com/talav/mapstructure) - Flexible struct unmarshaling
- [tagparser](https://github.com/talav/tagparser) - Struct tag parsing utilities
