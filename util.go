package schema

import (
	"fmt"
	"maps"
	"strings"
)

// splitDeepKey parses bracket notation into parts (e.g., "user[profile][name]" -> ["user", "profile", "name"]).
func splitDeepKey(key string) ([]string, error) {
	var parts []string
	remaining := key

	for remaining != "" {
		// Find the next opening bracket
		openIdx := strings.Index(remaining, "[")

		if openIdx == -1 {
			// No more brackets, add remaining text if any
			if remaining != "" {
				parts = append(parts, remaining)
			}

			break
		}

		// Add text before the bracket (if any)
		if openIdx > 0 {
			parts = append(parts, remaining[:openIdx])
		}

		// Find the closing bracket
		closeIdx := strings.Index(remaining[openIdx:], "]")
		if closeIdx == -1 {
			return nil, fmt.Errorf("invalid format: unclosed bracket in key %q", key)
		}

		// Extract content between brackets
		// openIdx points to '[', closeIdx is relative to openIdx
		content := remaining[openIdx+1 : openIdx+closeIdx]
		parts = append(parts, content)

		// Move past the closing bracket
		remaining = remaining[openIdx+closeIdx+1:]
	}

	return parts, nil
}

// stringSliceToAny converts []string to []any.
func stringSliceToAny(strs []string) []any {
	result := make([]any, len(strs))
	for i, s := range strs {
		result[i] = s
	}

	return result
}

// splitAndTrim splits a string by separator, trims spaces from each part, and filters empty strings.
func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// splitToArray splits a string and returns as []any, or single value if only one part.
func splitToArray(s, sep string) any {
	parts := splitAndTrim(s, sep)

	if len(parts) > 1 {
		return stringSliceToAny(parts)
	}

	if len(parts) == 1 {
		return parts[0]
	}

	return nil
}

// appendToArray appends a value to an array, creating one if needed.
func appendToArray(existing any, val string) []any {
	if arr, ok := existing.([]any); ok {
		return append(arr, val)
	}

	if existing == nil {
		return []any{val}
	}

	// Convert existing value to array
	return []any{existing, val}
}

func mergeMaps(inputMaps ...map[string]any) map[string]any {
	result := make(map[string]any)
	for _, m := range inputMaps {
		maps.Copy(result, m)
	}

	return result
}

// getBaseParamName extracts the base parameter name from a query key.
// Examples:
//
//	"filter.type" -> "filter"
//	"filter[type]" -> "filter"
//	"ids" -> "ids"
func getBaseParamName(key string) string {
	// Check for bracket notation: filter[type] -> filter
	if idx := strings.Index(key, "["); idx != -1 {
		return key[:idx]
	}

	// Check for dotted notation: filter.type -> filter
	if idx := strings.Index(key, "."); idx != -1 {
		return key[:idx]
	}

	// No prefix, return as-is
	return key
}

// Helper to figure out how to process the body.
type bodyContentType struct {
	contentType string
	bodyType    BodyType
}

func newBodyContentType(contentType string, bodyType BodyType) *bodyContentType {
	contentType = strings.Split(contentType, ";")[0]

	return &bodyContentType{contentType: strings.TrimSpace(contentType), bodyType: bodyType}
}

func (b *bodyContentType) isForm() bool {
	return b.bodyType == BodyTypeStructured && strings.Contains(b.contentType, "application/x-www-form-urlencoded")
}

func (b *bodyContentType) isMultipart() bool {
	return b.bodyType == BodyTypeMultipart && strings.Contains(b.contentType, "multipart/")
}

func (b *bodyContentType) isFile() bool {
	return b.bodyType == BodyTypeFile
}

func (b *bodyContentType) isXML() bool {
	return b.bodyType == BodyTypeStructured &&
		(strings.Contains(b.contentType, "application/xml") || strings.Contains(b.contentType, "text/xml"))
}

// setNestedMapValue sets a value in a nested map using dotted key notation.
// Example: setNestedMapValue(m, "user.profile.name", "John") creates:
//
//	m["user"]["profile"]["name"] = "John"
//
// Returns error if a key conflict occurs (e.g., trying to set both "user" and "user.name").
func setNestedMapValue(m map[string]any, dottedKey string, value any) error {
	parts := strings.Split(dottedKey, ".")

	return setNestedMapValueByParts(m, parts, value, dottedKey)
}

// setDeepObjectMapValue sets a value in a nested map using bracket notation.
// Example: setDeepObjectMapValue(m, "user[profile][name]", []string{"John"}) creates:
//
//	m["user"]["profile"]["name"] = "John"
//
// Handles multiple values by converting to []any when valSlice has more than one element.
// Returns error if bracket notation is malformed or key conflicts occur.
func setDeepObjectMapValue(m map[string]any, bracketKey string, valSlice []string) error {
	parts, err := splitDeepKey(bracketKey)
	if err != nil {
		return err
	}

	if len(parts) < 2 {
		return fmt.Errorf("invalid format: deep object key %q must have at least one bracket pair", bracketKey)
	}

	// Convert single value to string, multiple to []any
	var value any = valSlice[0]
	if len(valSlice) > 1 {
		value = stringSliceToAny(valSlice)
	}

	return setNestedMapValueByParts(m, parts, value, bracketKey)
}

// setNestedMapValueByParts navigates through or creates nested maps and sets the final value.
// Returns error if a key exists as both a map and a primitive value (conflict).
func setNestedMapValueByParts(m map[string]any, parts []string, value any, originalKey string) error {
	if len(parts) == 0 {
		return fmt.Errorf("invalid format: empty key parts for %q", originalKey)
	}

	// Navigate to the parent map, creating intermediate maps as needed
	current := m
	for _, key := range parts[:len(parts)-1] {
		if current[key] == nil {
			current[key] = make(map[string]any)
		}

		next, ok := current[key].(map[string]any)
		if !ok {
			return fmt.Errorf("key conflict: %q is both object and primitive at %q", key, originalKey)
		}
		current = next
	}

	// Set final value, checking for map-to-primitive conflicts
	finalKey := parts[len(parts)-1]
	if existing := current[finalKey]; existing != nil {
		if _, isMap := existing.(map[string]any); isMap {
			if _, valueIsMap := value.(map[string]any); !valueIsMap {
				return fmt.Errorf("key conflict: %q is both object and primitive at %q", finalKey, originalKey)
			}
		}
	}

	current[finalKey] = value

	return nil
}

// extractBoolean extracts a boolean value from options map.
func extractBoolean(options map[string]string, key string, defaultValue bool) bool {
	if value, exists := options[key]; exists {
		if value == "" {
			// Flag form: "required" means true
			return true
		}

		// Value form: "required=true/false"
		return value == optValueTrue
	}

	return defaultValue
}
