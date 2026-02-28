# Custom Tag Parsers Guide

This guide covers how to create custom struct tag parsers to extend Schema's metadata system.

## Understanding Tag Parsers

Tag parsers convert struct tags into typed metadata that can be used during request decoding.


## Tag Parser Function

### Function Signature

```go
type TagParserFunc func(field reflect.StructField, index int, tagValue string) (any, error)
```

**Parameters:**
- `field`: The struct field being parsed
- `index`: Field index in the struct
- `tagValue`: The tag value (e.g., `"name,location=query"`)

**Returns:**
- `any`: Parsed metadata (typically a pointer to a struct)
- `error`: Parse error if tag is invalid

### Example: Simple Tag Parser

```go
package main

import (
    "fmt"
    "reflect"
    "strings"
    
    "github.com/talav/schema"
    "github.com/talav/mapstructure"
)

// Define metadata structure
type CacheMetadata struct {
    TTL      int  // Time to live in seconds
    Enabled  bool
}

// Parse cache tag
func ParseCacheTag(field reflect.StructField, index int, tagValue string) (any, error) {
    meta := &CacheMetadata{
        TTL:     300, // Default 5 minutes
        Enabled: true,
    }
    
    // Parse tag options
    parts := strings.Split(tagValue, ",")
    for _, part := range parts {
        kv := strings.SplitN(part, "=", 2)
        if len(kv) != 2 {
            continue
        }
        
        key, value := kv[0], kv[1]
        switch key {
        case "ttl":
            fmt.Sscanf(value, "%d", &meta.TTL)
        case "enabled":
            meta.Enabled = value == "true"
        }
    }
    
    return meta, nil
}

// Register the parser
func main() {
    registry := schema.NewTagParserRegistry(
        schema.WithTagParser("schema", schema.ParseSchemaTag, schema.DefaultSchemaMetadata),
        schema.WithTagParser("body", schema.ParseBodyTag),
        schema.WithTagParser("cache", ParseCacheTag), // Custom tag
    )
    
    metadata := schema.NewMetadata(registry)
    decoder := schema.NewDefaultDecoder()
    unmarshaler := mapstructure.NewDefaultUnmarshaler()
    
    codec := schema.NewCodec(metadata, unmarshaler, decoder)
    
    // Use codec normally
    _ = codec
}

// Use the custom tag
type UserRequest struct {
    Username string `schema:"username" cache:"ttl=600,enabled=true"`
    Email    string `schema:"email" cache:"ttl=300"`
}
```

## Default Metadata Function

Provide default metadata for fields without the tag:

```go
type DefaultMetadataFunc func(field reflect.StructField, index int) any
```

### Example with Default

```go
// Default cache metadata
func DefaultCacheMetadata(field reflect.StructField, index int) any {
    return &CacheMetadata{
        TTL:     300,
        Enabled: true,
    }
}

// Register with default
registry := schema.NewTagParserRegistry(
    schema.WithTagParser("cache", ParseCacheTag, DefaultCacheMetadata),
)

// Now fields without cache tag get default metadata
type Request struct {
    Username string `schema:"username"` // Gets default cache metadata
    Email    string `schema:"email" cache:"ttl=600"` // Uses custom
}
```

## Accessing Custom Metadata

Use `GetTagMetadata` to access parsed metadata:

```go
// Type-safe access to custom metadata
func getCacheMetadata(field *schema.FieldMetadata) (*CacheMetadata, bool) {
    return schema.GetTagMetadata[*CacheMetadata](field, "cache")
}

// Usage in custom decoder
func (d *CustomDecoder) Decode(...) (map[string]any, error) {
    for _, field := range metadata.Fields {
        if cacheMeta, ok := getCacheMetadata(&field); ok {
            log.Printf("Field %s: TTL=%d, Enabled=%v", 
                field.StructFieldName, cacheMeta.TTL, cacheMeta.Enabled)
        }
    }
    
    // Continue decoding...
}
```

## Complete Examples

### Example 1: Validation Tag

