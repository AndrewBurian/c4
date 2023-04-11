package loader

import (
	"context"
	"testing"
)

func Test_loader_Load(t *testing.T) {

	tests := []struct {
		name    string
		setup   []loaderOption
		uri     string
		wantErr bool
	}{
		{
			name:    "local load",
			uri:     "testdata/main.c4",
			wantErr: false,
		},
		{
			name:    "block remote load",
			uri:     "https://example.com/resources/foo.c4",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLoader(tt.setup...)

			ctx := context.Background()
			_, err := l.Load(ctx, tt.uri)

			if (err != nil) != tt.wantErr {
				t.Errorf("loader.Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				t.Log(err)
				return
			}
		})
	}
}
