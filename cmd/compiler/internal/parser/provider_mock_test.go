package parser

import (
	"bytes"
	"fmt"

	"go.burian.dev/c4/cmd/compiler/internal/lexer"
)

type mockDependencies struct {
	sources      map[string]string
	lexedSources map[string]*lexer.LexedSource
	l            *lexer.Lexer
}

var _ Provider = &mockDependencies{}

func (m *mockDependencies) GetTokenStreamFor(name string) (lexer.TokenStream, error) {
	if m.lexedSources == nil {
		m.lexedSources = make(map[string]*lexer.LexedSource)
	}
	if ls, has := m.lexedSources[name]; has {
		return ls.TokenStream(), nil
	}

	newL, err := m.l.Run(name, m)
	if err != nil {
		return nil, err
	}

	m.lexedSources[name] = newL
	return newL.TokenStream(), nil
}

func (m *mockDependencies) GetSourceFor(name string) (*bytes.Reader, error) {
	if buf, has := m.sources[name]; has {
		return bytes.NewReader([]byte(buf)), nil
	}
	return nil, fmt.Errorf("no such source: %s", name)
}
