package gofpdf

import "testing"

// words encodes the values as consecutive 16-bit big-endian numbers, the way
// the tables of a font are stored.
func words(vals ...int) []byte {
	buf := make([]byte, 0, 2*len(vals))
	for _, v := range vals {
		buf = append(buf, byte(v>>8), byte(v))
	}
	return buf
}

// kerningFixture returns a GPOS table written by hand. The comment of each
// part gives the position where it starts; the offsets stored in the table
// are relative to the start of the part that holds them.
//
// The table has three lookups and two features: "liga" uses lookup 0 and
// "kern" uses lookups 1 and 2.
func kerningFixture() []byte {
	var t []byte
	add := func(vals ...int) {
		t = append(t, words(vals...)...)
	}
	// 0: header: version 1.0, script list (not used), feature list at 10,
	// lookup list at 38
	add(1, 0, 0, 10, 38)
	// 10: feature list: "liga" at 10+14, "kern" at 10+20
	add(2, 0x6C69, 0x6761, 14, 0x6B65, 0x726E, 20)
	// 24: feature "liga": lookup 0
	add(0, 1, 0)
	// 30: feature "kern": lookups 2 and 1
	add(0, 2, 2, 1)
	// 38: lookup list: lookups at 38+8, 38+40 and 38+160
	add(3, 8, 40, 160)

	// 46: lookup 0: pair adjustment with one subtable at 46+8
	add(2, 0, 1, 8)
	// 54: pairs listed one by one; coverage at 54+18, the values hold only
	// the advance, one set of pairs at 54+12
	add(1, 18, 4, 0, 1, 12)
	// 66: pairs of glyph 10: with glyph 20, -999
	add(1, 20, -999)
	// 72: coverage: glyph 10
	add(1, 1, 10)

	// 78: lookup 1: pair adjustment with subtables at 78+10 and 78+56
	add(2, 0, 2, 10, 56)
	// 88: pairs listed one by one; coverage at 88+36, the values hold a
	// placement and the advance, two sets of pairs at 88+14 and 88+28
	add(1, 36, 5, 0, 2, 14, 28)
	// 102: pairs of glyph 10: with glyph 20, -50; with glyph 21, -30
	add(2, 20, 7, -50, 21, 0, -30)
	// 116: pairs of glyph 11: with glyph 20, +40
	add(1, 20, 0, 40)
	// 124: coverage: the range of glyphs 10 to 11
	add(2, 1, 10, 11, 0)
	// 134: pairs by class; coverage at 134+28, the values hold only the
	// advance, classes of the first glyph at 134+38, classes of the second
	// glyph at 134+48, 2 classes by 3 classes
	add(2, 28, 4, 0, 38, 48, 2, 3)
	// 150: first glyph of class 0 followed by classes 0, 1 and 2, and then
	// first glyph of class 1. Class 0 is the class of the glyphs that the
	// class definition does not list.
	add(-3, -10, -20)
	add(-4, -70, -80)
	// 162: coverage: glyphs 10, 12 and 13
	add(1, 3, 10, 12, 13)
	// 172: classes of the first glyph: from glyph 12 on, classes 1 and 0
	add(1, 12, 2, 1, 0)
	// 182: classes of the second glyph: glyphs 20 to 21 are class 1, glyph
	// 22 is class 2
	add(2, 2, 20, 21, 1, 22, 22, 2)

	// 198: lookup 2: extension with one subtable at 198+8
	add(9, 0, 1, 8)
	// 206: extension: a pair adjustment 8 bytes ahead (32-bit offset)
	add(1, 2, 0, 8)
	// 214: pairs listed one by one; coverage at 214+22, the values hold
	// only the advance, one set of pairs at 214+12
	add(1, 22, 4, 0, 1, 12)
	// 226: pairs of glyph 10: with glyph 20, -5; with glyph 30, -6
	add(2, 20, -5, 30, -6)
	// 236: coverage: glyph 10
	add(1, 1, 10)
	return t
}

func TestPairKerning_adjustment(t *testing.T) {
	kern := parsePairKerning(kerningFixture())
	if kern == nil {
		t.Fatal("the table has a kern feature but no kerning was read")
	}
	tests := []struct {
		name          string
		first, second int
		want          int
	}{
		// Lookup 1 gives -50 and lookup 2 gives -5. The second subtable of
		// lookup 1 (-10 by class) is not used because the first one has the
		// pair, and lookup 0 (-999) does not belong to the kern feature.
		{"every lookup of the feature adds up", 10, 20, -55},
		{"pair listed in the first subtable", 10, 21, -30},
		{"pair of the second covered glyph", 11, 20, 40},
		{"pair by class", 12, 20, -70},
		{"pair by class, other class of the second glyph", 12, 22, -80},
		{"covered glyph of class 0", 13, 22, -20},
		{"falls through to the classes when the pair is not listed", 10, 22, -20},
		// Glyph 30 is not listed in any class, which makes it of class 0:
		// lookup 1 gives -3 by class and lookup 2 gives -6.
		{"second glyph of class 0, plus the extension", 10, 30, -9},
		{"second glyph of class 0", 12, 30, -4},
		{"first glyph not covered", 14, 20, 0},
		{"the order of the pair matters", 20, 10, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := kern.adjustment(tc.first, tc.second); got != tc.want {
				t.Errorf("glyphs %d and %d: got %d, want %d", tc.first, tc.second, got, tc.want)
			}
		})
	}
}

func TestPairKerning_tableWithoutKernFeature(t *testing.T) {
	table := kerningFixture()
	copy(table[18:22], "xxxx") // the tag of the second feature
	if kern := parsePairKerning(table); kern != nil {
		t.Errorf("got kerning from a table without a kern feature")
	}
}

// A font file cut short or damaged must not break the program that loads it:
// the font is then used without kerning.
func TestPairKerning_truncatedTable(t *testing.T) {
	table := kerningFixture()
	for size := 0; size < len(table); size++ {
		kern := parsePairKerning(table[:size])
		if kern != nil {
			kern.adjustment(10, 20)
			kern.adjustment(12, 22)
		}
	}
}

// Offsets and counts that point outside of the table are not followed.
func TestPairKerning_wildOffsets(t *testing.T) {
	for pos := 0; pos < len(kerningFixture()); pos += 2 {
		table := kerningFixture()
		table[pos], table[pos+1] = 0xFF, 0xFF
		if kern := parsePairKerning(table); kern != nil {
			kern.adjustment(10, 20)
			kern.adjustment(12, 22)
		}
	}
}
