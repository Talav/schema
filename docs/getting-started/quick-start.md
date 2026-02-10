# Quick Start

## Installation

```bash
go get github.com/talav/schema
```

**Requirements:** Go 1.18 or later (for generics support)

## Basic Example

Create a simple handler that decodes query parameters:

```go
package main

import (
    "fmt"
    "net/http"
    "github.com/talav/schema"
)

// Define your request structure
type HelloRequest struct {
    Name string `schema:"name"`
    Age  int    `schema:"age"`
}

func main() {
    // Create codec once and reuse
    codec := schema.NewDefaultCodec()
    
    http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
        var req HelloRequest
        
        // Decode request into struct
        if err := codec.DecodeRequest(r, nil, &req); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        
        // Use the decoded data
        fmt.Fprintf(w, "Hello %s, you are %d years old!", req.Name, req.Age)
    })
    
    http.ListenAndServe(":8080", nil)
}
```

Test it:

```bash
curl "http://localhost:8080/hello?name=Alice&age=30"
# Output: Hello Alice, you are 30 years old!
```

## With Path Parameters

Most routers provide path parameters. Here's how to use them:

### Using Chi Router

```go
package main

import (
    "fmt"
    "net/http"
    "github.com/go-chi/chi/v5"
    "github.com/talav/schema"
)

type GetUserRequest struct {
    UserID string `schema:"user_id,location=path"`
    Format string `schema:"format,location=query"`
}

func main() {
    codec := schema.NewDefaultCodec()
    r := chi.NewRouter()
    
    r.Get("/users/{user_id}", func(w http.ResponseWriter, r *http.Request) {
        // Get path params from Chi
        routerParams := map[string]string{
            "user_id": chi.URLParam(r, "user_id"),
        }
        
        var req GetUserRequest
        if err := codec.DecodeRequest(r, routerParams, &req); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        
        fmt.Fprintf(w, "User ID: %s, Format: %s", req.UserID, req.Format)
    })
    
    http.ListenAndServe(":8080", r)
}
```

Test it:

```bash
curl "http://localhost:8080/users/123?format=json"
# Output: User ID: 123, Format: json
```

### Using Gorilla Mux

```go
package main

import (
    "fmt"
    "net/http"
    "github.com/gorilla/mux"
    "github.com/talav/schema"
)

type GetUserRequest struct {
    UserID string `schema:"user_id,location=path"`
}

func main() {
    codec := schema.NewDefaultCodec()
    r := mux.NewRouter()
    
    r.HandleFunc("/users/{user_id}", func(w http.ResponseWriter, r *http.Request) {
        var req GetUserRequest
        
        // mux.Vars returns all path params as a map
        if err := codec.DecodeRequest(r, mux.Vars(r), &req); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        
        fmt.Fprintf(w, "User ID: %s", req.UserID)
    }).Methods("GET")
    
    http.ListenAndServe(":8080", r)
}
```

### Using Standard Library

For stdlib's `http.ServeMux` (Go 1.22+):

```go
package main

import (
    "fmt"
    "net/http"
    "github.com/talav/schema"
)

type GetUserRequest struct {
    UserID string `schema:"user_id,location=path"`
}

func main() {
    codec := schema.NewDefaultCodec()
    
    http.HandleFunc("GET /users/{user_id}", func(w http.ResponseWriter, r *http.Request) {
        routerParams := map[string]string{
            "user_id": r.PathValue("user_id"),
        }
        
        var req GetUserRequest
        if err := codec.DecodeRequest(r, routerParams, &req); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        
        fmt.Fprintf(w, "User ID: %s", req.UserID)
    })
    
    http.ListenAndServe(":8080", nil)
}
```

## With Request Body

Decode JSON request bodies:

```go
package main

import (
    "encoding/json"
    "net/http"
    "github.com/talav/schema"
)

type CreateUserRequest struct {
    // Query parameter
    Version string `schema:"version,location=query"`
    
    // Request body
    Body struct {
        Name  string `schema:"name"`
        Email string `schema:"email"`
        Age   int    `schema:"age"`
    } `body:"structured"`
}

func main() {
    codec := schema.NewDefaultCodec()
    
    http.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
        var req CreateUserRequest
        
        if err := codec.DecodeRequest(r, nil, &req); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        
        // Use req.Version, req.Body.Name, req.Body.Email, req.Body.Age
        response := map[string]interface{}{
            "message": "User created",
            "name":    req.Body.Name,
            "email":   req.Body.Email,
        }
        
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
    })
    
    http.ListenAndServe(":8080", nil)
}
```

Test it:

```bash
curl -X POST "http://localhost:8080/users?version=v2" \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice@example.com","age":30}'
```

## Mixed Parameters

Combine query, path, header, and body parameters:

```go
type CompleteRequest struct {
    // Path parameter
    OrgID string `schema:"org_id,location=path"`
    
    // Query parameters
    Page     int    `schema:"page,location=query"`
    PageSize int    `schema:"page_size,location=query"`
    
    // Header parameter
    APIKey string `schema:"X-API-Key,location=header"`
    
    // Cookie parameter
    SessionID string `schema:"session_id,location=cookie"`
    
    // Request body
    Body struct {
        Name        string `schema:"name"`
        Description string `schema:"description"`
    } `body:"structured"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    codec := schema.NewDefaultCodec()
    routerParams := map[string]string{"org_id": "123"}
    
    var req CompleteRequest
    if err := codec.DecodeRequest(r, routerParams, &req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // All fields are now populated from their respective sources
}
```

## Error Handling

Always check for decoding errors:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    var req MyRequest
    
    if err := codec.DecodeRequest(r, nil, &req); err != nil {
        // Log the error for debugging
        log.Printf("Failed to decode request: %v", err)
        
        // Return appropriate HTTP status
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    // Proceed with business logic
}
```

## Best Practices

### Pointers for Optional Fields

Distinguish between missing and zero values:

```go
type Request struct {
    Count *int `schema:"count"` // nil if missing, *0 if count=0
}
```

## Next Steps

- [Core Concepts](concepts.md) - Understand the architecture
- [Parameters Guide](../guides/parameters.md) - Deep dive into parameter handling
- [Request Bodies Guide](../guides/request-bodies.md) - Learn about body types
- [Type Conversion Guide](../guides/type-conversion.md) - Understand type handling