```go
package main

import (
    "fmt"
    "reflect"
    "regexp"
    "strconv"
    "strings"
    
    "github.com/talav/schema"
    "github.com/talav/mapstructure"
)

// Validation metadata
type ValidateMetadata struct {
    Required  bool
    MinLength int
    MaxLength int
    Pattern   *regexp.Regexp
    Min       *float64
    Max       *float64
}

// Parse validate tag
// Format: validate:"required,min:2,max:50,pattern:^[a-z]+$"
func ParseValidateTag(field reflect.StructField, index int, tagValue string) (any, error) {
    meta := &ValidateMetadata{}
    
    rules := strings.Split(tagValue, ",")
    for _, rule := range rules {
        rule = strings.TrimSpace(rule)
        
        if rule == "required" {
            meta.Required = true
            continue
        }
        
        parts := strings.SplitN(rule, ":", 2)
        if len(parts) != 2 {
            continue
        }
        
        key, value := parts[0], parts[1]
        switch key {
        case "min":
            if field.Type.Kind() == reflect.String {
                v, _ := strconv.Atoi(value)
                meta.MinLength = v
            } else {
                v, _ := strconv.ParseFloat(value, 64)
                meta.Min = &v
            }
            
        case "max":
            if field.Type.Kind() == reflect.String {
                v, _ := strconv.Atoi(value)
                meta.MaxLength = v
            } else {
                v, _ := strconv.ParseFloat(value, 64)
                meta.Max = &v
            }
            
        case "pattern":
            pattern, err := regexp.Compile(value)
            if err != nil {
                return nil, fmt.Errorf("invalid pattern: %w", err)
            }
            meta.Pattern = pattern
        }
    }
    
    return meta, nil
}

// Validator using custom metadata
func validateField(fieldValue any, meta *ValidateMetadata) error {
    // Check required
    if meta.Required && isZeroValue(fieldValue) {
        return fmt.Errorf("field is required")
    }
    
    // String validations
    if str, ok := fieldValue.(string); ok {
        if meta.MinLength > 0 && len(str) < meta.MinLength {
            return fmt.Errorf("minimum length is %d", meta.MinLength)
        }
        if meta.MaxLength > 0 && len(str) > meta.MaxLength {
            return fmt.Errorf("maximum length is %d", meta.MaxLength)
        }
        if meta.Pattern != nil && !meta.Pattern.MatchString(str) {
            return fmt.Errorf("does not match pattern")
        }
    }
    
    // Numeric validations
    if num, ok := convertToFloat64(fieldValue); ok {
        if meta.Min != nil && num < *meta.Min {
            return fmt.Errorf("minimum value is %f", *meta.Min)
        }
        if meta.Max != nil && num > *meta.Max {
            return fmt.Errorf("maximum value is %f", *meta.Max)
        }
    }
    
    return nil
}

// Helper: Check if value is zero
func isZeroValue(v any) bool {
    if v == nil {
        return true
    }
    val := reflect.ValueOf(v)
    return val.IsZero()
}

// Helper: Convert numeric types to float64
func convertToFloat64(v any) (float64, bool) {
    val := reflect.ValueOf(v)
    switch val.Kind() {
    case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
        return float64(val.Int()), true
    case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
        return float64(val.Uint()), true
    case reflect.Float32, reflect.Float64:
        return val.Float(), true
    default:
        return 0, false
    }
}

// Usage
type CreateUserRequest struct {
    Name     string  `schema:"name" validate:"required,min:2,max:50"`
    Email    string  `schema:"email" validate:"required,pattern:^[a-z0-9._%+-]+@[a-z0-9.-]+\\.[a-z]{2,}$"`
    Age      int     `schema:"age" validate:"min:18,max:120"`
    Username string  `schema:"username" validate:"required,min:3,max:20,pattern:^[a-zA-Z0-9_]+$"`
}
```

### Example 2: Documentation Tag

Generate API documentation from struct tags:

