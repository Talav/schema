package schema

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"reflect"
)

// decodeBody decodes the HTTP request body based on content type.
func (d *defaultDecoder) decodeBody(request *http.Request, metadata *StructMetadata) (map[string]any, error) {
	// Extract body field by iterating through fields
	var bodyField *FieldMetadata
	for i := range metadata.Fields {
		if _, ok := GetTagMetadata[*BodyMetadata](&metadata.Fields[i], d.bodyTag); ok {
			bodyField = &metadata.Fields[i]

			break
		}
	}

	bodyMeta, ok := GetTagMetadata[*BodyMetadata](bodyField, d.bodyTag)
	if !ok {
		return make(map[string]any), nil
	}

	bodyContentType := newBodyContentType(request.Header.Get("Content-Type"), bodyMeta.BodyType)

	// Multipart needs raw request for ParseMultipartForm - handle before reading body
	if bodyContentType.isMultipart() {
		return d.decodeMultipartBody(request, bodyField)
	}

	// Check if target field is io.ReadCloser for streaming (file body only)
	if bodyContentType.isFile() {
		return d.decodeFileBodyStream(request, bodyField, bodyMeta)
	}

	// Read body for other content types
	bodyBytes, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	if len(bodyBytes) == 0 {
		return make(map[string]any), nil
	}

	if bodyContentType.isForm() {
		return d.decodeURLEncodedForm(bodyBytes, bodyField)
	}

	if bodyContentType.isXML() {
		return d.decodeXMLBody(bodyBytes, bodyField)
	}

	// try JSON as a fallback
	return d.decodeJSONBody(bodyBytes, bodyField)
}

// decodeFileBodyStream handles file body streaming for io.ReadCloser fields.
func (d *defaultDecoder) decodeFileBodyStream(request *http.Request, bodyField *FieldMetadata, bodyMeta *BodyMetadata) (map[string]any, error) {
	// If target field is io.ReadCloser, return request body directly for streaming
	if bodyField.Type.Implements(reflect.TypeOf((*io.ReadCloser)(nil)).Elem()) {
		return map[string]any{bodyMeta.MapKey: request.Body}, nil
	}

	// Otherwise, read into memory as []byte
	bodyBytes, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	// Empty body returns empty map
	if len(bodyBytes) == 0 {
		return make(map[string]any), nil
	}

	return map[string]any{bodyMeta.MapKey: bodyBytes}, nil
}

// decodeMultipartBody decodes multipart form body content.
func (d *defaultDecoder) decodeMultipartBody(r *http.Request, bodyField *FieldMetadata) (map[string]any, error) {
	// Parse multipart form
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB max
		return nil, fmt.Errorf("failed to parse multipart form: %w", err)
	}

	form := r.MultipartForm
	if form == nil {
		return nil, fmt.Errorf("not a multipart form")
	}

	// Get the struct type
	structType := bodyField.Type
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}

	// Get metadata for the body struct to use schema tags
	metadata, err := d.metadata.GetStructMetadata(structType)
	if err != nil {
		return nil, fmt.Errorf("failed to get struct metadata: %w", err)
	}

	bodyMeta, ok := GetTagMetadata[*BodyMetadata](bodyField, "body")
	if !ok {
		return nil, fmt.Errorf("field is not a body field")
	}

	fieldMap := make(map[string]any)

	// Process each field using cached metadata
	for _, fieldMeta := range metadata.Fields {
		if err := d.processMultipartField(form, fieldMeta, fieldMap); err != nil {
			return nil, err
		}
	}

	return map[string]any{bodyMeta.MapKey: fieldMap}, nil
}

// decodeURLEncodedForm decodes URL-encoded form body content.
func (d *defaultDecoder) decodeURLEncodedForm(bodyBytes []byte, bodyField *FieldMetadata) (map[string]any, error) {
	// Decode form data
	decodedMap, err := d.decodeFormStyle(string(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to parse form: %w", err)
	}
	bodyMeta, ok := GetTagMetadata[*BodyMetadata](bodyField, d.bodyTag)
	if !ok {
		return nil, fmt.Errorf("field is not a body field")
	}

	// Get struct type and metadata to filter unknown fields
	structType := bodyField.Type
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}
	metadata, err := d.metadata.GetStructMetadata(structType)
	if err != nil {
		return nil, fmt.Errorf("failed to get struct metadata: %w", err)
	}

	// Filter decodedMap to only include fields present in metadata
	filteredMap := make(map[string]any)
	for _, fieldMeta := range metadata.Fields {
		schemaMeta, ok := GetTagMetadata[*SchemaMetadata](&fieldMeta, d.schemaTag)
		if !ok {
			continue
		}
		if val, exists := decodedMap[schemaMeta.ParamName]; exists {
			filteredMap[schemaMeta.ParamName] = val
		}
	}

	return map[string]any{bodyMeta.MapKey: filteredMap}, nil
}

