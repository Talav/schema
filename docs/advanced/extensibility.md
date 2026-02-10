# Extensibility Guide

Schema is designed for extensibility. This guide covers how to customize the library's core interfaces: Decoder and Unmarshaler.

!!! tip "Custom Tag Parsers"

    Looking to add custom struct tags? See the [Custom Tag Parsers Guide](custom-tags.md) for details on extending the metadata system.

## Overview

Schema provides two main extension points:

1. **Custom Decoder** - Customize how HTTP parameters are extracted from requests
2. **Custom Unmarshaler** - Customize how extracted data is converted to structs

Both interfaces use the decorator pattern, allowing you to wrap default implementations rather than reimplementing everything.

## Custom Decoder

The `Decoder` interface extracts parameters from HTTP requests into maps.

### Interface

```go
type Decoder interface {
    Decode(request *http.Request, routerParams map[string]string, metadata *StructMetadata) (map[string]any, error)
}
```

### Example 1: Request Logging

Wrap the default decoder to add structured logging:

```go
package main

import (
    "log"
    "net/http"
    "time"
    
    "github.com/talav/schema"
    "github.com/talav/mapstructure"
    "github.com/go-chi/chi/v5"
)

type LoggingDecoder struct {
    decoder schema.Decoder
}

func NewLoggingDecoder(decoder schema.Decoder) *LoggingDecoder {
    return &LoggingDecoder{decoder: decoder}
}

func (d *LoggingDecoder) Decode(
    request *http.Request,
    routerParams map[string]string,
    metadata *schema.StructMetadata,
) (map[string]any, error) {
    start := time.Now()
    
    log.Printf("[DECODE] %s %s - started", request.Method, request.URL.Path)
    
    data, err := d.decoder.Decode(request, routerParams, metadata)
    
    duration := time.Since(start)
    if err != nil {
        log.Printf("[DECODE] %s %s - failed in %v: %v", 
            request.Method, request.URL.Path, duration, err)
    } else {
        log.Printf("[DECODE] %s %s - success in %v (%d fields)", 
            request.Method, request.URL.Path, duration, len(data))
    }
    
    return data, err
}

// Usage
func main() {
    // Create codec with logging decoder
    metadata := schema.NewDefaultMetadata()
    defaultDecoder := schema.NewDefaultDecoder()
    loggingDecoder := NewLoggingDecoder(defaultDecoder)
    unmarshaler := mapstructure.NewDefaultUnmarshaler()
    
    codec := schema.NewCodec(metadata, unmarshaler, loggingDecoder)
    
    // Use in handler
    r := chi.NewRouter()
    r.Get("/users/{id}", func(w http.ResponseWriter, req *http.Request) {
        type Request struct {
            ID   string `schema:"id,location=path"`
            Page int    `schema:"page" default:"1"`
        }
        
        // Convert chi.RouteParams to map[string]string
        rctx := chi.RouteContext(req.Context())
        routeParams := make(map[string]string)
        for i, key := range rctx.URLParams.Keys {
            routeParams[key] = rctx.URLParams.Values[i]
        }
        
        var request Request
        if err := codec.DecodeRequest(req, routeParams, &request); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        
        w.Write([]byte("OK"))
    })
    
    http.ListenAndServe(":8080", r)
}
```

### Example 2: Request Metrics

Add Prometheus metrics for request decoding:

```go
package main

import (
    "net/http"
    "time"
    
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "github.com/talav/schema"
)

var (
    decodeRequests = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "schema_decode_requests_total",
            Help: "Total number of decode requests",
        },
        []string{"method", "status"},
    )
    
    decodeDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "schema_decode_duration_seconds",
            Help:    "Time spent decoding requests",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method"},
    )
)

type MetricsDecoder struct {
    decoder schema.Decoder
}

func NewMetricsDecoder(decoder schema.Decoder) *MetricsDecoder {
    return &MetricsDecoder{decoder: decoder}
}

func (d *MetricsDecoder) Decode(
    request *http.Request,
    routerParams map[string]string,
    metadata *schema.StructMetadata,
) (map[string]any, error) {
    start := time.Now()
    
    data, err := d.decoder.Decode(request, routerParams, metadata)
    
    duration := time.Since(start).Seconds()
    decodeDuration.WithLabelValues(request.Method).Observe(duration)
    
    status := "success"
    if err != nil {
        status = "error"
    }
    decodeRequests.WithLabelValues(request.Method, status).Inc()
    
    return data, err
}
```

