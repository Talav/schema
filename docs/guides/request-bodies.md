# Request Bodies Guide

This guide covers all request body types and how to decode them with Schema.

## Body Types Overview

Schema supports three main body types, each optimized for different use cases:

| Body Type | Use Case | Content-Type | Field Type |
|-----------|----------|--------------|------------|
| `structured` | JSON, XML, Forms | `application/json`, `application/xml`, `application/x-www-form-urlencoded` | `struct`, `map` |
| `file` | Single file upload | Any (typically `application/octet-stream`) | `[]byte`, `io.ReadCloser` |
| `multipart` | Form with files | `multipart/form-data` | `struct` with mixed fields |

## Structured Bodies

Use `body:"structured"` for JSON, XML, or form-encoded data.

### JSON Bodies

The most common use case - REST APIs with JSON:

```go
type CreateUserRequest struct {
    Body struct {
        Name     string   `schema:"name"`
        Email    string   `schema:"email"`
        Age      int      `schema:"age"`
        Tags     []string `schema:"tags"`
        Settings map[string]any `schema:"settings"`
    } `body:"structured"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    codec.DecodeRequest(r, nil, &req)
    
    // req.Body.Name = "Alice"
    // req.Body.Email = "alice@example.com"
    // req.Body.Age = 30
}
```


### Nested JSON Structures

Handle complex nested objects:

```go
type CreateOrderRequest struct {
    Body struct {
        Customer struct {
            Name  string `schema:"name"`
            Email string `schema:"email"`
            Address struct {
                Street  string `schema:"street"`
                City    string `schema:"city"`
                Country string `schema:"country"`
            } `schema:"address"`
        } `schema:"customer"`
        
        Items []struct {
            ProductID string `schema:"product_id"`
            Quantity  int    `schema:"quantity"`
            Price     float64 `schema:"price"`
        } `schema:"items"`
        
        Total float64 `schema:"total"`
    } `body:"structured"`
}
```


### XML Bodies

Full support for XML with struct tags:

```go
type ImportDataRequest struct {
    Body struct {
        Name  string `xml:"name"`
        Email string `xml:"email"`
        Items []struct {
            ID   string `xml:"id,attr"`  // Attribute
            Name string `xml:"name"`     // Element
        } `xml:"item"`
    } `body:"structured"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    var req ImportDataRequest
    codec.DecodeRequest(r, nil, &req)
    
    // XML is unmarshaled using Go's encoding/xml
}
```


**XML Notes:**

- Use `xml` struct tags to define XML element/attribute names
- Target field must be a struct, slice, or string
- `map[string]any` is not supported for XML
- Full support for Go's `encoding/xml` features (attributes, CDATA, etc.)

### URL-Encoded Form Bodies

Handle traditional HTML form submissions:

```go
type LoginRequest struct {
    Body struct {
        Username string `schema:"username"`
        Password string `schema:"password"`
        Remember bool   `schema:"remember"`
    } `body:"structured"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    var req LoginRequest
    codec.DecodeRequest(r, nil, &req)
    
    // req.Body.Username = "alice"
    // req.Body.Password = "secret"
    // req.Body.Remember = true
}
```


### Content-Type Auto-Detection

Schema automatically detects the content type:

```go
type FlexibleRequest struct {
    Body UserData `body:"structured"`
}

