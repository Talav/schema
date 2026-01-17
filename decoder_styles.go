package schema

import (
	"fmt"
	"net/url"
	"strings"
)

// decodeFormStyle parses form-style data (query string or form data).
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

// decodeSimpleStyle parses simple style (no prefix/suffix, comma-separated).
func (d *defaultDecoder) decodeSimpleStyle(data string) (any, error) {
	return data, nil
}

// decodeLabelStyle parses label style (period-prefixed).
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

// decodeMatrixStyle parses matrix style (semicolon-prefixed).
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
func (d *defaultDecoder) decodeSpaceDelimited(query string) (map[string]any, error) {
	return d.decodeDelimited(query, " ")
}

// decodePipeDelimited parses pipe-delimited query parameters.
func (d *defaultDecoder) decodePipeDelimited(query string) (map[string]any, error) {
	return d.decodeDelimited(query, "|")
}

// decodeDeepObject parses deepObject style query parameters.
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

// processFormValue processes a form value.
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

// decodeDelimited parses delimited query parameters (space or pipe).
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