### Example 3: Request Context Injection

Extract and inject request context (like request IDs) into the decoded data:

```go
package main

import (
    "net/http"
    
    "github.com/talav/schema"
)

type ContextDecoder struct {
    decoder schema.Decoder
}

func NewContextDecoder(decoder schema.Decoder) *ContextDecoder {
    return &ContextDecoder{decoder: decoder}
}

func (d *ContextDecoder) Decode(
    request *http.Request,
    routerParams map[string]string,
    metadata *schema.StructMetadata,
) (map[string]any, error) {
    data, err := d.decoder.Decode(request, routerParams, metadata)
    if err != nil {
        return nil, err
    }
    
    // Inject request context fields
    if requestID := request.Header.Get("X-Request-ID"); requestID != "" {
        data["request_id"] = requestID
    }
    
    if userAgent := request.Header.Get("User-Agent"); userAgent != "" {
        data["user_agent"] = userAgent
    }
    
    // Add remote IP
    data["remote_ip"] = request.RemoteAddr
    
    return data, nil
}

// Usage with context-aware request struct
type ContextualRequest struct {
    // Regular fields
    Name string `schema:"name"`
    
    // Context fields (not from schema tags)
    RequestID string
    UserAgent string
    RemoteIP  string
}
```

## Custom Unmarshaler

The `Unmarshaler` interface converts maps to typed structs.

### Interface

```go
type Unmarshaler interface {
    Unmarshal(data map[string]any, result any) error
}
```

### Example 1: Field Transformation

Transform field values during unmarshaling:

```go
package main

import (
    "strings"
    
    "github.com/talav/schema"
    "github.com/talav/mapstructure"
)

type TransformingUnmarshaler struct {
    unmarshaler schema.Unmarshaler
}

func NewTransformingUnmarshaler(unmarshaler schema.Unmarshaler) *TransformingUnmarshaler {
    return &TransformingUnmarshaler{unmarshaler: unmarshaler}
}

func (u *TransformingUnmarshaler) Unmarshal(data map[string]any, result any) error {
    // Transform string fields to lowercase before unmarshaling
    transformed := make(map[string]any)
    for key, value := range data {
        if str, ok := value.(string); ok {
            transformed[key] = strings.ToLower(str)
        } else {
            transformed[key] = value
        }
    }
    
    return u.unmarshaler.Unmarshal(transformed, result)
}

// Usage
func main() {
    metadata := schema.NewDefaultMetadata()
    decoder := schema.NewDefaultDecoder()
    baseUnmarshaler := mapstructure.NewDefaultUnmarshaler()
    transformingUnmarshaler := NewTransformingUnmarshaler(baseUnmarshaler)
    
    codec := schema.NewCodec(metadata, transformingUnmarshaler, decoder)
    
    // Now all string fields are automatically lowercased
}
```

### Example 2: Validation Hook

Add post-unmarshal validation:

```go
package main

import (
    "fmt"
    
    "github.com/talav/schema"
)

// Validator interface for structs that can validate themselves
type Validator interface {
    Validate() error
}

type ValidatingUnmarshaler struct {
    unmarshaler schema.Unmarshaler
}

func NewValidatingUnmarshaler(unmarshaler schema.Unmarshaler) *ValidatingUnmarshaler {
    return &ValidatingUnmarshaler{unmarshaler: unmarshaler}
}

func (u *ValidatingUnmarshaler) Unmarshal(data map[string]any, result any) error {
    // First, unmarshal the data
    if err := u.unmarshaler.Unmarshal(data, result); err != nil {
        return err
    }
    
    // Then validate if the struct implements Validator
    if validator, ok := result.(Validator); ok {
        if err := validator.Validate(); err != nil {
            return fmt.Errorf("validation failed: %w", err)
        }
    }
    
    return nil
}

// Example request struct with validation
type CreateUserRequest struct {
    Body struct {
        Name  string `schema:"name"`
        Email string `schema:"email"`
        Age   int    `schema:"age"`
    } `body:"structured"`
}

func (r *CreateUserRequest) Validate() error {
    if r.Body.Name == "" {
        return fmt.Errorf("name is required")
    }
    if r.Body.Age < 18 {
        return fmt.Errorf("must be 18 or older")
    }
    if !strings.Contains(r.Body.Email, "@") {
        return fmt.Errorf("invalid email format")
    }
    return nil
}

// Usage in handler
func createUserHandler(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    
    // Validation happens automatically during DecodeRequest
    if err := codec.DecodeRequest(r, nil, &req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Request is already validated at this point
    fmt.Fprintf(w, "Created user: %s\n", req.Body.Name)
}
```

