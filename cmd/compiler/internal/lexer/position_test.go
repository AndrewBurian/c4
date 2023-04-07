package lexer

import "testing"

func Test_PositionRangeClone(t *testing.T) {
	p := &PositionRange{
		Start: Position{
			Line:       1,
			Column:     1,
			ByteOffset: 1,
			File:       "a",
		},
	}

	b := p.Clone()

	p.Start.Line++
	p.Start.Column++
	p.Start.ByteOffset++
	p.Start.File = "b"

	t.Log(p, b)
}
