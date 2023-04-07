package loader

import (
	"bytes"
	"context"
	"errors"
	"net/url"
)

var (
	errCacheMiss       = errors.New("cache miss")
	errCacheAlreadyHas = errors.New("attempt to overwrite cache")
)

type memoryCache struct {
	sources map[string]*bytes.Buffer
}

func (l *memoryCache) Fetch(ctx context.Context, uri *url.URL) (*bytes.Buffer, error) {
	if buf, preloaded := l.sources[uri.String()]; preloaded {
		return buf, nil
	}
	return nil, errCacheMiss
}

func (mc *memoryCache) Store(uri *url.URL, data *bytes.Buffer) error {
	if mc.Has(uri) {
		return errCacheAlreadyHas
	}
	mc.sources[uri.String()] = data
	return nil
}

func (mc *memoryCache) Has(uri *url.URL) bool {
	_, has := mc.sources[uri.String()]
	return has
}
