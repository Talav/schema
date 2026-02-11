package schema

import (
	"fmt"
	"net/url"
	"strings"
)

// decodeFormStyle parses form-style parameters (the default for query and cookie parameters).
// Handles both exploded (?ids=1&ids=2) and non-exploded (?ids=1,2,3) arrays.
// Supports nested structures using dot notation (filter.type=car&filter.color=red).
func (d *defaultDecoder) decodeFormStyle(data string) (map[string]any, error) {
	result := make(map[string]any)

	values, err := url.ParseQuery(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query: %w", err)
	}

	for key, valSlice := range values {
		value := d.processFormValue(valSlice)
		if value == nil {
			continue
		}

		// Handle nested structures (dotted notation: filter.type, filter.color)
		if err := setNestedMapValue(result, key, value); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// decodeSimpleStyle parses simple style parameters (default for path and header parameters).
// Simple style has no prefix/suffix and treats comma-separated values as arrays.
// Examples: "123" (scalar), "1,2,3" (array).
func (d *defaultDecoder) decodeSimpleStyle(data string) (any, error) {
	return data, nil
}

// decodeLabelStyle parses label style parameters (used in path parameters).
// Label style is period-prefixed and handles both arrays and objects.
//
// When explode=true:
//   - Arrays: .1.2.3 becomes ["1", "2", "3"]
//   - Objects: .x.1024.y.768 becomes {"x": "1024", "y": "768"}
//
// When explode=false:
//   - Arrays: .1,2,3 becomes ["1", "2", "3"]
//   - Objects: .x,1024,y,768 becomes {"x": "1024", "y": "768"}
func (d *defaultDecoder) decodeLabelStyle(data string, explode bool) (any, error) {
	data = strings.TrimPrefix(data, ".")

	if explode {
		// Period-separated: .1.2.3 (array) or .x.1024.y.768 (object)
		parts := strings.Split(data, ".")
		if len(parts) == 0 {
			return data, nil
		}

		// If even number of parts, likely an object (key-value pairs)
		// If odd or single, likely an array
		if len(parts) > 1 && len(parts)%2 == 0 {
			// Object: .x.1024.y.768 -> {"x": "1024", "y": "768"}
			result := make(map[string]any)
			for i := 0; i < len(parts); i += 2 {
				result[parts[i]] = parts[i+1]
			}

			return result, nil
		}

		// Array: period-separated values
		return splitToArray(data, "."), nil
	}

	// Non-exploded: comma-separated
	// Arrays: .1,2,3
	// Objects: .x,1024,y,768
	parts := splitAndTrim(data, ",")
	if len(parts) > 1 && len(parts)%2 == 0 {
		// Object: comma-separated key-value pairs
		result := make(map[string]any)
		for i := 0; i < len(parts); i += 2 {
			result[parts[i]] = parts[i+1]
		}

		return result, nil
	}

	// Array or single value: comma-separated
	return splitToArray(data, ","), nil
}

// decodeMatrixStyle parses matrix style parameters (used in path parameters).
// Matrix style is semicolon-prefixed with key=value pairs.
//
// When explode=true:
//
//	Each value is a separate key=value pair: ;ids=1;ids=2;ids=3
//
// When explode=false:
//
//	Values are comma-separated: ;ids=1,2,3
//
// Returns a map where duplicate keys accumulate values into arrays.
func (d *defaultDecoder) decodeMatrixStyle(path string, explode bool) (map[string]any, error) {
	result := make(map[string]any)

	data := strings.TrimPrefix(path, ";")

	// Parse key=value pairs
	for pair := range strings.SplitSeq(data, ";") {
		if pair == "" {
			continue
		}

		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid format: matrix pair %q", pair)
		}

		key := parts[0]
		val := parts[1]

		if explode {
			// Each value is separate: ;ids=1;ids=2;ids=3
			result[key] = appendToArray(result[key], val)
		} else {
			// Comma-separated: ;ids=1,2,3
			result[key] = splitToArray(val, ",")
		}
	}

	return result, nil
}

// decodeSpaceDelimited parses space-delimited query parameters.
// Example: ?ids=1%202%203 becomes {"ids": ["1", "2", "3"]}
// Only valid for query parameters with primitive arrays.
func (d *defaultDecoder) decodeSpaceDelimited(query string) (map[string]any, error) {
	return d.decodeDelimited(query, " ")
}

// decodePipeDelimited parses pipe-delimited query parameters.
// Example: ?ids=1|2|3 becomes {"ids": ["1", "2", "3"]}
// Only valid for query parameters with primitive arrays.
func (d *defaultDecoder) decodePipeDelimited(query string) (map[string]any, error) {
	return d.decodeDelimited(query, "|")
}

// decodeDeepObject parses deepObject style query parameters.
// Supports bracket notation for nested objects and arrays.
// Example: ?filter[type]=car&filter[color]=red becomes {"filter": {"type": "car", "color": "red"}}
//
// Only valid for query parameters with object values.
// Per OpenAPI spec, deepObject does not support arrays - use form style for array parameters.
func (d *defaultDecoder) decodeDeepObject(query string) (map[string]any, error) {
	result := make(map[string]any)

	values, err := url.ParseQuery(query)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query: %w", err)
	}

	for key, valSlice := range values {
		// Deep object format: filter[type]=car
		if strings.Contains(key, "[") && strings.Contains(key, "]") {
			if err := setDeepObjectMapValue(result, key, valSlice); err != nil {
				return nil, err
			}
		} else {
			// Regular key-value
			result[key] = valSlice[0]
			if len(valSlice) > 1 {
				result[key] = stringSliceToAny(valSlice)
			}
		}
	}

	return result, nil
}

// processFormValue determines how to handle form values based on their structure.
// Multiple values (?ids=1&ids=2) are treated as arrays.
// Single comma-separated values (?ids=1,2,3) are split into arrays.
// Single non-comma values are returned as-is.
func (d *defaultDecoder) processFormValue(valSlice []string) any {
	if len(valSlice) == 0 {
		return nil
	}

	// Multiple values: treat as array (aligns with standard HTTP behavior)
	// This handles both explode=true (spec: ?ids=1&ids=2) and
	// explode=false edge case (non-spec: ?ids=1&ids=2 when expecting ?ids=1,2)
	if len(valSlice) > 1 {
		return stringSliceToAny(valSlice)
	}

	// Single value: check if comma-separated (explode=false spec case: ?ids=1,2,3)
	val := valSlice[0]
	if val == "" {
		return nil
	}

	if strings.Contains(val, ",") {
		return splitToArray(val, ",")
	}

	return val
}

// decodeDelimited parses delimited query parameters with a custom separator.
// Used by decodeSpaceDelimited and decodePipeDelimited.
// Takes the last value if multiple values are present for the same key.
func (d *defaultDecoder) decodeDelimited(query string, sep string) (map[string]any, error) {
	result := make(map[string]any)

	values, err := url.ParseQuery(query)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query: %w", err)
	}

	for key, valSlice := range values {
		if len(valSlice) > 0 {
			val := valSlice[len(valSlice)-1]
			if val != "" {
				result[key] = splitToArray(val, sep)
			}
		}
	}

	return result, nil
}
