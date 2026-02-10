# Core Concepts

Learn the essential concepts to use Schema effectively.

## How It Works

Schema uses a simple three-step process:

1. **Define** - Create a struct with tags describing where data comes from
2. **Decode** - Call `DecodeRequest()` with the HTTP request
3. **Use** - Access the populated struct fields

```go
// 1. Define your request structure
type CreateUserRequest struct {
    Name  string `schema:"name"`           // from query param
    Email string `schema:"email"`          // from query param
    Body  User   `body:"structured"`       // from request body
}

// 2. Decode the request
var req CreateUserRequest
err := codec.DecodeRequest(r, nil, &req)

// 3. Use the data
user := createUser(req.Name, req.Email, req.Body)
```

**That's it.** The codec handles all the HTTP parsing, type conversion, and validation.

## Parameter Locations

Use the `schema` tag to specify where data comes from:

```go
type Request struct {
    // Query parameters (default location)
    Search string `schema:"q"`
    
    // Path parameters (from router)
    UserID string `schema:"user_id,location=path"`
    
    // Headers
    Token string `schema:"Authorization,location=header"`
    
    // Cookies
    SessionID string `schema:"session_id,location=cookie"`
    
    // Skip this field
    Internal string `schema:"-"`
}
```

**Common Options:**

- `location`: Where to read from - `query`, `path`, `header`, `cookie`
- `style`, `explode`: OpenAPI serialization options (see [Serialization Guide](../guides/serialization.md))

## Request Bodies

Use the `body` tag for request body content:

```go
type Request struct {
    // JSON/XML/Form data - automatically detected by Content-Type
    User UserData `body:"structured"`
}

type FileUploadRequest struct {
    // Small files (loaded into memory)
    Avatar []byte `body:"file"`
    
    // Large files (streaming)
    Video io.ReadCloser `body:"file"`
}

type FormRequest struct {
    // Multipart forms with files
    Form SubmissionData `body:"multipart"`
}
```

**When to use each type:**

| Body Type | Use For | Field Type |
|-----------|---------|------------|
| `structured` | REST APIs, JSON, XML, forms | `struct` |
| `file` | Single file uploads | `[]byte` or `io.ReadCloser` |
| `multipart` | Forms with text + files | `struct` with mixed fields |

See [Request Bodies Guide](../guides/request-bodies.md) for details.

## Codec Setup and Performance

**Create the codec once** at application startup and reuse it:

```go
// ✅ GOOD: Create once, reuse everywhere
var codec = schema.NewDefaultCodec()

func handler1(w http.ResponseWriter, r *http.Request) {
    var req Request1
    codec.DecodeRequest(r, nil, &req) // Fast - metadata cached after first use
}

func handler2(w http.ResponseWriter, r *http.Request) {
    var req Request2
    codec.DecodeRequest(r, nil, &req) // Fast - metadata cached after first use
}
```

**Don't create a new codec per request:**

```go
// ❌ BAD: Wastes performance
func handler(w http.ResponseWriter, r *http.Request) {
    codec := schema.NewDefaultCodec() // Creates new cache every time!
    codec.DecodeRequest(r, nil, &req)
}
```

**Why?** The codec caches struct metadata (parsed field tags). Creating it once means:
- First request for a type: ~microseconds (parses tags)
- Subsequent requests: ~nanoseconds (cache lookup)

Creating a new codec every request means you lose the cache benefit.

## Common Patterns

### REST API with Path Parameters

```go
type GetUserRequest struct {
    UserID string `schema:"id,location=path"`
}

// GET /users/:id
var req GetUserRequest
codec.DecodeRequest(r, routerParams, &req)
```

### JSON Request Body

```go
type CreateUserRequest struct {
    User User `body:"structured"`
}

// POST /users with JSON body
var req CreateUserRequest
codec.DecodeRequest(r, nil, &req)
```

### Query Filters with Arrays

```go
type SearchRequest struct {
    Query  string   `schema:"q"`
    Tags   []string `schema:"tags"`  // ?tags=go&tags=http
    Status []string `schema:"status,style=form,explode=false"` // ?status=active,pending
}
```

### Mixed Parameters

```go
type UpdateUserRequest struct {
    UserID string   `schema:"id,location=path"`
    Token  string   `schema:"Authorization,location=header"`
    Body   UserData `body:"structured"`
}

// PUT /users/:id with Authorization header and JSON body
```

## Thread Safety

**The codec is safe to use concurrently** across multiple goroutines and HTTP handlers. Create it once and share it:

```go
// Create once at application startup
var codec = schema.NewDefaultCodec()

// Use in multiple handlers concurrently - completely safe
func handler1(w http.ResponseWriter, r *http.Request) {
    var req Request1
    codec.DecodeRequest(r, nil, &req)
}

func handler2(w http.ResponseWriter, r *http.Request) {
    var req Request2
    codec.DecodeRequest(r, nil, &req)
}
```

## Type Conversion

Schema automatically converts HTTP parameter strings to Go types:

```go
type Request struct {
    Count    int       `schema:"count"`        // "42" → 42
    Price    float64   `schema:"price"`        // "19.99" → 19.99
    Active   bool      `schema:"active"`       // "true" → true
    Tags     []string  `schema:"tags"`         // Multiple values → slice
    Created  time.Time `schema:"created"`      // ISO 8601 → time.Time
    UserID   *string   `schema:"user_id"`      // Optional with pointer
}
```

**Default values** for missing parameters:

```go
type Request struct {
    Page  int    `schema:"page" default:"1"`
    Limit int    `schema:"limit" default:"20"`
    Sort  string `schema:"sort" default:"created_at"`
}
```

See [Type Conversion Guide](../guides/type-conversion.md) for all supported types.

## Error Handling

Errors can occur at different stages:

```go
err := codec.DecodeRequest(r, routerParams, &req)
if err != nil {
    // Could be:
    // - Invalid type conversion (e.g., "abc" → int)
    // - Invalid request body
    // - Body too large
    
    http.Error(w, err.Error(), http.StatusBadRequest)
    return
}
```

## Next Steps

- [Parameters Guide](../guides/parameters.md) - Learn about parameter locations and styles
- [Request Bodies Guide](../guides/request-bodies.md) - Deep dive into body types
- [Serialization Guide](../guides/serialization.md) - Understand OpenAPI styles