```go
package main

import (
    "fmt"
    "reflect"
    "strings"
    
    "github.com/talav/schema"
    "github.com/talav/mapstructure"
)

// Documentation metadata
type DocMetadata struct {
    Description string
    Example     string
    Deprecated  bool
    Since       string
}

// Parse doc tag
// Format: doc:"description|example|deprecated|since:v1.0"
func ParseDocTag(field reflect.StructField, index int, tagValue string) (any, error) {
    parts := strings.Split(tagValue, "|")
    
    meta := &DocMetadata{}
    
    if len(parts) > 0 {
        meta.Description = parts[0]
    }
    if len(parts) > 1 {
        meta.Example = parts[1]
    }
    if len(parts) > 2 {
        meta.Deprecated = parts[2] == "deprecated"
    }
    if len(parts) > 3 {
        meta.Since = strings.TrimPrefix(parts[3], "since:")
    }
    
    return meta, nil
}

// Generate OpenAPI-style documentation
func generateDocumentation(typ reflect.Type, metadata *schema.Metadata) {
    structMeta, _ := metadata.GetStructMetadata(typ)
    
    fmt.Printf("## %s\n\n", typ.Name())
    
    for _, field := range structMeta.Fields {
        // Get schema metadata
        schemaMeta, hasSchema := schema.GetTagMetadata[*schema.SchemaMetadata](&field, "schema")
        if !hasSchema {
            continue
        }
        
        // Get doc metadata
        docMeta, hasDoc := schema.GetTagMetadata[*DocMetadata](&field, "doc")
        
        fmt.Printf("### %s\n", schemaMeta.ParamName)
        fmt.Printf("- **Location**: %s\n", schemaMeta.Location)
        fmt.Printf("- **Type**: %s\n", field.Type.String())
        
        if hasDoc {
            fmt.Printf("- **Description**: %s\n", docMeta.Description)
            if docMeta.Example != "" {
                fmt.Printf("- **Example**: `%s`\n", docMeta.Example)
            }
            if docMeta.Deprecated {
                fmt.Printf("- **⚠️ Deprecated** (since %s)\n", docMeta.Since)
            }
        }
        fmt.Println()
    }
}

// Usage
type SearchRequest struct {
    Query    string `schema:"q" doc:"Search query text|golang web framework"`
    Page     int    `schema:"page" doc:"Page number (1-indexed)|1"`
    Limit    int    `schema:"limit" doc:"Results per page|20"`
    OldField string `schema:"old_field" doc:"Legacy field|deprecated|since:v2.0"`
}

func main() {
    // Register parsers
    registry := schema.NewTagParserRegistry(
        schema.WithTagParser("schema", schema.ParseSchemaTag, schema.DefaultSchemaMetadata),
        schema.WithTagParser("doc", ParseDocTag),
    )
    
    metadata := schema.NewMetadata(registry)
    
    // Generate docs
    generateDocumentation(reflect.TypeOf(SearchRequest{}), metadata)
}
```

### Example 3: Authorization Tag

Add authorization metadata:

```go
package main

import (
    "fmt"
    "reflect"
    "strings"
    
    "github.com/talav/schema"
    "github.com/talav/mapstructure"
)

// Authorization metadata
type AuthMetadata struct {
    Roles       []string
    Permissions []string
    Public      bool
}

// Parse auth tag
// Format: auth:"roles:admin,editor;permissions:read,write" or auth:"public"
func ParseAuthTag(field reflect.StructField, index int, tagValue string) (any, error) {
    meta := &AuthMetadata{}
    
    if tagValue == "public" {
        meta.Public = true
        return meta, nil
    }
    
    parts := strings.Split(tagValue, ";")
    for _, part := range parts {
        kv := strings.SplitN(part, ":", 2)
        if len(kv) != 2 {
            continue
        }
        
        key, value := kv[0], kv[1]
        switch key {
        case "roles":
            meta.Roles = strings.Split(value, ",")
        case "permissions":
            meta.Permissions = strings.Split(value, ",")
        }
    }
    
    return meta, nil
}

// Authorization middleware using metadata
func withAuthorization(codec *schema.Codec, metadata *schema.Metadata, reqType reflect.Type, handler http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        structMeta, _ := metadata.GetStructMetadata(reqType)
        
        // Check authorization metadata
        for _, field := range structMeta.Fields {
            if authMeta, ok := schema.GetTagMetadata[*AuthMetadata](&field, "auth"); ok {
                if authMeta.Public {
                    continue
                }
                
                // Check user roles/permissions
                userRoles := extractUserRoles(r)
                if !hasAnyRequiredRole(userRoles, authMeta.Roles) {
                    http.Error(w, "Forbidden", http.StatusForbidden)
                    return
                }
            }
        }
        
        handler(w, r)
    }
}

// Helper: Extract user roles from request (example - implement based on your auth system)
func extractUserRoles(r *http.Request) []string {
    // Example: extract from JWT, session, etc.
    token := r.Header.Get("Authorization")
    if token == "" {
        return []string{}
    }
    // Parse token and return roles
    return []string{"user"} // Simplified example
}

// Helper: Check if user has any required role
func hasAnyRequiredRole(userRoles, requiredRoles []string) bool {
    roleMap := make(map[string]bool)
    for _, role := range userRoles {
        roleMap[role] = true
    }
    
    for _, required := range requiredRoles {
        if roleMap[required] {
            return true
        }
    }
    
    return len(requiredRoles) == 0 // Allow if no roles required
}

// Usage
type AdminRequest struct {
    Action string `schema:"action" auth:"roles:admin;permissions:write"`
    UserID string `schema:"user_id" auth:"public"`
}
```

