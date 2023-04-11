package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"go.burian.dev/c4/cmd/compiler/internal/lexer"
)

type CodeError interface {
	TokenAtError() *lexer.Token
}

func (c *compiler) prettyPrintError(e error) string {

	var ce CodeError
	if !errors.As(e, &ce) {
		return e.Error()
	}

	buf := new(strings.Builder)
	buf.WriteString(e.Error())
	buf.WriteRune('\n')

	code, err := c.GetSourceFor(ce.TokenAtError().Positions().Start.File)
	if err != nil {
		panic("printing error message for unavailable DSL file")
	}

	writeCodeContext(buf, code, ce.TokenAtError(), 60, 4)

	return buf.String()
}

const (
	errorDecorator = "~ ~ ~ ~ ~ "
	tokenDecorator = "^"
	tabReplacement = " "
)

func writeCodeContext(buf *strings.Builder, source *bytes.Reader, tok *lexer.Token, limit, contextLines int) {
	pos := tok.Positions()
	code, err := io.ReadAll(source)
	if err != nil {
		panic("error reading byte buffer from source reader")
	}

	fileName := pos.Start.File

	const lineFormat = "%s: %03d > %s\n"

	startCode := 0
	lineStarts := make([]int, 0, contextLines)
	lineStarts = append(lineStarts, 0)

	for {
		nextLine := strings.IndexRune(string(code[startCode:]), '\n') + 1
		if nextLine == 0 {
			break
		}
		if nextLine+startCode >= pos.Start.ByteOffset {
			break
		}

		startCode = nextLine + startCode
		lineStarts = append(lineStarts, startCode)
	}

	lineNo := 0
	for i := contextLines; i > 0; i-- {
		lineNo = len(lineStarts) - i - 1
		if lineNo >= 0 && lineNo < pos.Start.Line-1 {
			l := string(code[lineStarts[lineNo] : lineStarts[lineNo+1]-1])
			fmt.Fprintf(buf, lineFormat, fileName, lineNo+1, strings.ReplaceAll(l, "\t", tabReplacement))
		}
	}

	endCode := pos.End.ByteOffset

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
	fmt.Fprintf(buf, lineFormat, fileName, pos.Start.Line, codeLine)

	// problem token highlight

	// pre-indicator fill
	// print another line but without the newline
	fmt.Fprintf(buf, lineFormat[0:len(lineFormat)-3], fileName, pos.Start.Line)
	lineToFill := pos.Start.Column
	if spaces := lineToFill - len(errorDecorator); spaces > 0 {
		buf.WriteString(strings.Repeat(" ", spaces))
		lineToFill -= spaces
	}
	buf.WriteString(errorDecorator[len(errorDecorator)-lineToFill:])

	// token indicator
	symbolLen := pos.End.Column - pos.Start.Column
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

	lineNo = pos.End.Line + 1
	for i := 0; i < len(lineStarts)-1; i++ {
		l := string(code[lineStarts[i] : lineStarts[i+1]-1])
		fmt.Fprintf(buf, lineFormat, fileName, lineNo+i, strings.ReplaceAll(l, "\t", tabReplacement))
	}

	return
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