// Works with:
// - Content-Type: application/json → JSON decoding
// - Content-Type: application/xml → XML decoding
// - Content-Type: application/x-www-form-urlencoded → Form decoding
// - No Content-Type → Defaults to JSON
```

## File Uploads

Use `body:"file"` for single file uploads.

### Small Files (In Memory)

Load entire file into memory as bytes:

```go
type UploadAvatarRequest struct {
    // Query parameter for metadata
    Filename string `schema:"filename,location=query"`
    
    // File content as bytes
    Body []byte `body:"file"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    var req UploadAvatarRequest
    codec.DecodeRequest(r, nil, &req)
    
    // req.Body contains entire file
    fmt.Printf("Uploaded %s (%d bytes)\n", req.Filename, len(req.Body))
    
    // Save to disk
    os.WriteFile("/uploads/" + req.Filename, req.Body, 0644)
}
```


**⚠️ Warning:** Only use `[]byte` for small files (< 10MB). Large files will consume excessive memory.

### Large Files (Memory Loaded)

For `[]byte` fields, files are read entirely into memory:

```go
type UploadImageRequest struct {
    Filename string `schema:"filename,location=query"`
    Body     []byte `body:"file"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    var req UploadImageRequest
    codec.DecodeRequest(r, nil, &req)
    
    // Write to disk
    err := os.WriteFile("/uploads/"+req.Filename, req.Body, 0644)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    fmt.Fprintf(w, "Uploaded %d bytes\n", len(req.Body))
}
```

### Large Files (Streaming)

For `io.ReadCloser` fields, the request body is streamed without loading into memory:

```go
type UploadVideoRequest struct {
    Filename string        `schema:"filename,location=query"`
    Body     io.ReadCloser `body:"file"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    var req UploadVideoRequest
    codec.DecodeRequest(r, nil, &req)
    defer req.Body.Close()
    
    // Stream to disk
    file, err := os.Create("/uploads/" + req.Filename)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer file.Close()
    
    // Copy streams data chunk by chunk
    written, err := io.Copy(file, req.Body)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    fmt.Fprintf(w, "Uploaded %d bytes\n", written)
}
```

### File Upload with Metadata

Combine file upload with other parameters:

```go
type DocumentUploadRequest struct {
    // Path parameter
    ProjectID string `schema:"project_id,location=path"`
    
    // Query parameters
    Title       string `schema:"title,location=query"`
    Description string `schema:"description,location=query"`
    
    // Header
    APIKey string `schema:"X-API-Key,location=header"`
    
    // File content
    Body []byte `body:"file"`
}

// Route: /projects/{project_id}/documents
r.Post("/projects/{project_id}/documents", handler)
```


## Multipart Forms

Use `body:"multipart"` for forms with both text fields and file uploads.

### Basic Multipart Form

```go
type UploadFormRequest struct {
    Body struct {
        // Text fields
        Title       string `schema:"title"`
        Description string `schema:"description"`
        Category    string `schema:"category"`
        
        // Single file
        Document io.ReadCloser `schema:"document"`
    } `body:"multipart"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    var req UploadFormRequest
    codec.DecodeRequest(r, nil, &req)
    defer req.Body.Document.Close()
    
    // req.Body.Title = "My Document"
    // req.Body.Description = "Important file"
    // req.Body.Document = (file content stream)
    
    // Save file
    file, _ := os.Create("/uploads/" + req.Body.Title)
    defer file.Close()
    io.Copy(file, req.Body.Document)
}
```


### Multiple File Uploads

Handle multiple files in a single request:

```go
type GalleryUploadRequest struct {
    Body struct {
        // Text fields
        AlbumName   string `schema:"album_name"`
        Description string `schema:"description"`
        
        // Multiple files
        Images []io.ReadCloser `schema:"images"`
    } `body:"multipart"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    var req GalleryUploadRequest
    codec.DecodeRequest(r, nil, &req)
    
    // Close all files when done
    defer func() {
        for _, img := range req.Body.Images {
            img.Close()
        }
    }()
    
    fmt.Printf("Album: %s\n", req.Body.AlbumName)
    fmt.Printf("Received %d images\n", len(req.Body.Images))
    
    // Process each file
    for i, img := range req.Body.Images {
        filename := fmt.Sprintf("/uploads/%s_%d.jpg", req.Body.AlbumName, i)
        file, _ := os.Create(filename)
        io.Copy(file, img)
        file.Close()
    }
}
```


### Optional Files

Use pointers for optional file uploads:

```go
type ProfileUpdateRequest struct {
    Body struct {
        Name  string `schema:"name"`
        Email string `schema:"email"`
        
        // Optional avatar (nil if not uploaded)
        Avatar *io.ReadCloser `schema:"avatar"`
    } `body:"multipart"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    var req ProfileUpdateRequest
    codec.DecodeRequest(r, nil, &req)
    
    if req.Body.Avatar != nil {
        defer (*req.Body.Avatar).Close()
        // Process avatar upload
    } else {
        // No avatar provided - keep existing
    }
}
```

## Body Type Selection Guide

Choose the right body type for your use case:

### Decision Matrix

| Scenario | Body Type | Field Type | Example Use Case |
|----------|-----------|------------|------------------|
| REST API with JSON | `structured` | `struct` | User CRUD operations |
| XML web service | `structured` | `struct` with `xml` tags | Legacy system integration |
| HTML form submission | `structured` | `struct` | Login, contact forms |
| Profile picture upload | `file` | `[]byte` | Small image < 10MB |
| Video upload | `file` | `io.ReadCloser` | Large media files (streaming) |
| Document with metadata | `multipart` | `struct` with mixed fields | PDF with title/description |
| Photo gallery upload | `multipart` | `struct` with `[]io.ReadCloser` | Multiple images + album info |

## Next Steps

- [Type Conversion Guide](type-conversion.md) - Understand type handling
- [Serialization Guide](serialization.md) - Learn about OpenAPI styles
