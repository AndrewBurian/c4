package parser

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"go.burian.dev/c4/compiler/lexer"
)

const (
	errorDecorator = "~ ~ ~ ~ ~ "
	tokenDecorator = "^"
	tabReplacement = " "
)

type ExpectationError struct {
	gotToken   *lexer.Token
	gotKeyword Keyword

	tokenTypes []lexer.TokenType
	keywords   []Keyword

	message string
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

	// code context
	ee.message = writeCodeContext(p.code, t, 80, 3)
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

func writeCodeContext(code []byte, tok *lexer.Token, limit, contextLines int) string {
	buf := new(strings.Builder)
	start, end := tok.Positions()

	// File declaration
	buf.WriteByte('\n')
	if start.File != "" {
		fmt.Fprintf(buf, "In file %s\n", start.File)
	}

	const lineFormat = "Line % -4d> %s\n"

	startCode := 0
	lineStarts := make([]int, 0, contextLines)
	lineStarts = append(lineStarts, 0)

	for {
		nextLine := strings.IndexRune(string(code[startCode:]), '\n') + 1
		if nextLine == 0 {
			break
		}
		if nextLine+startCode >= start.ByteOffset {
			break
		}

		startCode = nextLine + startCode
		lineStarts = append(lineStarts, startCode)
	}

	lineNo := 0
	for i := contextLines; i > 0; i-- {
		lineNo = len(lineStarts) - i - 1
		if lineNo >= 0 && lineNo < start.Line-1 {
			l := string(code[lineStarts[lineNo] : lineStarts[lineNo+1]-1])
			fmt.Fprintf(buf, lineFormat, lineNo+1, strings.ReplaceAll(l, "\t", tabReplacement))
		}
	}

	endCode := end.ByteOffset

	// expand to cover the rest of the line
	// unless we start just after the end of a line
	//if code[end.ByteOffset-1] != '\n' {
	for _ = endCode; endCode < len(code); endCode++ {
		if code[endCode-1] == '\n' {
			break
		}
	}
	//}

	// print the offending line
	if startCode == endCode {
		startCode = lineStarts[len(lineStarts)-1]
	}
	codeLine := string(code[startCode:endCode])
	codeLine = strings.ReplaceAll(codeLine, "\t", tabReplacement)
	codeLine = strings.TrimRight(codeLine, "\n\t ")
	fmt.Fprintf(buf, "Line % -4d> %s\n", start.Line, codeLine)

	// problem token highlight

	// pre-indicator fill
	lineToFill := start.Column + 11
	if spaces := lineToFill - len(errorDecorator); spaces > 0 {
		buf.WriteString(strings.Repeat(" ", spaces))
		lineToFill -= spaces
	}
	buf.WriteString(errorDecorator[len(errorDecorator)-lineToFill:])

	// token indicator
	symbolLen := end.Column - start.Column
	if symbolLen < 1 {
		symbolLen = 2
	}
	buf.WriteString(strings.Repeat("^", symbolLen))
	buf.WriteString(" Here\n")

	// post-context
	lineStarts = lineStarts[:0]
	lineStarts = append(lineStarts, endCode)
	startCode = endCode
	for i := 0; i < contextLines; i++ {
		nextLine := strings.IndexRune(string(code[startCode:]), '\n') + 1

		if nextLine == 0 {
			break
		}
		startCode = nextLine + startCode
		lineStarts = append(lineStarts, startCode)

	}
	lineStarts = append(lineStarts, len(code))

	lineNo = end.Line + 1
	for i := 0; i < len(lineStarts)-1; i++ {
		l := string(code[lineStarts[i] : lineStarts[i+1]-1])
		fmt.Fprintf(buf, lineFormat, lineNo+i, strings.ReplaceAll(l, "\t", tabReplacement))
	}

	return buf.String()
}

// moves the given end offset back until len(input[start:end]) <= size
// but ensures the end of input isn't a partial UTF8 sequence
func shrinkToAligned(input []byte, size, start int, end *int) {
	if *end-start > size {
		*end = start + size
		if *end > len(input) {
			*end = len(input)
			return
		}
	}
	for {
		pointsTrimmed := len(input) - *end
		b0 := input[*end] & 0b1111_0000
		if b0 < utf8.RuneSelf {
			return
		}
		if b0 >= 0xF0 && pointsTrimmed > 3 {
			return
		}
		if b0 >= 0xE0 && pointsTrimmed > 2 {
			return
		}
		if b0 >= 0xC0 && pointsTrimmed > 1 {
			return
		}
		*end--
	}
}
