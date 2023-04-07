package parser

import (
	"testing"
	"unicode/utf8"
)

func Test_shrinkToAligned(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		size  int
	}{
		{
			name:  "straightforward",
			input: []byte("test"),
			size:  3,
		},
		{
			name:  "1 and 4",
			input: []byte("a👍🏾"),
			size:  3,
		},
		{
			name:  "no hope",
			input: []byte("👍🏾"),
			size:  3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			start := 0
			end := len(tt.input)
			shrinkToAligned(tt.input, tt.size, start, &end)
			t.Logf("start:%d end:%d", start, end)
			newString := tt.input[start:end]
			t.Logf("'%s' len %d", string(newString), len(newString))
			for i := 0; i < len(newString); {
				c, size := utf8.DecodeRune(newString[i:])
				if c == utf8.RuneError {
					t.Errorf("invalid utf8 character at offset %d", i)
					break
				}
				t.Logf("Got rune %q of size %d", c, size)
				i += size
			}
			if len(tt.input[start:end]) > tt.size {
				t.Error("wrong size")
			}
		})
	}
}
