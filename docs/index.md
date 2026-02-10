# Schema

[![Go Reference](https://pkg.go.dev/badge/github.com/talav/schema.svg)](https://pkg.go.dev/github.com/talav/schema)
[![Go Report Card](https://goreportcard.com/badge/github.com/talav/schema)](https://goreportcard.com/report/github.com/talav/schema)
[![CI](https://github.com/talav/schema/actions/workflows/schema-ci.yml/badge.svg)](https://github.com/talav/schema/actions)
[![codecov](https://codecov.io/gh/talav/schema/graph/badge.svg)](https://codecov.io/gh/talav/schema)

**Decode HTTP requests into Go structs with OpenAPI 3.0 compliance**

A Go package for decoding HTTP requests into Go structs with full support for [OpenAPI 3.0/3.1 parameter serialization](https://spec.openapis.org/oas/v3.1.0#parameter-serialization). Handles query, path, header, cookie parameters and request bodies (JSON, XML, forms, multipart, files) in a single unified API.

Decouples business logic from HTTP request handling. Request types are defined with struct tags. The package extracts query parameters, path variables, headers, cookies, and request bodies, converting them into typed Go structs with OpenAPI 3.0 compliance.

## Key Features

- **OpenAPI 3.0/3.1 Compliant** - Full support for parameter serialization styles
- **All Parameter Locations** - Query, path, header, cookie parameters
- **All Serialization Styles** - form, simple, matrix, label, spaceDelimited, pipeDelimited, deepObject
- **Multiple Body Types** - JSON, XML, URL-encoded forms, multipart forms, file uploads
- **Type Conversion** - Automatic conversion between types with sensible defaults
- **Performance Optimized** - Metadata caching for high-throughput applications
- **Thread-Safe** - Safe for concurrent use across goroutines
- **Extensible** - Custom decoders, unmarshalers, and tag parsers

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
    
    // Router params from your router (chi, gorilla, etc.)
    routerParams := map[string]string{"org_id": "123"}
    
    var req CreateUserRequest
    if err := codec.DecodeRequest(r, routerParams, &req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Use req.Version, req.OrgID, req.APIKey, req.Body.Name, req.Body.Email
}
```

## Getting Started

### Quick Start

Install and build your first request handler

[Quick Start →](getting-started/quick-start.md)

### Core Concepts

Understand the architecture and how it works

[Learn Concepts →](getting-started/concepts.md)

## Documentation

### User Guides

Learn about parameters, bodies, and serialization

[Browse Guides →](guides/parameters.md)

### Advanced Topics

Extensibility and customization

[Advanced Docs →](advanced/extensibility.md)

## API Documentation

Complete API documentation with types, methods, and interfaces is available at [pkg.go.dev/github.com/talav/schema](https://pkg.go.dev/github.com/talav/schema).

## Architecture

The package consists of four main components:

1. **Codec** - High-level API that orchestrates the decoding pipeline
2. **Metadata** - Parses and caches struct tag metadata for performance
3. **Decoder** - Extracts parameters from HTTP requests into maps
4. **Unmarshaler** - Converts maps to typed structs (default: mapstructure)

All components are thread-safe and designed for reuse across concurrent requests.

## Why Schema?

Traditional HTTP handlers mix request parsing with business logic:

```go
// Traditional approach - mixed concerns
func handler(w http.ResponseWriter, r *http.Request) {
    name := r.URL.Query().Get("name")
    ageStr := r.URL.Query().Get("age")
    age, _ := strconv.Atoi(ageStr)
    apiKey := r.Header.Get("X-API-Key")
    
    var body RequestBody
    json.NewDecoder(r.Body).Decode(&body)
    
    // Finally, business logic...
}
```

With Schema, define your request structure once:

```go
// With schema - clean separation
type UserRequest struct {
    Name   string `schema:"name"`
    Age    int    `schema:"age"`
    APIKey string `schema:"X-API-Key,location=header"`
    Body   struct {
        Email string `schema:"email"`
    } `body:"structured"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    var req UserRequest
    codec.DecodeRequest(r, nil, &req)
    // Business logic with strongly-typed req
}
```

## What's Next?

- [Quick Start](getting-started/quick-start.md)
- [Serialization Styles](guides/serialization.md)
- [Extensibility](advanced/extensibility.md)
- Visit [pkg.go.dev](https://pkg.go.dev/github.com/talav/schema)
