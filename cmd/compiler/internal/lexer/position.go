package lexer

type Position struct {
	Line   int
	Column int

	File string

	ByteOffset int
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
