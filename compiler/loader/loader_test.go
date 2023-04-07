package loader

import (
	"bytes"
	"context"
	"io"
	"net/url"
	"testing"
)

type mockFetcher struct {
	string // data
	bool   // Has Cache
	int    // Fetch Count
}

func (m *mockFetcher) Fetch(_ context.Context, _ *url.URL) (*bytes.Buffer, error) {
	m.int++
	return bytes.NewBufferString(m.string), nil
}

func (m *mockFetcher) Has(_ *url.URL) bool {
	return true
}

type fetchReporter interface {
	Cache
	WasExecuted() bool
}

func (m *mockFetcher) WasExecuted() bool {
	return m.int > 0
}

func Test_loader_Load(t *testing.T) {

	tests := []struct {
		name    string
		setup   []loaderOptions
		uri     string
		want    string
		wantErr bool
	}{
		{
			name:    "local load",
			uri:     "testdata/main.c4",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLoader(tt.setup...)

			ctx := context.Background()
			got, err := l.Load(ctx, tt.uri)

			if (err != nil) != tt.wantErr {
				t.Errorf("loader.Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				t.Log(err)
				return
			}

			gotBytes, _ := io.ReadAll(got)
			if bytes.Compare(gotBytes, []byte(tt.want)) != 0 {
				t.Errorf("Mismatched response\nwant %q\n got  %q", tt.want, gotBytes)
			}
		})
	}
}
