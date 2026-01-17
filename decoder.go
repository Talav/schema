package schema

import (
	"fmt"
	"net/http"
	"net/url"
)

// defaultDecoder handles decoding of parameter strings to maps.
type defaultDecoder struct {
	schemaTag string
	bodyTag   string
	metadata  *Metadata
}

// newDefaultDecoder creates a new decoder.
func NewDecoder(metadata *Metadata, schemaTag string, bodyTag string) Decoder {
	return &defaultDecoder{
		metadata:  metadata,
		schemaTag: schemaTag,
		bodyTag:   bodyTag,
	}
}

func NewDefaultDecoder() Decoder {
	return NewDecoder(NewDefaultMetadata(), defaultSchemaTag, defaultBodyTag)
}

// Decode decodes HTTP request parameters into a map.
func (d *defaultDecoder) Decode(request *http.Request, routerParams map[string]string, metadata *StructMetadata) (map[string]any, error) {
	queryResult, err := d.decodeQuery(request, metadata)
	if err != nil {
		return nil, err
	}

	headerResult, err := d.decodeHeader(request, metadata)
	if err != nil {
		return nil, err
	}

	cookieResult, err := d.decodeCookie(request, metadata)
	if err != nil {
		return nil, err
	}

	pathResult, err := d.decodePath(routerParams, metadata)
	if err != nil {
		return nil, err
	}

	bodyResult, err := d.decodeBody(request, metadata)
	if err != nil {
		return nil, err
	}

	return mergeMaps(queryResult, headerResult, cookieResult, pathResult, bodyResult), nil
}

// decodePath decodes path parameters from router params.
func (d *defaultDecoder) decodePath(routerParams map[string]string, metadata *StructMetadata) (map[string]any, error) {
	result := make(map[string]any)
	for _, field := range filterByLocation(metadata.Fields, LocationPath) {
		schemaMeta, ok := GetTagMetadata[*SchemaMetadata](&field, d.schemaTag)
		if !ok {
			continue
		}
		v, err := d.decodeValueByStyle(routerParams[schemaMeta.ParamName], schemaMeta.Style, schemaMeta.Explode)
		if err != nil {
			return nil, err
		}
		result[schemaMeta.ParamName] = v
	}

	return result, nil
}

// decodeCookie decodes cookie parameters from HTTP request.
func (d *defaultDecoder) decodeCookie(request *http.Request, metadata *StructMetadata) (map[string]any, error) {
	result := make(map[string]any)
	for _, field := range filterByLocation(metadata.Fields, LocationCookie) {
		schemaMeta, ok := GetTagMetadata[*SchemaMetadata](&field, d.schemaTag)
		if !ok {
			continue
		}
		cookie, err := request.Cookie(schemaMeta.ParamName)
		if err != nil {
			// Cookie not present - skip (required validation happens elsewhere)
			continue
		}

		// Decode cookie value according to OpenAPI v3 form style
		v, err := d.decodeValueByStyle(cookie.Value, schemaMeta.Style, schemaMeta.Explode)
		if err != nil {
			return nil, fmt.Errorf("failed to decode cookie %q: %w", schemaMeta.ParamName, err)
		}
		// Use ParamName (tag name) directly - mapstructure will map to struct fields
		result[schemaMeta.ParamName] = v
	}

	return result, nil
}

// decodeHeader decodes header parameters from HTTP request.
func (d *defaultDecoder) decodeHeader(request *http.Request, metadata *StructMetadata) (map[string]any, error) {
	result := make(map[string]any)
	for _, field := range filterByLocation(metadata.Fields, LocationHeader) {
		schemaMeta, ok := GetTagMetadata[*SchemaMetadata](&field, d.schemaTag)
		if !ok {
			continue
		}
		v, err := d.decodeValueByStyle(request.Header.Get(schemaMeta.ParamName), schemaMeta.Style, schemaMeta.Explode)
		if err != nil {
			return nil, err
		}
		result[schemaMeta.ParamName] = v
	}

	return result, nil
}

