package lexer

import "fmt"

type Position struct {
	Line   int
	Column int

	ByteOffset int

	File string
}

type PositionRange struct {
	Start Position
	End   Position
}

func (p *PositionRange) truncateForward() {
	p.Start.ByteOffset = p.End.ByteOffset
	p.Start.Line = p.End.Line
	p.Start.Column = p.End.Column
	p.Start.File = p.End.File
}

func (p *PositionRange) Clone() *PositionRange {
	n := new(PositionRange)
	*n = *p
	return n
}

func (pr *PositionRange) String() string {
	return fmt.Sprintf("%s (line %d col %d + next %d bytes)",
		pr.Start.File,
		pr.Start.Line, pr.Start.Column,
		pr.End.ByteOffset-pr.Start.ByteOffset,
	)
}
