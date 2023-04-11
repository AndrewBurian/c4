package lexer

import (
	"bytes"
	"fmt"
	"testing"
)

type mockDependencies struct {
	sources map[string]string
}

var _ Provider = &mockDependencies{}

func (m *mockDependencies) GetSourceFor(name string) (*bytes.Reader, error) {
	if buf, has := m.sources[name]; has {
		return bytes.NewReader([]byte(buf)), nil
	}
	return nil, fmt.Errorf("no such source: %s", name)
}

func TestLexer_Run(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantTokens []TokenType
		wantErr    bool
	}{
		{
			name:       "string",
			input:      `"this is a string"`,
			wantTokens: []TokenType{TypeString, TypeTerminator, TypeEOF},
		},
		{
			name:       "two strings",
			input:      `"this is a string"  "this is another"`,
			wantTokens: []TokenType{TypeString, TypeString, TypeTerminator, TypeEOF},
		},
		{
			name:       "mixed string quotes",
			input:      `'yay for "me"'`,
			wantTokens: []TokenType{TypeString, TypeTerminator, TypeEOF},
		},
		{
			name:       "escaped quote",
			input:      `"still one \"string\""`,
			wantTokens: []TokenType{TypeString, TypeTerminator, TypeEOF},
		},
		{
			name:       "escaped escape",
			input:      `"\\"`,
			wantTokens: []TokenType{TypeString, TypeTerminator, TypeEOF},
		},
		{
			name:       "identifiers",
			input:      `"this is a string"  thisIsAnIdentifier "and another string"`,
			wantTokens: []TokenType{TypeString, TypeIdentifier, TypeString, TypeTerminator, TypeEOF},
		},
		{
			name:       "comments",
			input:      `"this is a string"  thisIsAnIdentifier // this all gets ignored`,
			wantTokens: []TokenType{TypeString, TypeIdentifier, TypeTerminator, TypeEOF},
		},
		{
			name:       "lots of tokens",
			input:      `"string val" id = { } //comment`,
			wantTokens: []TokenType{TypeString, TypeIdentifier, TypeAssignment, TypeStartBlock, TypeEndBlock, TypeEOF},
		},
		{
			name: "block comment",
			input: `
				{
					/* this is a
					block comment */
				}
			`,
			wantTokens: []TokenType{TypeStartBlock, TypeEndBlock, TypeEOF},
		},
		{
			name: "inline block comment",
			input: `
				{
					"foo" /* this is a block comment */"bar"
				}
			`,
			wantTokens: []TokenType{TypeStartBlock, TypeString, TypeString, TypeTerminator, TypeEndBlock, TypeEOF},
		},
		{
			name: "lexically insert semi-colons",
			input: `baz foo bar
			foo "yay for me"`,
			wantTokens: []TokenType{TypeIdentifier, TypeIdentifier, TypeIdentifier, TypeTerminator, TypeIdentifier, TypeString, TypeTerminator, TypeEOF},
		},
		{
			name:       "handle actual semi-colons",
			input:      `baz foo bar; foo "yay for me";`,
			wantTokens: []TokenType{TypeIdentifier, TypeIdentifier, TypeIdentifier, TypeTerminator, TypeIdentifier, TypeString, TypeTerminator, TypeEOF},
		},
		{
			name:       "identity keywords",
			input:      `workspace "workspace" model foobar`,
			wantTokens: []TokenType{TypeKeyword, TypeString, TypeKeyword, TypeIdentifier, TypeTerminator, TypeEOF},
		},
		{
			name: "real workspace",
			input: `
				workspace foo {
					description "I'm a real boy now"
					model {}
					views {}
				}
			`,
			wantTokens: []TokenType{
				TypeKeyword, TypeIdentifier, TypeStartBlock,
				TypeKeyword, TypeString, TypeTerminator,
				TypeKeyword, TypeStartBlock, TypeEndBlock,
				TypeKeyword, TypeStartBlock, TypeEndBlock,
				TypeEndBlock, TypeEOF,
			},
		},
		{
			name:       "pragmas",
			input:      `"foo" #include_file`,
			wantTokens: []TokenType{TypeString, TypePragma, TypeEOF},
		},
	}

	l := new(Lexer)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			deps := &mockDependencies{
				sources: map[string]string{
					"main.c4": tt.input,
				},
			}

			t.Log(tt.input)
			out, err := l.Run("main.c4", deps)
			if (err != nil) != tt.wantErr {
				t.Errorf("Lexer.Run() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(tt.wantTokens) != len(out.tokens) {
				t.Errorf("Lexer returned wrong number of tokens: expected = %d / got = %d", len(tt.wantTokens), len(l.tokens))
				t.Log(l.tokens)
			}

			for i, got := range out.tokens {
				if i >= len(tt.wantTokens) {
					break
				}

				if !l.tokens[i].Is(tt.wantTokens[i]) {
					t.Errorf("Wrong token type at %d: expected '%s' but got '%s'", i, tt.wantTokens[i], got.tokenType)
				}

				if got.Is(TypeError) {
					t.Errorf("Lexing error: %s", got.err)
				}
			}
		})
	}
}
