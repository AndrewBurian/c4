package lexer

import (
	"reflect"
	"testing"
)

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
				{Position{1, 0, 0, ""}, Position{1, 3, 3, ""}},
			},
		},
		{
			name:  "unicode",
			input: `"👍"`,

			wantTokens: []TokenType{TypeString},
			wantPositions: []PositionRange{
				{Position{1, 0, 0, ""}, Position{1, 3, 6, ""}},
			},
		},
		{
			name:  "two string",
			input: `'string' 'string'`,

			wantTokens: []TokenType{TypeString, TypeString},
			wantPositions: []PositionRange{
				{Position{1, 0, 0, ""}, Position{1, 8, 8, ""}},
				{Position{1, 9, 9, ""}, Position{1, 17, 17, ""}},
			},
		},
		{
			name:  "newlines",
			input: "'string'\n'string'",

			wantTokens: []TokenType{TypeString, TypeTerminator, TypeString},
			wantPositions: []PositionRange{
				{Position{1, 0, 0, ""}, Position{1, 8, 8, ""}},
				{Position{1, 8, 8, ""}, Position{1, 9, 9, ""}},
				{Position{2, 0, 9, ""}, Position{2, 8, 17, ""}},
			},
		},
		{
			name:  "newlines unicode",
			input: "'string'\n'st👍'",

			wantTokens: []TokenType{TypeString, TypeTerminator, TypeString},
			wantPositions: []PositionRange{
				{Position{1, 0, 0, ""}, Position{1, 8, 8, ""}},
				{Position{1, 8, 8, ""}, Position{1, 9, 9, ""}},
				{Position{2, 0, 9, ""}, Position{2, 5, 17, ""}},
			},
		},
		{
			name:  "newlines multiunicode",
			input: "'st👍'\n'st👍👍👍'",

			wantTokens: []TokenType{TypeString, TypeTerminator, TypeString},
			wantPositions: []PositionRange{
				{Position{1, 0, 0, ""}, Position{1, 5, 8, ""}},
				{Position{1, 5, 8, ""}, Position{1, 6, 9, ""}},
				{Position{2, 0, 9, ""}, Position{2, 7, 25, ""}},
			},
		},
	}
	l := new(Lexer)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := &mockDependencies{
				sources: map[string]string{
					"": tt.input,
				},
			}
			out, err := l.Run("", deps)
			if err != nil {
				t.Fatalf("unexpected lex error: %s", err)
			}
			for i := range tt.wantPositions {
				tok := out.tokens[i]
				if !tok.Is(tt.wantTokens[i]) {
					t.Errorf("wrong token type")
				}
				if !reflect.DeepEqual(*tok.position, tt.wantPositions[i]) {
					t.Errorf("bad start position for %s: got (start: %v end: %v), want (start: %v end: %v)", tt.wantTokens[i],
						tok.position.Start, tok.position.End,
						tt.wantPositions[i].Start, tt.wantPositions[i].End)
				}
			}
		})
	}
}
