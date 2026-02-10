# Serialization Styles Guide

This guide explains OpenAPI 3.0/3.1 parameter serialization styles and how to use them in Schema.

## Overview

Schema implements the [OpenAPI 3.1 Specification](https://spec.openapis.org/oas/v3.1.0#parameter-serialization) for parameter serialization. This ensures consistent behavior across OpenAPI-compliant tools and frameworks.

## Serialization Concepts

### Style

The **style** defines how arrays and objects are serialized in parameters.

### Explode

The **explode** parameter controls whether arrays and objects generate multiple parameter instances:

- **`explode=true`**: Each array element or object property gets its own parameter
- **`explode=false`**: Values are combined into a single parameter

## Style by Location

Different parameter locations support different styles:

| Location | Allowed Styles | Default Style | Default Explode |
|----------|---------------|---------------|-----------------|
| `query` | form, spaceDelimited, pipeDelimited, deepObject | form | true |
| `path` | simple, label, matrix | simple | false |
| `header` | simple | simple | false |
| `cookie` | form | form | true |

## Query Parameter Styles

### Form Style (Default)

The most common style for query parameters.

#### Form with explode=true (default)

```go
type SearchRequest struct {
    // Array: ?tags=go&tags=api&tags=rest
    Tags []string `schema:"tags,location=query,style=form,explode=true"`
    
    // Object: ?color=blue&size=large
    Filter struct {
        Color string `schema:"color"`
        Size  string `schema:"size"`
    } `schema:",location=query,style=form,explode=true"`
}
```

**Examples:**

| Type | Input | URL |
|------|-------|-----|
| Array | `["go", "api", "rest"]` | `?tags=go&tags=api&tags=rest` |
| Object | `{color: "blue", size: "large"}` | `?color=blue&size=large` |

#### Form with explode=false

```go
type SearchRequest struct {
    // Array: ?tags=go,api,rest
    Tags []string `schema:"tags,location=query,style=form,explode=false"`
    
    // Object: ?filter=color,blue,size,large
    Filter struct {
        Color string `schema:"color"`
        Size  string `schema:"size"`
    } `schema:"filter,location=query,style=form,explode=false"`
}
```

**Examples:**

| Type | Input | URL |
|------|-------|-----|
| Array | `["go", "api", "rest"]` | `?tags=go,api,rest` |
| Object | `{color: "blue", size: "large"}` | `?filter=color,blue,size,large` |

### Space Delimited Style

Values separated by spaces (URL-encoded as `%20`).

```go
type SearchRequest struct {
    // ?tags=go%20api%20rest (spaces are %20 encoded)
    Tags []string `schema:"tags,location=query,style=spaceDelimited"`
}
```

**Example:**

```bash
curl "http://localhost:8080/search?tags=go%20api%20rest"
# Decoded: tags=go api rest
# Result: req.Tags = ["go", "api", "rest"]
```

**Note:** Explode is not applicable to spaceDelimited style.

### Pipe Delimited Style

Values separated by pipe characters (`|`).

```go
type FilterRequest struct {
    // ?status=active|pending|completed
    Statuses []string `schema:"status,location=query,style=pipeDelimited"`
}
```

**Example:**

```bash
curl "http://localhost:8080/filter?status=active|pending|completed"
# Result: req.Statuses = ["active", "pending", "completed"]
```

**Note:** Explode is not applicable to pipeDelimited style.

### Deep Object Style

For complex nested objects using bracket notation.

```go
type SearchRequest struct {
    // ?filter[status]=active&filter[type]=user&filter[role]=admin
    Filter struct {
        Status string `schema:"status"`
        Type   string `schema:"type"`
        Role   string `schema:"role"`
    } `schema:"filter,location=query,style=deepObject"`
}
```

**Example:**

```bash
curl "http://localhost:8080/search?filter[status]=active&filter[type]=user&filter[role]=admin"
# Result:
# req.Filter.Status = "active"
# req.Filter.Type = "user"
# req.Filter.Role = "admin"
```

**Nested Deep Objects:**

```go
type SearchRequest struct {
    // ?user[profile][name]=Alice&user[profile][age]=30
    User struct {
        Profile struct {
            Name string `schema:"name"`
            Age  int    `schema:"age"`
        } `schema:"profile"`
    } `schema:"user,location=query,style=deepObject"`
}
```

**Note:** Deep object style always uses explode=true semantics.

## Path Parameter Styles

### Simple Style (Default)

Comma-separated values (most common for path parameters).

```go
type GetUserRequest struct {
    // /users/123
    UserID string `schema:"user_id,location=path,style=simple"`
    
    // Array: /resources/1,2,3
    IDs []int `schema:"ids,location=path,style=simple,explode=false"`
}
```

**Examples:**

| Type | Input | Path |
|------|-------|------|
| Scalar | `"123"` | `/users/123` |
| Array (explode=false) | `[1, 2, 3]` | `/resources/1,2,3` |
| Array (explode=true) | `[1, 2, 3]` | `/resources/1,2,3` |

### Label Style

Period-prefixed values.

```go
type GetResourceRequest struct {
    // Scalar: /resources/.123
    ID string `schema:"id,location=path,style=label"`
    
    // Array (explode=false): /resources/.1.2.3
    IDs []int `schema:"ids,location=path,style=label,explode=false"`
    
    // Array (explode=true): /resources/.1.2.3
    IDsExploded []int `schema:"ids_exploded,location=path,style=label,explode=true"`
}
```

**Examples:**

| Type | Input | Path |
|------|-------|------|
| Scalar | `"123"` | `/resources/.123` |
| Array (explode=false) | `[1, 2, 3]` | `/resources/.1.2.3` |
| Array (explode=true) | `[1, 2, 3]` | `/resources/.1.2.3` |

### Matrix Style

Semicolon-prefixed key-value pairs.

```go
type GetResourceRequest struct {
    // Scalar: /resources;id=123
    ID string `schema:"id,location=path,style=matrix"`
    
    // Array (explode=false): /resources;ids=1,2,3
    IDs []int `schema:"ids,location=path,style=matrix,explode=false"`
    
    // Array (explode=true): /resources;ids=1;ids=2;ids=3
    IDsExploded []int `schema:"ids_exploded,location=path,style=matrix,explode=true"`
}
```

**Examples:**

| Type | Input | Path |
|------|-------|------|
| Scalar | `"123"` | `/resources;id=123` |
| Array (explode=false) | `[1, 2, 3]` | `/resources;ids=1,2,3` |
| Array (explode=true) | `[1, 2, 3]` | `/resources;ids=1;ids=2;ids=3` |

## Header Parameter Style

Headers only support the **simple** style.

```go
type AuthRequest struct {
    // Header: X-User-IDs: 1,2,3
    UserIDs []int `schema:"X-User-IDs,location=header"`
}
```

**Example:**

```bash
curl -H "X-User-IDs: 1,2,3" http://localhost:8080/api
# Result: req.UserIDs = [1, 2, 3]
```

## Cookie Parameter Style

Cookies only support the **form** style with explode=true.

```go
type PreferencesRequest struct {
    // Cookie: preferences=theme,dark,lang,en
    Preferences map[string]string `schema:"preferences,location=cookie"`
}
```

**Example:**

```bash
curl -H "Cookie: preferences=theme,dark,lang,en" http://localhost:8080/api
```

## Complete Examples

### REST API with Multiple Styles

```go
type ComplexSearchRequest struct {
    // Query: form style (default)
    Query string `schema:"q,location=query"`
    
    // Query: array with explode=true (default)
    Tags []string `schema:"tags,location=query"`
    
    // Query: deep object for filtering
    Filter struct {
        Status   string `schema:"status"`
        Category string `schema:"category"`
        MinPrice float64 `schema:"min_price"`
        MaxPrice float64 `schema:"max_price"`
    } `schema:"filter,location=query,style=deepObject"`
    
    // Query: pipe delimited categories
    Categories []string `schema:"categories,location=query,style=pipeDelimited"`
    
    // Path: simple style
    ProjectID string `schema:"project_id,location=path"`
    
    // Header: API key
    APIKey string `schema:"X-API-Key,location=header"`
}
```

**Usage:**

```bash
# Route: /projects/{project_id}/search
curl -H "X-API-Key: secret123" \
  "http://localhost:8080/projects/proj1/search?\
q=golang&\
tags=backend&tags=api&\
filter[status]=active&\
filter[category]=tools&\
filter[min_price]=0&\
filter[max_price]=100&\
categories=dev|ops|testing"
```

### Matrix Style Path Parameters

```go
type ResourceRequest struct {
    ResourceID string `schema:"resource_id,location=path,style=matrix"`
    Filters    []string `schema:"filters,location=path,style=matrix,explode=true"`
}

// Route: /resources{resource_id}{filters}
// Example: /resources;resource_id=123;filters=active;filters=verified
```

## Style Compatibility Matrix

Reference for which styles work with which locations:

| Style | Query | Path | Header | Cookie | Explode Support |
|-------|-------|------|--------|--------|-----------------|
| form | ✅ | ❌ | ❌ | ✅ | ✅ |
| simple | ❌ | ✅ | ✅ | ❌ | ✅ (path only) |
| matrix | ❌ | ✅ | ❌ | ❌ | ✅ |
| label | ❌ | ✅ | ❌ | ❌ | ✅ |
| spaceDelimited | ✅ | ❌ | ❌ | ❌ | ❌ |
| pipeDelimited | ✅ | ❌ | ❌ | ❌ | ❌ |
| deepObject | ✅ | ❌ | ❌ | ❌ | N/A (always exploded) |


## OpenAPI Specification Compliance

Schema implements OpenAPI 3.0/3.1 parameter serialization as specified in:

- [OpenAPI 3.0.3 - Parameter Object](https://spec.openapis.org/oas/v3.0.3#parameter-object)
- [OpenAPI 3.1.0 - Parameter Serialization](https://spec.openapis.org/oas/v3.1.0#parameter-serialization)

This ensures that your API behaves consistently with OpenAPI tools like:

- Swagger UI
- Redoc
- OpenAPI Generator
- API clients generated from OpenAPI specs

## Next Steps

- [Parameters Guide](parameters.md) - Learn about parameter locations
- [Type Conversion Guide](type-conversion.md) - Understand type handling
- [OpenAPI 3.1 Spec](https://spec.openapis.org/oas/v3.1.0) - Official specification
