package lexer

import (
	"reflect"
	"testing"
)

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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer([]byte(tt.input))

			t.Log(tt.input)
			if err := l.Run(); (err != nil) != tt.wantErr {
				t.Errorf("Lexer.Run() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(tt.wantTokens) != len(l.tokens) {
				t.Errorf("Lexer returned wrong number of tokens: expected = %d / got = %d", len(tt.wantTokens), len(l.tokens))
			}

			for i, got := range l.tokens {
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

func TestLexer_Position(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantTokens    []TokenType
		wantPositions []PositionRange
	}{
		{
			name:  "one string",
			input: `"a"`,

			wantTokens: []TokenType{TypeString},
			wantPositions: []PositionRange{
				{Position{1, 0, "", 0}, Position{1, 3, "", 3}},
			},
		},
		{
			name:  "unicode",
			input: `"👍"`,

			wantTokens: []TokenType{TypeString},
			wantPositions: []PositionRange{
				{Position{1, 0, "", 0}, Position{1, 3, "", 6}},
			},
		},
		{
			name:  "two string",
			input: `'string' 'string'`,

			wantTokens: []TokenType{TypeString, TypeString},
			wantPositions: []PositionRange{
				{Position{1, 0, "", 0}, Position{1, 8, "", 8}},
				{Position{1, 9, "", 9}, Position{1, 17, "", 17}},
			},
		},
		{
			name:  "newlines",
			input: "'string'\n'string'",

			wantTokens: []TokenType{TypeString, TypeTerminator, TypeString},
			wantPositions: []PositionRange{
				{Position{1, 0, "", 0}, Position{1, 8, "", 8}},
				{Position{1, 8, "", 8}, Position{1, 9, "", 9}},
				{Position{2, 0, "", 9}, Position{2, 8, "", 17}},
			},
		},
		{
			name:  "newlines unicode",
			input: "'string'\n'st👍'",

			wantTokens: []TokenType{TypeString, TypeTerminator, TypeString},
			wantPositions: []PositionRange{
				{Position{1, 0, "", 0}, Position{1, 8, "", 8}},
				{Position{1, 8, "", 8}, Position{1, 9, "", 9}},
				{Position{2, 0, "", 9}, Position{2, 5, "", 17}},
			},
		},
		{
			name:  "newlines multiunicode",
			input: "'st👍'\n'st👍👍👍'",

			wantTokens: []TokenType{TypeString, TypeTerminator, TypeString},
			wantPositions: []PositionRange{
				{Position{1, 0, "", 0}, Position{1, 5, "", 8}},
				{Position{1, 5, "", 8}, Position{1, 6, "", 9}},
				{Position{2, 0, "", 9}, Position{2, 7, "", 25}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer([]byte(tt.input))
			if err := l.Run(); err != nil {
				t.Fatalf("unexpected lex error: %s", err)
			}
			for i := range tt.wantPositions {
				tok := l.tokens[i]
				if !tok.Is(tt.wantTokens[i]) {
					t.Errorf("wrong token type")
				}
				if !reflect.DeepEqual(tok.startPosition, &tt.wantPositions[i].Start) {
					t.Errorf("bad start position for %s: got %v, want %v", tt.wantTokens[i], *tok.startPosition, tt.wantPositions[i].Start)
				}
				if !reflect.DeepEqual(tok.endPosition, &tt.wantPositions[i].End) {
					t.Errorf("bad end position for %s: got %v, want %v", tt.wantTokens[i], *tok.endPosition, tt.wantPositions[i].End)
				}
			}
		})
	}
}
