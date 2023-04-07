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

func (p1 *Position) SetTo(p2 *Position) {
	p1.ByteOffset = p2.ByteOffset
	p1.Line = p2.Line
	p1.Column = p2.Column
}

func (p *Position) Offset(cols, bytes int) {
	p.ByteOffset += bytes
	p.Column += cols
}
