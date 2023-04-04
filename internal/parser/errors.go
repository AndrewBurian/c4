package parser

import (
	"fmt"
	"strings"

	"go.burian.dev/c4arch/internal/lexer"
)

const errorDecorator = "~ ~ ~ ~ ~ "

type ExpectationError struct {
	gotToken   *lexer.Token
	gotKeyword Keyword

	tokenTypes []lexer.TokenType
	keywords   []Keyword

	message string
}

func (p *Parser) errExpectedNext() *ExpectationError {
	ee := new(ExpectationError)
	p.nextToken()
	ee.gotToken = p.currentToken
	p.backupToken()
	if ee.gotToken.Is(lexer.TypeKeyword) {
		ee.gotKeyword = p.currentKeyword()
	}

	buf := new(strings.Builder)

	start, end := ee.gotToken.Positions()

	fmt.Fprintf(buf, "\nIn file %s\n", p.currentFile)
	codeLine := string(p.code[start.ByteOffset-start.Column : end.ByteOffset])
	codeLine = strings.ReplaceAll(codeLine, "\t", " ")
	fmt.Fprintf(buf, "Line % -4d> %s\n", start.Line, codeLine)
	lineToFill := start.Column + 11
	if spaces := lineToFill - len(errorDecorator); spaces > 0 {
		buf.WriteString(strings.Repeat(" ", spaces))
		lineToFill -= spaces
	}
	buf.WriteString(errorDecorator[len(errorDecorator)-lineToFill:])
	symbolLen := end.Column - start.Column
	if symbolLen < 1 {
		symbolLen = 2
	}
	buf.WriteString(strings.Repeat("^", symbolLen))
	buf.WriteString(" Here\n")

	ee.message = buf.String()

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
		fmt.Fprintf(buf, "got keyword %s but expected ", ee.gotKeyword)
	} else {
		fmt.Fprintf(buf, "got %s but expected ", ee.gotToken)
	}

	if len(ee.tokenTypes) > 0 {
		plural := ""
		if len(ee.tokenTypes) > 1 {
			plural = "s"
		}
		fmt.Fprintf(buf, "token type%s: ", plural)
		for i := range ee.tokenTypes {
			fmtStr := ""
			if i < len(ee.tokenTypes)-2 {
				fmtStr = "'%s', "
			} else {
				fmtStr = "or '%s'"
			}
			fmt.Fprintf(buf, fmtStr, ee.tokenTypes[i])
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
		fmt.Fprintf(buf, "keyword%s: ", plural)
		for i := range ee.keywords {
			fmtStr := ""
			if i < len(ee.keywords)-2 {
				fmtStr = "'%s', "
			} else if i == 0 {
				fmtStr = "'%s' "
			} else {
				fmtStr = "or '%s'"
			}
			fmt.Fprintf(buf, fmtStr, ee.keywords[i])
		}
	}

	buf.WriteString(ee.message)

	return buf.String()
}
