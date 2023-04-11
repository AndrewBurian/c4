package lexer

type TokenStream interface {
	NextToken() *Token
	BackupToken()
}

type LexedSource struct {
	tokens []*Token
}

func (l *LexedSource) TokenStream() TokenStream {
	return &stream{
		tokens:        l.tokens,
		tokenCursor:   -1,
		lastReadToken: nil,
	}
}

type stream struct {
	tokens        []*Token
	tokenCursor   int
	lastReadToken *Token
}

func (l *stream) NextToken() *Token {

	l.tokenCursor++

	if l.tokenCursor < len(l.tokens) {
		l.lastReadToken = l.tokens[l.tokenCursor]
	}

	return l.lastReadToken
}

func (l *stream) BackupToken() {
	if l.tokenCursor < 0 {
		return
	}
	l.tokenCursor--
}
