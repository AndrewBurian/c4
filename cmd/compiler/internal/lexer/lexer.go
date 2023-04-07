package lexer

import (
	"bytes"
	"errors"
	"io"
)

const EOF = rune(-1)

type Lexer struct {
	inputBuffer *bytes.Reader

	currentRune     rune
	currentRuneSize int
	canBackup       bool
	atEOF           bool

	previousRune     rune
	previousLineCols int

	cursor *PositionRange

	state stateFn
	err   error

	tokens        []*Token
	lastReadToken *Token
	previousToken *Token
	tokenCursor   int
}

// Run clears the state of the lexer and starts it again with a new input
func (l *Lexer) Run(input *bytes.Reader) error {

	l.inputBuffer = input

	l.currentRune = 0
	l.atEOF = false

	l.canBackup = false
	l.previousRune = 0
	l.previousLineCols = 0

	l.cursor = new(PositionRange)
	l.cursor.End.Line = 1
	l.cursor.truncateForward()

	l.state = rootState
	l.err = nil

	l.tokenCursor = -1
	if l.tokens != nil {
		l.tokens = l.tokens[0:0]
	}

	for l.state != nil {
		l.state = l.state(l)
	}

	return l.err
}

func (l *Lexer) next() rune {

	if l.currentRune == EOF {
		panic("Advancing from EOF")
	}

	r, s, err := l.inputBuffer.ReadRune()
	if errors.Is(err, io.EOF) {
		r = EOF
		err = nil
		s = 0
		l.atEOF = true
	}
	if err != nil {
		panic(err)
	}

	l.cursor.End.Column++
	l.cursor.End.ByteOffset += s
	l.currentRuneSize = s
	if l.currentRune == '\n' {
		l.cursor.End.Line++
		l.previousLineCols = l.cursor.End.Column
		l.cursor.End.Column = 1
	}

	l.previousRune = l.currentRune
	l.canBackup = true
	l.currentRune = r

	return r
}

func (l *Lexer) backupOne() {
	if !l.canBackup {
		panic("unexpected backup")
	}

	if !l.atEOF {
		// unread rune will panic if last read was EOF
		if err := l.inputBuffer.UnreadRune(); err != nil {
			panic("unread rune error")
		}
	}

	l.cursor.End.ByteOffset = l.cursor.End.ByteOffset - l.currentRuneSize
	l.cursor.End.Column--
	l.currentRune = l.previousRune
	if l.currentRune == '\n' {
		l.cursor.End.Line--
		l.cursor.End.Column = l.previousLineCols
	}

	l.currentRune = l.previousRune
	l.previousRune = 0
	l.canBackup = false
	l.atEOF = false
}

func (l *Lexer) acceptOne(r ...rune) bool {
	n := l.next()
	for i := range r {
		if n == r[i] {
			return true
		}
	}
	l.backupOne()
	return false
}

func (l *Lexer) acceptWhile(f func(rune) bool) {
	n := l.next()

	for f(n) {
		n = l.next()
	}

	l.backupOne()
}

func (l *Lexer) acceptWhileNot(not ...rune) {
	l.acceptWhile(func(r rune) bool {
		for i := range not {
			if r == not[i] {
				return false
			}
		}
		return true
	})
}

func (l *Lexer) createToken(t TokenType) {
	tok := &Token{
		tokenType: t,
		position:  l.cursor.Clone(),
	}

	l.tokens = append(l.tokens, tok)
	l.previousToken = tok

	l.discardToCurrent()
}

func (l *Lexer) createError(err error) {
	tok := &Token{
		tokenType: TypeError,
		err:       err,
		position:  l.cursor.Clone(),
	}
	l.tokens = append(l.tokens, tok)
	l.previousToken = tok

	l.discardToCurrent()
}

func (l *Lexer) discardToCurrent() {
	if l.atEOF {
		return
	}

	l.cursor.truncateForward()

	/*
				If this is the end of the line, things get weird

				The end position cursor is the end point of a half-open range, it
				falls one character AFTER the token. So it includes a \n character without
				actually advancing to the next line since the \n is on the same line

				'foooooobar'\n
		        ^start        ^end

				When we advance the start cursor to it, it puts it at this position floating off
				the end of the line

				'foooooobar'\n
		                      ^end
							  ^start

				This isn't what we want, since the start cursor should always point to the
				first valid character. It is pointing to the correct byte, it's only broken in the
				conceptual line/column model. So we advance it here if needed to the next line.
				This means it's "leading" the end cursor by lines, but the end cursor will catch up on the next read

	*/
	if l.currentRune == '\n' {
		l.cursor.Start.Line++
		l.cursor.Start.Column = 0
	}
}
