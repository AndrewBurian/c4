package lexer

type TokenStream interface {
	NextToken() *Token
	BackupToken() error
}

func (l *Lexer) NextToken() *Token {

	l.tokenCursor++

	if l.tokenCursor < len(l.tokens) {
		l.lastReadToken = l.tokens[l.tokenCursor]
	}

	return l.lastReadToken
}

func (l *Lexer) BackupToken() {
	l.tokenCursor--
}
