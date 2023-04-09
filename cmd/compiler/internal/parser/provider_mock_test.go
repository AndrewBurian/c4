package parser

import (
	"bytes"
	"fmt"

	"go.burian.dev/c4/cmd/compiler/internal/lexer"
)

type mockDependencies struct {
	sources map[string]string
	lexers  map[string]*lexer.Lexer
	l       *lexer.Lexer
}

var _ Provider = &mockDependencies{}

func (m *mockDependencies) GetTokenStreamFor(name string) (lexer.TokenStream, error) {
	if l, has := m.lexers[name]; has {
		return l.TokenStream(), nil
	}
	if _, has := m.sources[name]; !has {
		return nil, fmt.Errorf("no data file")
	}
	newL := new(lexer.Lexer)
	if err := newL.Run(name, m); err != nil {
		return nil, err
	}
	m.lexers[name] = newL
	return newL.TokenStream(), nil
}

func (m *mockDependencies) GetSourceFor(name string) (*bytes.Reader, error) {
	if buf, has := m.sources[name]; has {
		return bytes.NewReader([]byte(buf)), nil
	}
	return nil, fmt.Errorf("no such source: %s", name)
}
