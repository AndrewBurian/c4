package lexer

import "fmt"

type Token struct {
	tokenType TokenType
	position  *PositionRange
	err       error
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
	return code[t.position.Start.ByteOffset:t.position.End.ByteOffset]
}

func (t *Token) Positions() *PositionRange {
	return t.position.Clone()
}

func (t *Token) String() string {
	if t.err != nil {
		return fmt.Sprintf("Error(%s) at %s", t.err, t.position)
	}
	return fmt.Sprintf("%s at %s", t.Type(), t.position)
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
	TypePragma

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
	case TypePragma:
		return "#Pragma"

	case TypeUndefined:
		return "[UNDEFINED TOKEN]"

	}
	panic("unknown token type")
}
