# Parameters Guide

This guide covers all parameter locations and how to work with them in Schema.

## Parameter Locations

Schema supports all four parameter locations defined in the OpenAPI specification:

| Location | Description | Source in HTTP Request | Default Style |
|----------|-------------|------------------------|---------------|
| `query` | Query string parameters | `?name=value` | `form` |
| `path` | Path parameters | `/users/{id}` | `simple` |
| `header` | HTTP headers | `X-API-Key: value` | `simple` |
| `cookie` | Cookie values | `Cookie: session=abc` | `form` |

## Query Parameters

Query parameters are the most common type. By default, fields without a location are treated as query parameters.

### Basic Query Parameters

```go
type SearchRequest struct {
    Query string `schema:"q"`        // ?q=golang
    Page  int    `schema:"page"`     // ?page=2
    Limit int    `schema:"limit"`    // ?limit=20
}

func handler(w http.ResponseWriter, r *http.Request) {
    var req SearchRequest
    codec.DecodeRequest(r, nil, &req)
    
    // Use req.Query, req.Page, req.Limit
}
```

### Query Parameters with Default Values

Use the `default` tag to provide fallback values:

```go
type APIRequest struct {
    Format  string `schema:"format" default:"json"`
    Timeout int    `schema:"timeout" default:"30"`
    Debug   bool   `schema:"debug" default:"false"`
}

// GET /api
// Result: Format="json", Timeout=30, Debug=false

// GET /api?format=xml&timeout=60
// Result: Format="xml", Timeout=60, Debug=false
```

### Array Query Parameters

Handle multiple values for the same parameter:

```go
type FilterRequest struct {
    // ?tags=go&tags=api&tags=rest
    Tags []string `schema:"tags"`
    
    // ?ids=1&ids=2&ids=3
    IDs []int `schema:"ids"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    var req FilterRequest
    codec.DecodeRequest(r, nil, &req)
    
    // req.Tags = ["go", "api", "rest"]
    // req.IDs = [1, 2, 3]
}
```

### Object Query Parameters

Use structs for complex query parameters:

```go
type SearchRequest struct {
    // Deep object style: ?filter[status]=active&filter[type]=user
    Filter struct {
        Status string `schema:"status"`
        Type   string `schema:"type"`
    } `schema:"filter,location=query,style=deepObject"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    var req SearchRequest
    codec.DecodeRequest(r, nil, &req)
    
    // req.Filter.Status = "active"
    // req.Filter.Type = "user"
}
```

### Optional Query Parameters

All query parameters are optional by default. Missing parameters result in zero values:

```go
type SearchRequest struct {
    Query  string `schema:"q"`       // "" if missing
    Page   int    `schema:"page"`    // 0 if missing
    Active *bool  `schema:"active"`  // nil if missing (pointer for explicit absence)
}

// GET /search
// Result: Query="", Page=0, Active=nil

// GET /search?q=golang&page=2
// Result: Query="golang", Page=2, Active=nil
```

## Path Parameters

Path parameters come from your router.

### Basic Path Parameters

```go
type GetUserRequest struct {
    UserID string `schema:"user_id,location=path"`
}

// Using Chi router
r.Get("/users/{user_id}", func(w http.ResponseWriter, r *http.Request) {
    routerParams := map[string]string{
        "user_id": chi.URLParam(r, "user_id"),
    }
    
    var req GetUserRequest
    codec.DecodeRequest(r, routerParams, &req)
    
    // req.UserID = "123"
})
```

### Multiple Path Parameters

```go
type GetResourceRequest struct {
    OrgID      string `schema:"org_id,location=path"`
    ProjectID  string `schema:"project_id,location=path"`
    ResourceID string `schema:"resource_id,location=path"`
}

// Route: /orgs/{org_id}/projects/{project_id}/resources/{resource_id}
r.Get("/orgs/{org_id}/projects/{project_id}/resources/{resource_id}", 
    func(w http.ResponseWriter, r *http.Request) {
        routerParams := map[string]string{
            "org_id":      chi.URLParam(r, "org_id"),
            "project_id":  chi.URLParam(r, "project_id"),
            "resource_id": chi.URLParam(r, "resource_id"),
        }
        
        var req GetResourceRequest
        codec.DecodeRequest(r, routerParams, &req)
        
        // req.OrgID = "acme"
        // req.ProjectID = "website"
        // req.ResourceID = "42"
    })
```

### Path Parameters with Type Conversion

Path parameters are automatically converted to the target type:

```go
type GetItemRequest struct {
    ItemID int    `schema:"item_id,location=path"`  // Converts string to int
    Slug   string `schema:"slug,location=path"`
}

// Route: /items/{item_id}/slug/{slug}
// GET /items/123/slug/my-item
// Result: ItemID=123 (int), Slug="my-item" (string)
```

## Header Parameters

Headers are case-insensitive in HTTP but should be specified in canonical form.

### Basic Headers

```go
type AuthRequest struct {
    APIKey      string `schema:"X-API-Key,location=header"`
    ContentType string `schema:"Content-Type,location=header"`
    UserAgent   string `schema:"User-Agent,location=header"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    var req AuthRequest
    codec.DecodeRequest(r, nil, &req)
    
    // Headers are case-insensitive
    // "X-API-Key", "x-api-key", "X-Api-Key" all work
}
```

### Custom Headers

```go
type RequestWithHeaders struct {
    RequestID   string `schema:"X-Request-ID,location=header"`
    TraceID     string `schema:"X-Trace-ID,location=header"`
    Correlation string `schema:"X-Correlation-ID,location=header"`
}

