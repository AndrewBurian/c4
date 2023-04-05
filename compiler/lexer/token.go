package lexer

import "fmt"

type Token struct {
	tokenType     TokenType
	startPosition *Position
	endPosition   *Position
	err           error
}

func (t *Token) Is(ty ...TokenType) bool {
	if t == nil {
		return false
	}
	for i := range ty {
		if ty[i] == t.tokenType {
			return true
		}
	}
	return false
}

func (t *Token) Type() TokenType {
	return t.tokenType
}

func (t *Token) BytesAt(code []byte) []byte {
	return code[t.startPosition.ByteOffset:t.endPosition.ByteOffset]
}

func (t *Token) Positions() (start, end *Position) {
	start, end = new(Position), new(Position)
	start.SetTo(t.startPosition)
	end.SetTo(t.endPosition)
	return
}

func (t *Token) CodeLineAt(code []byte) string {
	byteStart := t.startPosition.ByteOffset - t.startPosition.Column
	byteEnd := t.endPosition.ByteOffset

	return string(code[byteStart:byteEnd])
}

func (t *Token) String() string {
	if t.err == nil {
		return t.Type().String()
	}
	return fmt.Sprintf("Error(%s)", t.err)
}

type TokenType int

const (
	TypeError = TokenType(iota - 1)
	TypeUndefined

	TypeKeyword
	TypeString
	TypeIdentifier
	TypeAssignment
	TypeStartBlock
	TypeEndBlock
	TypeDirective
	TypeTerminator
	TypeRelationship

	TypeEOF
)

func (t TokenType) String() string {
	switch t {
	case TypeKeyword:
		return "Keyword"
	case TypeError:
		return "Error"
	case TypeString:
		return "String"
	case TypeIdentifier:
		return "Identifier"
	case TypeAssignment:
		return "'='"
	case TypeStartBlock:
		return "'{'"
	case TypeEndBlock:
		return "'}'"
	case TypeDirective:
		return "'!'"
	case TypeRelationship:
		return `"->"`
	case TypeTerminator:
		return "Newline or terminator (';')"
	case TypeEOF:
		return "End of File"

	case TypeUndefined:
		return "[UNDEFINED TOKEN]"

	}
	panic("unknown token type")
}
