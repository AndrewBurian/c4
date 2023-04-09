package parser

import (
	"fmt"
	"strings"
	"unicode"

	"go.burian.dev/c4/cmd/compiler/internal/lexer"
)

// alias for string that can't have spaces
type IdentifierString string

func (p *Parser) acceptIdentifierString() bool {
	if !p.acceptOne(lexer.TypeIdentifier) {
		return false
	}

	cut := false
	str := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			r == '_' || r == '-' {
			return r
		}
		cut = true
		return -1
	}, p.currentSymbol())

	if cut {
		panic("identitifer string contains invalid characters")
	}
	idStr := IdentifierString(str)
	p.holdIdentifierForAssignment(idStr)
	return true
}

func (p *Parser) parseString() (string, error) {
	if !p.acceptOne(lexer.TypeString) {
		return "", p.errExpectedNext().Tokens(lexer.TypeString)
	}

	str := p.currentSymbol()
	if strings.ContainsRune(str, '\n') {
		return "", ErrorForToken(p.currentToken, fmt.Errorf("multiline string not allowed in this context"))
	}
	return p.cleanString(str), nil
}

func (p *Parser) cleanString(s string) string {
	s = strings.Trim(s, "\"'`\n\t\r")
	return s
}

/*
multiline strings have to start at an inline quote, but then will be
indented to match the code. e.g.

	{
		a = `
			this is a
			multi line string
			that has some indenting
		`

		b = `or this
			with the first line inline`
	}

	- skip to a non-empty line
	- if the first line has leading whitespace, note it
	- otherwise, take the second line's leading whitespace
	- strip that whitespace from all lines
*/
func (p *Parser) parseOneOrMoreLineString() (string, error) {
	if !p.acceptOne(lexer.TypeString) {
		return "", p.errExpectedNext().Tokens(lexer.TypeString)
	}
	str := string(p.currentSymbol())
	lines := strings.Split(str, "\n")
	if len(lines) < 2 {
		return str, nil
	}

	notSpace := func(r rune) bool { return !unicode.IsSpace(r) }

	// find a non-empty line
	firstCharacterLine := -1
	for i := range lines {
		if strings.IndexFunc(lines[i], notSpace) >= 0 {
			firstCharacterLine = i
			break
		}
	}

	if firstCharacterLine < 0 {
		// there are none, it's a giant empty string?
		return "", nil
	}

	// now find the first one that has a whitespace prefix
	prefixLine := -1

	// easy if it's the first line
	prefixIndex := strings.IndexFunc(lines[0], notSpace)
	if prefixIndex > 0 {
		prefixLine = 0
	}

	// if the first non-empty line wasn't the first, we can use that
	if firstCharacterLine > 0 {
		prefixIndex = strings.IndexFunc(lines[0], notSpace)
		prefixLine = firstCharacterLine
	}

	// otherwise we gotta go looking for the second non-empty line
	for i := range lines[1:] {
		prefixIndex = strings.IndexFunc(lines[i+1], notSpace)
		if prefixIndex > 0 {
			prefixLine = i + 1
			break
		}
	}

	// if there is none, just quit
	if prefixLine == -1 {
		// but at least drop the first empty lines
		return strings.Join(lines[firstCharacterLine:], "\n"), nil
	}

	// chop the prefix off everything
	prefix := lines[prefixLine][:prefixIndex]
	for i := range lines[firstCharacterLine:] {
		lines[i] = strings.TrimPrefix(lines[i], prefix)
	}

	// join it all back together
	str = strings.Join(lines[firstCharacterLine:], "\n")
	return p.cleanString(str), nil
}