// All standard and custom headers are supported
```

### Header Arrays

Multiple header values are supported:

```go
type HeaderArrayRequest struct {
    // Header: Accept: application/json, text/html
    Accept []string `schema:"Accept,location=header"`
}

// req.Accept = ["application/json", "text/html"]
```

## Cookie Parameters

Extract values from cookies:

### Basic Cookies

```go
type SessionRequest struct {
    SessionID string `schema:"session_id,location=cookie"`
    UserLang  string `schema:"user_lang,location=cookie"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    var req SessionRequest
    codec.DecodeRequest(r, nil, &req)
    
    // req.SessionID = "abc123"
    // req.UserLang = "en"
}
```

### Cookie with Default Values

```go
type PreferencesRequest struct {
    Theme    string `schema:"theme,location=cookie" default:"light"`
    Timezone string `schema:"timezone,location=cookie" default:"UTC"`
}

// If cookies not present, defaults are used
```

## Mixed Parameters

Combine parameters from multiple locations:

```go
type CompleteRequest struct {
    // Path parameter
    UserID string `schema:"user_id,location=path"`
    
    // Query parameters
    Format string `schema:"format,location=query"`
    Page   int    `schema:"page,location=query"`
    
    // Header parameters
    APIKey string `schema:"X-API-Key,location=header"`
    
    // Cookie parameters
    SessionID string `schema:"session_id,location=cookie"`
}

// Route: /users/{user_id}
r.Get("/users/{user_id}", func(w http.ResponseWriter, r *http.Request) {
    routerParams := map[string]string{
        "user_id": chi.URLParam(r, "user_id"),
    }
    
    var req CompleteRequest
    codec.DecodeRequest(r, routerParams, &req)
    
    // All parameters are decoded from their respective sources
})
```
## Parameter Naming

### Field Name vs Parameter Name

The schema tag value determines the parameter name:

```go
type Request struct {
    // Parameter name: "user_id"
    UserID string `schema:"user_id"`
    
    // Parameter name: "api-key" (matches HTTP conventions)
    APIKey string `schema:"api-key,location=header"`
    
    // Parameter name: same as field name "Name" (no tag value)
    Name string `schema:",location=query"`
}
```

### Skipping Fields

Use `-` to skip fields:

```go
type Request struct {
    Username string `schema:"username"`
    Password string `schema:"password"`
    
    // Not decoded from request even if present
    Internal string `schema:"-"`
    Computed int    `schema:"-"`
}
```

## Next Steps

- [Request Bodies Guide](request-bodies.md) - Learn about body decoding
- [Serialization Guide](serialization.md) - Understand OpenAPI styles
- [Type Conversion Guide](type-conversion.md) - Type handling details