## Tag Parser Best Practices

### 1. Handle Errors Gracefully

```go
// ✅ Good: Return descriptive errors
func ParseCustomTag(field reflect.StructField, index int, tagValue string) (any, error) {
    if tagValue == "" {
        return nil, fmt.Errorf("empty tag value for field %s", field.Name)
    }
    
    // Parse...
    if invalid {
        return nil, fmt.Errorf("invalid tag format: expected key=value, got %s", tagValue)
    }
    
    return meta, nil
}
```

### 2. Provide Sensible Defaults

```go
// ✅ Good: Default values
func ParseCacheTag(field reflect.StructField, index int, tagValue string) (any, error) {
    meta := &CacheMetadata{
        TTL:     300,  // Default 5 minutes
        Enabled: true, // Default enabled
    }
    
    // Override with tag values
    // ...
    
    return meta, nil
}
```

### 3. Validate Tag Values

```go
// ✅ Good: Validate inputs
func ParseValidateTag(field reflect.StructField, index int, tagValue string) (any, error) {
    // Parse pattern
    pattern := extractPattern(tagValue)
    
    // Validate pattern is valid regex
    if _, err := regexp.Compile(pattern); err != nil {
        return nil, fmt.Errorf("invalid regex pattern: %w", err)
    }
    
    return meta, nil
}
```

### 4. Use Type-Safe Accessors

```go
// ✅ Good: Type-safe helper function
func getValidationMetadata(field *schema.FieldMetadata) (*ValidateMetadata, bool) {
    return schema.GetTagMetadata[*ValidateMetadata](field, "validate")
}

// Usage
if validateMeta, ok := getValidationMetadata(&field); ok {
    // Type-safe access
    if validateMeta.Required {
        // ...
    }
}
```

### 5. Document Tag Format

```go
// ✅ Good: Clear documentation
// ParseMyTag parses the "mytag" struct tag.
//
// Format: mytag:"option1=value1,option2=value2"
//
// Options:
//   - enabled: true/false (default: true)
//   - ttl: duration in seconds (default: 300)
//   - priority: high/medium/low (default: medium)
//
// Examples:
//   `mytag:"enabled=true,ttl=600"`
//   `mytag:"priority=high"`
func ParseMyTag(field reflect.StructField, index int, tagValue string) (any, error) {
    // Implementation...
}
```

## Registering Tag Parsers

### Basic Registration

```go
registry := schema.NewTagParserRegistry(
    schema.WithTagParser("schema", schema.ParseSchemaTag, schema.DefaultSchemaMetadata),
    schema.WithTagParser("body", schema.ParseBodyTag),
    schema.WithTagParser("validate", ParseValidateTag),
    schema.WithTagParser("cache", ParseCacheTag),
)

metadata := schema.NewMetadata(registry)
```

### With Defaults

```go
registry := schema.NewTagParserRegistry(
    schema.WithTagParser("validate", ParseValidateTag, DefaultValidateMetadata),
    schema.WithTagParser("cache", ParseCacheTag, DefaultCacheMetadata),
)
```

### Dynamic Registration

```go
// Start with default registry
registry := schema.NewDefaultTagParserRegistry()

// Add custom parsers dynamically
customRegistry := schema.NewTagParserRegistry(
    append(
        registry.All(),
        schema.WithTagParser("custom", ParseCustomTag),
    )...
)
```

## Next Steps

- [Extensibility Guide](extensibility.md) - Custom decoders and unmarshalers
- [API Documentation](https://pkg.go.dev/github.com/talav/schema) - Full API reference
