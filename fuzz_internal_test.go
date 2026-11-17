package gofpdf

import "testing"

// The parsers below read data that can come from anywhere: whatever they are
// given, they must return, with an error or without it, and never fail.

func FuzzPathParse(f *testing.F) {
	for _, seed := range []string{
		"M1.5.5-2,3l.5-.25",
		"M0 0C1 2 3 4 5 6S9 10 11 12s4 4 6 6",
		"M0 0Q1 2 3 4T7 8t4 4",
		"M10 0a10 10 0 01-10 10A20 10 90 1 0 0 20z",
		"M10 10h5v5zm1 1h2V3H1",
		"M1e1 2E-1L-1.5e+1 0",
		"",
		"M",
		"M0 0A",
		"5 5",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, d string) {
		segs, err := pathParse(d)
		if err != nil {
			return
		}
		for _, seg := range segs {
			switch seg.Cmd {
			case 'M', 'L', 'H', 'V', 'C', 'Q', 'Z':
			default:
				t.Fatalf("path %q: unexpected command %q in the result", d, seg.Cmd)
			}
		}
	})
}

func FuzzParsePairKerning(f *testing.F) {
	f.Add(kerningFixture())
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, table []byte) {
		kern := parsePairKerning(table)
		if kern == nil {
			return
		}
		for first := 9; first < 15; first++ {
			for second := 19; second < 32; second++ {
				kern.adjustment(first, second)
			}
		}
	})
}
