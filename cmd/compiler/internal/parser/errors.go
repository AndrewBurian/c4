package parser

import (
	"fmt"
	"strings"

	"go.burian.dev/c4/cmd/compiler/internal/lexer"
)

const (
	errorDecorator = "~ ~ ~ ~ ~ "
	tokenDecorator = "^"
	tabReplacement = " "
)

type CodeError struct {
	offendingToken *lexer.Token
	wrappedErr     error
}

func (ce *CodeError) Error() string {
	if ce.wrappedErr != nil {
		return fmt.Sprintf("%s at %s", ce.wrappedErr.Error(), ce.offendingToken)
	}
	return fmt.Sprintf("at %s", ce.offendingToken)
}

func (ce *CodeError) Wrap(e error) {
	ce.wrappedErr = e
}

func (ce *CodeError) Unwrap() error {
	return ce.wrappedErr
}

func (ce *CodeError) TokenAtError() *lexer.Token {
	return ce.offendingToken
}

func ErrorForToken(t *lexer.Token, e error) error {
	return &CodeError{
		offendingToken: t,
		wrappedErr:     e,
	}

}

type ExpectationError struct {
	gotToken   *lexer.Token
	gotKeyword Keyword

	tokenTypes []lexer.TokenType
	keywords   []Keyword

	message string
}

func (ee *ExpectationError) TokenAtError() *lexer.Token {
	return ee.gotToken
}

func (p *Parser) errExpectedCurrent() *ExpectationError {
	return p.newExpectationErrFor(p.currentToken)
}

func (p *Parser) errExpectedNext() *ExpectationError {
	p.nextToken()
	t := p.currentToken
	p.backupToken()
	return p.newExpectationErrFor(t)
}

func (p *Parser) newExpectationErrFor(t *lexer.Token) *ExpectationError {

	ee := new(ExpectationError)
	ee.gotToken = t
	if ee.gotToken.Is(lexer.TypeKeyword) {
		ee.gotKeyword = p.currentKeyword()
	}

	return ee
}
func (ee *ExpectationError) Tokens(ts ...lexer.TokenType) *ExpectationError {
	ee.tokenTypes = append(ee.tokenTypes, ts...)
	return ee
}

func (ee *ExpectationError) Keywords(ks ...Keyword) *ExpectationError {
	ee.keywords = append(ee.keywords, ks...)
	return ee
}

func (ee ExpectationError) Error() string {
	buf := new(strings.Builder)
	if ee.gotKeyword != "" {
		fmt.Fprintf(buf, "got keyword '%s' but expected ", ee.gotKeyword)
	} else {
		fmt.Fprintf(buf, "got %s but expected ", ee.gotToken)
	}

	if len(ee.tokenTypes) > 0 {
		plural := ""
		if len(ee.tokenTypes) > 1 {
			plural = "s"
		}
		fmt.Fprintf(buf, "token type%s ", plural)
		if len(ee.tokenTypes) == 1 {
			fmt.Fprintf(buf, "%s", ee.tokenTypes[0])
		} else {
			for i := range ee.tokenTypes {
				fmtStr := ""
				if i < len(ee.tokenTypes)-1 {
					fmtStr = "%s, "
				} else if i == 0 {
					fmtStr = "%s"
				} else {
					fmtStr = "or %s"
				}
				fmt.Fprintf(buf, fmtStr, ee.tokenTypes[i])
			}
		}

		if len(ee.keywords) > 0 {
			fmt.Fprintf(buf, ", or ")
		}
	}
	if len(ee.keywords) > 0 {
		plural := ""
		if len(ee.keywords) > 1 {
			plural = "s"
		}
		fmt.Fprintf(buf, "keyword%s ", plural)
		if len(ee.keywords) == 1 {
			fmt.Fprintf(buf, "'%s'", ee.keywords[0])
		} else {
			for i := range ee.keywords {
				fmtStr := ""
				if i < len(ee.keywords)-1 {
					fmtStr = "'%s', "
				} else if i == 0 {
					fmtStr = "'%s'"
				} else {
					fmtStr = "or '%s'"
				}
				fmt.Fprintf(buf, fmtStr, ee.keywords[i])
			}
		}
	}

	buf.WriteString(ee.message)

	return buf.String()
}