// decodeXMLBody decodes XML body content.
func (d *defaultDecoder) decodeXMLBody(bodyBytes []byte, bodyField *FieldMetadata) (map[string]any, error) {
	bodyMeta, ok := GetTagMetadata[*BodyMetadata](bodyField, "body")
	if !ok {
		return nil, fmt.Errorf("field is not a body field")
	}

	// Create a new instance of the target type
	targetType := bodyField.Type
	if targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
	}

	targetValue := reflect.New(targetType).Interface()
	if err := xml.Unmarshal(bodyBytes, targetValue); err != nil {
		return nil, fmt.Errorf("failed to unmarshal XML: %w", err)
	}

	// Dereference the pointer created by reflect.New
	parsed := reflect.ValueOf(targetValue).Elem().Interface()

	return map[string]any{bodyMeta.MapKey: parsed}, nil
}

// decodeJSONBody decodes JSON body content.
// For struct types, unmarshals into the concrete type so json tags are respected.
// For non-struct types (slices, scalars), falls back to generic unmarshal into any.
func (d *defaultDecoder) decodeJSONBody(bodyBytes []byte, bodyField *FieldMetadata) (map[string]any, error) {
	bodyMeta, ok := GetTagMetadata[*BodyMetadata](bodyField, d.bodyTag)
	if !ok {
		return nil, fmt.Errorf("field is not a body field")
	}

	targetType := bodyField.Type
	if targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
	}

	// For struct types, unmarshal into the concrete type (respects json tags)
	if targetType.Kind() == reflect.Struct {
		target := reflect.New(targetType).Interface()
		if err := json.Unmarshal(bodyBytes, target); err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
		}

		parsed := reflect.ValueOf(target).Elem().Interface()

		return map[string]any{bodyMeta.MapKey: parsed}, nil
	}

	// For non-struct types (arrays, scalars), fall back to generic unmarshal
	var parsed any
	if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return map[string]any{bodyMeta.MapKey: parsed}, nil
}

// processMultipartField processes a single multipart form field (file or regular).
func (d *defaultDecoder) processMultipartField(form *multipart.Form, fieldMeta FieldMetadata, result map[string]any) error {
	var paramName string
	if schemaMeta, ok := GetTagMetadata[*SchemaMetadata](&fieldMeta, "schema"); ok {
		paramName = schemaMeta.ParamName
	} else {
		return nil
	}

	if d.isFileField(fieldMeta.Type) {
		fileHeaders := form.File[paramName]
		if len(fileHeaders) == 0 {
			return nil
		}

		fileHeaderType := reflect.TypeOf((*multipart.FileHeader)(nil))

		if fieldMeta.Type == fileHeaderType {
			result[paramName] = fileHeaders[0]

			return nil
		}

		if fieldMeta.Type.Kind() == reflect.Slice && fieldMeta.Type.Elem() == fileHeaderType {
			result[paramName] = fileHeaders

			return nil
		}

		fileReaders, err := d.openMultipartFiles(fileHeaders, fieldMeta.Type)
		if err != nil {
			return fmt.Errorf("failed to open file field %s: %w", paramName, err)
		}
		result[paramName] = fileReaders

		return nil
	}

	// Regular form field: use schema tag name
	values := form.Value[paramName]
	if len(values) > 0 {
		if len(values) == 1 {
			result[paramName] = values[0]
		} else {
			result[paramName] = stringSliceToAny(values)
		}
	}

	return nil
}

// isFileField checks if a type represents a file field.
func (d *defaultDecoder) isFileField(typ reflect.Type) bool {
	fileHeaderType := reflect.TypeOf((*multipart.FileHeader)(nil))

	if typ == fileHeaderType {
		return true
	}

	if typ.Kind() == reflect.Slice && typ.Elem() == fileHeaderType {
		return true
	}

	if typ.Implements(reflect.TypeOf((*io.ReadCloser)(nil)).Elem()) {
		return true
	}

	if typ.Kind() == reflect.Slice && typ.Elem().Kind() == reflect.Uint8 {
		return true
	}

	if typ.Kind() == reflect.Slice {
		elemType := typ.Elem()
		if elemType.Implements(reflect.TypeOf((*io.ReadCloser)(nil)).Elem()) {
			return true
		}
	}

	return false
}

// openMultipartFiles opens multipart file headers and returns io.ReadCloser(s).
// Returns io.ReadCloser for single file fields, []io.ReadCloser for multiple file fields.
func (d *defaultDecoder) openMultipartFiles(fileHeaders []*multipart.FileHeader, fieldType reflect.Type) (any, error) {
	// Check if field is a slice
	if fieldType.Kind() == reflect.Slice {
		// Multiple files - return []io.ReadCloser
		readers := make([]io.ReadCloser, len(fileHeaders))
		for i, fh := range fileHeaders {
			file, err := fh.Open()
			if err != nil {
				// Close already opened files on error
				for j := 0; j < i; j++ {
					_ = readers[j].Close()
				}

				return nil, fmt.Errorf("failed to open file[%d]: %w", i, err)
			}
			readers[i] = file
		}

		return readers, nil
	}

	file, err := fileHeaders[0].Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}
