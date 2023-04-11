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
			name:    "invalid local file",
			uri:     "nonexistant.llama",
			wantErr: true,
		},
		{
			name:    "chrooted local load",
			uri:     "main.c4",
			setup:   []loaderOption{RootedAt("testdata")},
			wantErr: false,
		},
		{
			name:    "chrooted local load blocks outside",
			uri:     "/etc/passwd",
			setup:   []loaderOption{RootedAt("testdata")},
			wantErr: true,
		},
		{
			name:    "block remote load",
			uri:     "https://example.com/resources/foo.c4",
			wantErr: true,
		},
		{
			name:    "allow remote load",
			setup:   []loaderOption{AllowRemote()},
			uri:     "https://github.com/AndrewBurian/c4/blob/main/cmd/compiler/internal/loader/testdata/main.c4",
			wantErr: false,
		},
		{
			name:    "allow remote but allow list block",
			setup:   []loaderOption{AllowRemote(), AllowedRemoteHosts("example.com")},
			uri:     "https://github.com/AndrewBurian/c4/blob/main/cmd/compiler/internal/loader/testdata/main.c4",
			wantErr: true,
		},
		{
			name:    "allow remote and allow listed",
			setup:   []loaderOption{AllowRemote(), AllowedRemoteHosts("github.com")},
			uri:     "https://github.com/AndrewBurian/c4/blob/main/cmd/compiler/internal/loader/testdata/main.c4",
			wantErr: false,
		},
		{
			name:  "block redirect",
			setup: []loaderOption{AllowRemote(), AllowedRemoteHosts("github.com")},
			// /raw/ will redirect to raw.githubusercontent
			uri:     "https://github.com/AndrewBurian/c4/raw/main/cmd/compiler/internal/loader/testdata/main.c4",
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