// decodeQuery decodes query parameters from HTTP request.
func (d *defaultDecoder) decodeQuery(request *http.Request, metadata *StructMetadata) (map[string]any, error) {
	result := make(map[string]any)

	// Filter fields by location, we only want query fields
	queryFields := filterByLocation(metadata.Fields, LocationQuery)
	if len(queryFields) == 0 {
		return result, nil
	}

	// Parse query string to get all values
	allValues, err := url.ParseQuery(request.URL.RawQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query string: %w", err)
	}
	if len(allValues) == 0 {
		return result, nil
	}

	styleGroups := groupByStyle(queryFields)
	// Decode query string once per style group
	for styleKey, fields := range styleGroups {
		// Build filtered query string for this style group
		filteredValues := filterQueryValuesByFields(allValues, fields)

		if len(filteredValues) == 0 {
			continue
		}

		// Decode full query string with this style
		decodedMap, err := d.decodeByStyle(filteredValues.Encode(), styleKey.Style, styleKey.Explode)
		if err != nil {
			return nil, fmt.Errorf("failed to decode query with style %q: %w", styleKey.Style, err)
		}

		// Merge decoded map into result
		result = mergeMaps(result, decodedMap)
	}

	return result, nil
}

// decodeValueByStyle dispatches to the appropriate style-specific decoder for single values.
func (d *defaultDecoder) decodeValueByStyle(value string, style Style, explode bool) (any, error) {
	switch style {
	case StyleSimple:
		return d.decodeSimpleStyle(value)
	case StyleLabel:
		return d.decodeLabelStyle(value, explode)
	case StyleMatrix, StyleForm, StyleSpaceDelimited, StylePipeDelimited, StyleDeepObject:
		// These styles are not valid for path/header/cookie parameters
		return nil, fmt.Errorf("invalid style: %q is not valid for single-value parameters", style)
	default:
		// Defensive: should never happen if Options validation works correctly
		return nil, fmt.Errorf("invalid style: %q", style)
	}
}

// decodeByStyle dispatches to the appropriate style-specific decoder for map values.
func (d *defaultDecoder) decodeByStyle(value string, style Style, explode bool) (map[string]any, error) {
	switch style {
	case StyleForm:
		return d.decodeFormStyle(value)
	case StyleMatrix:
		return d.decodeMatrixStyle(value, explode)
	case StyleSpaceDelimited:
		return d.decodeSpaceDelimited(value)
	case StylePipeDelimited:
		return d.decodePipeDelimited(value)
	case StyleDeepObject:
		return d.decodeDeepObject(value)
	case StyleSimple, StyleLabel:
		// These styles return single values, not maps, and are handled by decodeValueByStyle instead
		return nil, fmt.Errorf("invalid style: %q returns single value, not map", style)
	default:
		// Defensive: should never happen if Options validation works correctly
		return nil, fmt.Errorf("invalid style: %q", style)
	}
}

func filterQueryValuesByFields(allValues url.Values, fields []FieldMetadata) url.Values {
	filteredValues := url.Values{}
	for key, vals := range allValues {
		baseName := getBaseParamName(key)
		// Check if this key matches any field's ParamName (tag name) in this style group
		for _, field := range fields {
			// Get schema metadata (could be from explicit tag or default)
			schemaMeta, ok := GetTagMetadata[*SchemaMetadata](&field, "schema")
			if !ok {
				continue
			}

			// Use ParamName (tag name) for filtering, not MapKey (field name)
			// Query params in URL use tag names (e.g., "user_name"), not field names (e.g., "UserName")
			if schemaMeta.ParamName == baseName {
				filteredValues[key] = vals

				break
			}
		}
	}

	return filteredValues
}

func filterByLocation(fields []FieldMetadata, location ParameterLocation) []FieldMetadata {
	var result []FieldMetadata
	for _, field := range fields {
		if schemaMeta, ok := GetTagMetadata[*SchemaMetadata](&field, "schema"); ok {
			if schemaMeta.Location == location {
				result = append(result, field)
			}
		}
	}

	return result
}

func groupByStyle(fields []FieldMetadata) map[styleGroup][]FieldMetadata {
	styleGroups := make(map[styleGroup][]FieldMetadata)
	for _, field := range fields {
		if schemaMeta, ok := GetTagMetadata[*SchemaMetadata](&field, "schema"); ok {
			sg := styleGroup{
				Style:   schemaMeta.Style,
				Explode: schemaMeta.Explode,
			}
			styleGroups[sg] = append(styleGroups[sg], field)
		}
	}

	return styleGroups
}
