package schema

import (
	"fmt"
	"reflect"
	"sync"
)

// metadataCache provides caching for struct field metadata.
type metadataCache struct {
	cache   sync.Map // map[reflect.Type]*StructMetadata
	builder *metadataBuilder
}

// newMetadataCache creates a new metadata cache.
func newMetadataCache(builder *metadataBuilder) *metadataCache {
	return &metadataCache{
		builder: builder,
	}
}

// Get retrieves or builds cached struct field metadata.
func (c *metadataCache) get(typ reflect.Type) (*StructMetadata, error) {
	// Check cache first
	if cached, ok := c.cache.Load(typ); ok {
		if fields, ok := cached.(*StructMetadata); ok {
			return fields, nil
		}
	}

	// Build cache
	fields, err := c.builder.buildStructMetadata(typ)
	if err != nil {
		return nil, fmt.Errorf("failed to build struct metadata: %w", err)
	}

	// Store in cache (or get existing if another goroutine stored it first)
	actual, _ := c.cache.LoadOrStore(typ, fields)
	fields, _ = actual.(*StructMetadata)

	return fields, nil
}
