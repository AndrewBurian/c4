package lexer

import (
	"bytes"
	"errors"
	"io"
)

const EOF = rune(-1)

type Lexer struct {
	inputBuffer *bytes.Buffer
	input       []byte

	currentRune     rune
	currentRuneSize int
	canBackup       bool

	previousRune     rune
	previousLineCols int

	startPosition *Position
	endPosition   *Position

	atEOF bool

	state stateFn
	err   error

	lastReadToken *Token
	tokens        []*Token
	previousToken *Token
	tokenCursor   int
	peekToken     *Token
}

func NewLexer(input []byte) *Lexer {

	l := new(Lexer)

	l.input = input
	l.inputBuffer = bytes.NewBuffer(input)

	l.currentRune = 0
	l.startPosition = new(Position)
	l.endPosition = new(Position)
	l.startPosition.Line = 1
	l.startPosition.ByteOffset = 0
	l.startPosition.Column = 0

	l.endPosition.SetTo(l.startPosition)
	//l.endPosition.Offset(1, 1)

	l.state = rootState

	l.tokenCursor = -1

	return l
}

func (l *Lexer) Run() error {
	for l.state != nil {
		l.state = l.state(l)
	}

	return l.err
}

func (l *Lexer) NextToken() *Token {

	l.tokenCursor++

	if l.tokenCursor < len(l.tokens) {
		l.lastReadToken = l.tokens[l.tokenCursor]
	}

	if l.tokenCursor+1 < len(l.tokens) {
		l.peekToken = l.tokens[l.tokenCursor+1]
	}

	return l.lastReadToken
}

func (l *Lexer) BackupToken() {
	l.tokenCursor--
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

	l.endPosition.Column++
	l.endPosition.ByteOffset += s
	l.currentRuneSize = s
	if l.currentRune == '\n' {
		l.endPosition.Line++
		l.previousLineCols = l.endPosition.Column
		l.endPosition.Column = 1
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

	l.endPosition.ByteOffset = l.endPosition.ByteOffset - l.currentRuneSize
	l.endPosition.Column--
	l.currentRune = l.previousRune
	if l.currentRune == '\n' {
		l.endPosition.Line--
		l.endPosition.Column = l.previousLineCols

	}

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
		tokenType:     t,
		startPosition: new(Position),
		endPosition:   new(Position),
	}
	tok.startPosition.SetTo(l.startPosition)
	tok.endPosition.SetTo(l.endPosition)
	l.tokens = append(l.tokens, tok)
	l.previousToken = tok

	l.discardToCurrent()
}

func (l *Lexer) createError(err error) {
	tok := &Token{
		tokenType:     TypeError,
		err:           err,
		startPosition: new(Position),
		endPosition:   new(Position),
	}
	tok.startPosition.SetTo(l.startPosition)
	tok.endPosition.SetTo(l.endPosition)
	l.tokens = append(l.tokens, tok)
	l.previousToken = tok

	l.discardToCurrent()
}

func (l *Lexer) discardToCurrent() {
	if l.atEOF {
		return
	}

	l.startPosition.SetTo(l.endPosition)

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
		l.startPosition.Line++
		l.startPosition.Column = 0
	}
}
