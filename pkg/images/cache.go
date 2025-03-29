package images

import (
	"context"
	"github.com/mdfriday/hugoverse/pkg/images/cache"
	"github.com/mdfriday/hugoverse/pkg/images/storage"
	"github.com/mdfriday/hugoverse/pkg/tracing"
)

// Cache is an image cache
type Cache = cache.Auto

// NewCache instantiates a new cache
func NewCache(tracer *tracing.Tracer, cacheProvider cache.Provider, storageProvider storage.Provider) *Cache {
	return &Cache{
		Tracer:   tracer,
		Provider: cacheProvider,
		Loader: func(ctx context.Context, key string) (data []byte, err error) {
			ctx, span := tracer.Start(ctx, "image.Cache.Loader")
			defer span.End()

			return storageProvider.Get(ctx, key)
		},
	}
}
