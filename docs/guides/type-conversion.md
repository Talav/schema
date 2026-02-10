# Type Conversion Guide

This guide explains how Schema handles type conversion and the type system.

## Automatic Type Conversion

Schema uses the [mapstructure](https://github.com/talav/mapstructure) library for automatic type conversion. This provides flexible conversion between types with sensible defaults.

## Supported Conversions

### String Conversions

The most common conversion - from string parameters to typed fields:

| Target Type | Accepted Input | Example Conversion |
|-------------|---------------|-------------------|
| `string` | string, int, bool, float | `42` → `"42"` |
| `bool` | bool, int, string (`"true"`, `"false"`, `"1"`, `"0"`) | `"true"` → `true` |
| `int`, `int8`-`int64` | int, uint, float, bool, string | `"42"` → `42` |
| `uint`, `uint8`-`uint64` | int, uint, float, bool, string | `"42"` → `uint(42)` |
| `float32`, `float64` | int, uint, float, bool, string | `"3.14"` → `3.14` |
| `[]byte` | string, []byte | `"hello"` → `[]byte("hello")` |

### Example

```go
type ConfigRequest struct {
    Host     string  `schema:"host"`      // "localhost"
    Port     int     `schema:"port"`      // "8080" → 8080
    Enabled  bool    `schema:"enabled"`   // "true" → true
    Timeout  float64 `schema:"timeout"`   // "30.5" → 30.5
    Replicas uint    `schema:"replicas"`  // "3" → 3
}

// Query: ?host=localhost&port=8080&enabled=true&timeout=30.5&replicas=3
```

Test:
```bash
curl "http://localhost:8080/config?host=localhost&port=8080&enabled=true&timeout=30.5&replicas=3"
```

## Boolean Conversions

Multiple representations of boolean values are supported:

| Input | Result |
|-------|--------|
| `"true"`, `"True"`, `"TRUE"`, `"1"` | `true` |
| `"false"`, `"False"`, `"FALSE"`, `"0"`, `""` | `false` |

```go
type FeatureFlags struct {
    Debug     bool `schema:"debug"`      // ?debug=true
    Profiling bool `schema:"profiling"`  // ?profiling=1
    Logging   bool `schema:"logging"`    // ?logging=false
}

// All of these work:
// ?debug=true
// ?debug=True
// ?debug=TRUE
// ?debug=1

// ?profiling=0
// ?profiling=false
// ?profiling= (empty = false)
```

## Numeric Conversions

### Integer Types

```go
type RangeRequest struct {
    Min    int   `schema:"min"`     // ?min=0
    Max    int   `schema:"max"`     // ?max=100
    Offset int64 `schema:"offset"`  // ?offset=1000
    Limit  uint  `schema:"limit"`   // ?limit=50
}

// Conversions:
// "42" → 42
// "3.14" → 3 (truncates)
// "0" → 0
// "-10" → -10
```

### Floating Point Types

```go
type MathRequest struct {
    Latitude  float64 `schema:"lat"`   // ?lat=40.7128
    Longitude float64 `schema:"lon"`   // ?lon=-74.0060
    Precision float32 `schema:"prec"`  // ?prec=0.001
}

// Conversions:
// "3.14" → 3.14
// "42" → 42.0
// "1e-6" → 0.000001
```

## Collection Types

### Slices and Arrays

```go
type FilterRequest struct {
    // String slice
    Tags []string `schema:"tags"`  // ?tags=go&tags=api
    
    // Integer slice
    IDs []int `schema:"ids"`      // ?ids=1&ids=2&ids=3
    
    // Fixed-size array
    Coords [2]float64 `schema:"coords"` // ?coords=40.7&coords=-74.0
}

// Result:
// req.Tags = ["go", "api"]
// req.IDs = [1, 2, 3]
// req.Coords = [40.7, -74.0]
```

### Maps

```go
type MetadataRequest struct {
    // Map from string to string
    Labels map[string]string `schema:"labels,style=deepObject"`
    
    // Map with typed values
    Counts map[string]int `schema:"counts,style=deepObject"`
}

// Query: ?labels[env]=prod&labels[region]=us&counts[users]=100&counts[posts]=50
// Result:
// req.Labels = {"env": "prod", "region": "us"}
// req.Counts = {"users": 100, "posts": 50}
```

## Struct Types

### Nested Structs

Nested structs are handled automatically:

```go
type Address struct {
    Street  string `schema:"street"`
    City    string `schema:"city"`
    ZipCode string `schema:"zip_code"`
}

type UserRequest struct {
    Name    string  `schema:"name"`
    Address Address `schema:"address,style=deepObject"`
}

// Query: ?name=Alice&address[street]=123 Main St&address[city]=NYC&address[zip_code]=10001
```

**JSON Body Example:**

```go
type CreateUserRequest struct {
    Body struct {
        Name    string `schema:"name"`
        Address struct {
            Street string `schema:"street"`
            City   string `schema:"city"`
        } `schema:"address"`
    } `body:"structured"`
}

// POST with JSON body:
// {
//   "name": "Alice",
//   "address": {
//     "street": "123 Main St",
//     "city": "NYC"
//   }
// }
```

### Embedded Structs

Embedded structs promote their fields to the parent:

```go
type Timestamps struct {
    CreatedAt string `schema:"created_at"`
    UpdatedAt string `schema:"updated_at"`
}

type Article struct {
    Timestamps        // Fields promoted to Article
    Title      string `schema:"title"`
    Content    string `schema:"content"`
}

// JSON body (promoted fields at top level):
// {
//   "title": "My Article",
//   "content": "Article content...",
//   "created_at": "2024-01-01T00:00:00Z",
//   "updated_at": "2024-01-02T00:00:00Z"
// }

// Alternative (named embedded access):
// {
//   "title": "My Article",
//   "Timestamps": {
//     "created_at": "2024-01-01T00:00:00Z",
//     "updated_at": "2024-01-02T00:00:00Z"
//   }
// }
// Both work!
```

## Pointer Types

### Detecting Missing vs Zero Values

Use pointers to distinguish between missing parameters and zero values:

```go
type UpdateRequest struct {
    // Without pointer: 0 could be missing OR explicitly set to 0
    Count int `schema:"count"`
    
    // With pointer: nil = missing, *0 = explicitly set to 0
    Limit *int `schema:"limit"`
    
    // Pointer to bool: nil = missing
    Enabled *bool `schema:"enabled"`
    
    // Pointer to string: nil = missing, *"" = empty string
    Name *string `schema:"name"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    var req UpdateRequest
    codec.DecodeRequest(r, nil, &req)
    
    // Check if Limit was provided
    if req.Limit == nil {
        // Not provided - use default or skip update
        fmt.Println("Limit not provided")
    } else {
        // Provided - use the value (could be 0)
        fmt.Printf("Limit set to %d\n", *req.Limit)
    }
    
    // Check Enabled flag
    if req.Enabled != nil && *req.Enabled {
        fmt.Println("Explicitly enabled")
    }
}
```

**Example Scenarios:**

| Query String | Count | Limit |
|--------------|-------|-------|
| `?count=0&limit=0` | `0` | `*0` |
| `?count=5&limit=10` | `5` | `*10` |
| _(no params)_ | `0` | `nil` |
| `?count=0` | `0` | `nil` |

### Optional Structs

```go
type SearchRequest struct {
    Query   string `schema:"q"`
    
    // Optional filter - nil if not provided
    Filter *struct {
        Status   string `schema:"status"`
        Category string `schema:"category"`
    } `schema:"filter,style=deepObject"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    var req SearchRequest
    codec.DecodeRequest(r, nil, &req)
    
    if req.Filter != nil {
        // Filter was provided
        fmt.Printf("Status: %s, Category: %s\n", 
            req.Filter.Status, req.Filter.Category)
    } else {
        // No filter provided
        fmt.Println("No filter")
    }
}
```

## Default Values

Use the `default` tag to provide fallback values:

```go
type APIRequest struct {
    Format   string `schema:"format" default:"json"`
    PageSize int    `schema:"page_size" default:"20"`
    Timeout  int    `schema:"timeout" default:"30"`
    Debug    bool   `schema:"debug" default:"false"`
}

// Query: (empty)
// Result: Format="json", PageSize=20, Timeout=30, Debug=false

// Query: ?format=xml&page_size=50
// Result: Format="xml", PageSize=50, Timeout=30, Debug=false
```

### Defaults with Pointers

Defaults work with pointers too:

```go
type Request struct {
    Count *int `schema:"count" default:"10"`
}

// Query: (empty)
// Result: req.Count = *10

// Query: ?count=5
// Result: req.Count = *5
```

## Custom Types

### Type Aliases

Type aliases work automatically:

```go
type UserID string
type Count int

type Request struct {
    User  UserID `schema:"user_id"`
    Total Count  `schema:"total"`
}

// Query: ?user_id=user123&total=42
// Result: req.User = UserID("user123"), req.Total = Count(42)
```

### Struct Types

Custom struct types require JSON body or deep object style:

```go
type Coordinate struct {
    Lat float64 `schema:"lat"`
    Lon float64 `schema:"lon"`
}

type LocationRequest struct {
    // Deep object in query
    Location Coordinate `schema:"location,style=deepObject"`
}

// Query: ?location[lat]=40.7128&location[lon]=-74.0060
```

## Time and Date Types

For time/date handling, use strings and parse manually:

```go
import "time"

type EventRequest struct {
    StartDate string `schema:"start_date"` // "2024-01-01"
    EndDate   string `schema:"end_date"`   // "2024-12-31"
}

func handler(w http.ResponseWriter, r *http.Request) {
    var req EventRequest
    codec.DecodeRequest(r, nil, &req)
    
    // Parse dates
    start, err := time.Parse("2006-01-02", req.StartDate)
    if err != nil {
        http.Error(w, "Invalid start_date", http.StatusBadRequest)
        return
    }
    
    end, err := time.Parse("2006-01-02", req.EndDate)
    if err != nil {
        http.Error(w, "Invalid end_date", http.StatusBadRequest)
        return
    }
    
    // Use parsed times
    fmt.Printf("Range: %s to %s\n", start, end)
}
```

!!! tip "Custom Type Conversion"

    For automatic parsing of custom types (like dates, enums, etc.), you can implement Go's `encoding.TextUnmarshaler` interface. See [Extensibility](../advanced/extensibility.md) for details on custom type handling.

## Next Steps

- [Parameters Guide](parameters.md) - Learn about parameter locations
- [Request Bodies Guide](request-bodies.md) - Understand body types
- [mapstructure docs](https://github.com/talav/mapstructure) - Deep dive into conversions