### Example 3: Audit Logging

Log all unmarshaled values for audit purposes:

```go
package main

import (
    "encoding/json"
    "log"
    "reflect"
    
    "github.com/talav/schema"
)

type AuditUnmarshaler struct {
    unmarshaler schema.Unmarshaler
}

func NewAuditUnmarshaler(unmarshaler schema.Unmarshaler) *AuditUnmarshaler {
    return &AuditUnmarshaler{unmarshaler: unmarshaler}
}

func (u *AuditUnmarshaler) Unmarshal(data map[string]any, result any) error {
    // Unmarshal first
    if err := u.unmarshaler.Unmarshal(data, result); err != nil {
        return err
    }
    
    // Log the unmarshaled result for audit
    typeName := reflect.TypeOf(result).String()
    jsonData, _ := json.Marshal(result)
    log.Printf("[AUDIT] Unmarshaled %s: %s", typeName, string(jsonData))
    
    return nil
}
```

## Combining Multiple Extensions

You can chain multiple decorators for complex behavior:

```go
package main

import (
    "github.com/talav/schema"
    "github.com/talav/mapstructure"
)

func NewProductionCodec() *schema.Codec {
    metadata := schema.NewDefaultMetadata()
    
    // Chain decoders: Logging → Metrics → Default
    decoder := schema.NewDefaultDecoder()
    decoder = NewMetricsDecoder(decoder)
    decoder = NewLoggingDecoder(decoder)
    
    // Chain unmarshalers: Validation → Audit → Transform → Default
    unmarshaler := mapstructure.NewDefaultUnmarshaler()
    unmarshaler = NewTransformingUnmarshaler(unmarshaler)
    unmarshaler = NewAuditUnmarshaler(unmarshaler)
    unmarshaler = NewValidatingUnmarshaler(unmarshaler)
    
    return schema.NewCodec(metadata, unmarshaler, decoder)
}

// Now you have a fully instrumented, validated, audited codec
func main() {
    codec := NewProductionCodec()
    
    http.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
        var req CreateUserRequest
        
        // All decorators run automatically:
        // 1. Logging starts
        // 2. Metrics recorded
        // 3. Request decoded
        // 4. Data transformed (lowercase)
        // 5. Data audited
        // 6. Data validated
        // 7. Logging ends
        if err := codec.DecodeRequest(r, nil, &req); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        
        w.Write([]byte("OK"))
    })
    
    http.ListenAndServe(":8080", nil)
}
```

## Sharing Metadata Cache

For optimal performance, share the `Metadata` instance across multiple codecs:

```go
package main

import (
    "github.com/talav/schema"
    "github.com/talav/mapstructure"
)

func main() {
    // Create shared metadata instance (expensive to initialize)
    metadata := schema.NewDefaultMetadata()
    
    // Create different codecs for different use cases, sharing the same cache
    standardCodec := schema.NewCodec(
        metadata,
        mapstructure.NewDefaultUnmarshaler(),
        schema.NewDefaultDecoder(),
    )
    
    validatingCodec := schema.NewCodec(
        metadata,
        NewValidatingUnmarshaler(mapstructure.NewDefaultUnmarshaler()),
        schema.NewDefaultDecoder(),
    )
    
    auditCodec := schema.NewCodec(
        metadata,
        NewAuditUnmarshaler(mapstructure.NewDefaultUnmarshaler()),
        schema.NewDefaultDecoder(),
    )
    
    // All codecs benefit from the shared metadata cache
    // Struct metadata is parsed once and reused across all codecs
    
    http.HandleFunc("/public", func(w http.ResponseWriter, r *http.Request) {
        var req PublicRequest
        standardCodec.DecodeRequest(r, nil, &req)
        // ...
    })
    
    http.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
        var req AdminRequest
        validatingCodec.DecodeRequest(r, nil, &req)
        // ...
    })
    
    http.HandleFunc("/audit", func(w http.ResponseWriter, r *http.Request) {
        var req AuditRequest
        auditCodec.DecodeRequest(r, nil, &req)
        // ...
    })
}
```

## Next Steps

- [Custom Tag Parsers Guide](custom-tags.md) - Extend the metadata system with custom struct tags
- [API Documentation](https://pkg.go.dev/github.com/talav/schema) - Full API reference
