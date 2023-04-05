package lexer

import (
	"fmt"
	"strings"
	"unicode"
)

type stateFn func(*Lexer) stateFn

func rootState(l *Lexer) stateFn {
	for {
		l.next()

		switch l.currentRune {
		case '\'', '"', '`':
			return stringState
		case '=':
			l.createToken(TypeAssignment)
			continue
		case '{':
			l.createToken(TypeStartBlock)
			continue
		case '}':
			l.createToken(TypeEndBlock)
			continue
		case '#':
			l.createError(fmt.Errorf("comments beginning with '#' are not allowed in this grammar, use '//'"))
			return clearLineState
		case '!':
			l.createToken(TypeDirective)
			continue
		case '/':
			if l.acceptOne('/') {
				return lineCommentState
			}
			if l.acceptOne('*') {
				return blockCommentState
			}
		case '-':
			if l.acceptOne('>') {
				l.createToken(TypeRelationship)
				continue
			}
		case ';':
			l.createToken(TypeTerminator)
			continue
		case EOF:
			if l.previousToken.Is(TypeIdentifier, TypeString) {
				l.createToken(TypeTerminator)
			}
			l.createToken(TypeEOF)
			return nil
		}

		if unicode.IsSpace(l.currentRune) {
			return spaceState
		}

		if l.currentRune >= 'a' && l.currentRune <= 'z' {
			return identifierState
		}

		l.createError(fmt.Errorf("unexpected token %q", l.currentRune))
		return nil
	}
}

func spaceState(l *Lexer) stateFn {
	if l.previousToken.Is(TypeIdentifier, TypeString) {
		return spaceWithOptionalTerminatorState
	}
	l.acceptWhile(unicode.IsSpace)
	l.discardToCurrent()
	return rootState
}

func spaceWithOptionalTerminatorState(l *Lexer) stateFn {
	for {
		if l.acceptOne('\n', '\t', '\v', '\f', '\r', ' ') {
			if l.currentRune != '\n' {
				continue
			}

			l.backupOne()
			l.discardToCurrent()
			l.next()
			l.createToken(TypeTerminator)
			return rootState
		}

		if l.acceptOne(EOF) {
			l.backupOne()
			l.discardToCurrent()
			l.createToken(TypeTerminator)
			return rootState
		}

		l.discardToCurrent()
		return rootState
	}
}

func lineCommentState(l *Lexer) stateFn {
	l.acceptWhileNot('\n', EOF)
	l.discardToCurrent()
	return spaceState
}

func blockCommentState(l *Lexer) stateFn {
	for {
		l.acceptWhileNot('*', EOF)
		if l.acceptOne('*') {
			if l.acceptOne('/') {
				l.discardToCurrent()
				return rootState
			}
		}
		if l.acceptOne(EOF) {
			l.createError(fmt.Errorf("unexpected EOF: expected end of block comment"))
			l.backupOne()
			break
		}
	}
	l.discardToCurrent()
	return rootState
}

func stringState(l *Lexer) stateFn {
	quote := l.currentRune

	// consume all characters
	l.acceptWhile(func(r rune) bool {
		if r == EOF {
			return false
		}

		if r == '\n' && quote != '`' {
			return false
		}

		if r == quote {
			return l.previousRune == '\\'
		}

		return true
	})

	if l.acceptOne(EOF) {
		l.createError(fmt.Errorf("unexpected EOF: expected end of string"))
		l.backupOne()
		return rootState
	}
	if l.acceptOne(quote) {
		l.createToken(TypeString)
		return spaceState
	}

	r := l.next()
	l.createError(fmt.Errorf("unexpected value in string: %q", r))
	l.backupOne()
	return rootState
}

func multilineStringState(l *Lexer) stateFn {
	l.acceptWhile(func(r rune) bool {
		return r != '`'
	})

	if l.currentRune != '`' {
		l.createError(fmt.Errorf("unexpected end of multiline string"))
		return rootState
	}

	l.createToken(TypeString)
	return rootState
}

func identifierState(l *Lexer) stateFn {
	builder := new(strings.Builder)
	builder.WriteRune(l.currentRune)
	l.acceptWhile(func(r rune) bool {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r == '_' || r == '.' || r == '-') {
			builder.WriteRune(r)
			return true
		}
		return false
	})

	identifier := builder.String()

	if lastChracter := identifier[len(identifier)-1]; lastChracter == '.' ||
		lastChracter == '-' || lastChracter == '_' {
		l.createError(fmt.Errorf("illegal identifier suffix character '%c'", lastChracter))
	}

	if isKeyword(identifier) {
		l.createToken(TypeKeyword)
	} else {
		l.createToken(TypeIdentifier)
	}
	return spaceWithOptionalTerminatorState
}

func clearLineState(l *Lexer) stateFn {
	l.acceptWhileNot('\n', EOF)
	l.discardToCurrent()
	return spaceState
}
