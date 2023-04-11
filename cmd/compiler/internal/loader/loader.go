package loader

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
)

type Loader interface {
	Load(context.Context, string) ([]byte, error)
}

type fetcher interface {
	Fetch(context.Context, *url.URL) (*bytes.Buffer, error)
}

type cache interface {
	fetcher

	Store(*url.URL, *bytes.Buffer) error
	Has(*url.URL) bool
}

type sourceLoader struct {
	cache   cache
	fetcher fetcher

	config *SourceLoadConfig
}

type SourceLoadConfig struct {
	CanFetchRemote bool
}

func (l *sourceLoader) Load(ctx context.Context, uri string) ([]byte, error) {

	netUrl, err := l.validateUri(uri)
	if err != nil {
		return nil, err
	}

	var buf *bytes.Buffer
	if l.cache != nil && l.cache.Has(netUrl) {
		buf, _ = l.cache.Fetch(ctx, netUrl)
		return buf.Bytes(), nil
	}

	var f fetcher
	if l.fetcher != nil {
		f = l.fetcher
	} else {
		f = defaultClient
	}

	buf, err = f.Fetch(ctx, netUrl)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (l *sourceLoader) validateUri(uri string) (*url.URL, error) {
	netUrl, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid remote file specified %s: %s", uri, err)
	}

	if netUrl.Scheme != "https" && netUrl.Scheme != "file" && netUrl.Scheme != "" {
		return nil, fmt.Errorf("unable to fetch remote file: scheme must be 'https:' or 'file:' got %s", netUrl.Scheme)
	}

	if netUrl.Scheme == "https" && !l.config.CanFetchRemote {
		return nil, fmt.Errorf("error fetching remote file: remote fetch is disabled")
	}

	netUrl.Fragment = ""

	return netUrl, nil
}

type loaderOptions func(*sourceLoader)

func WithMemoryCache() loaderOptions {
	return func(l *sourceLoader) {
		l.cache = new(memoryCache)
	}
}

func NewLoader(opts ...loaderOptions) Loader {
	l := new(sourceLoader)
	for i := range opts {
		opts[i](l)
	}
	return l
}
