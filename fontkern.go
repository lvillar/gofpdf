package gofpdf

import "math"

// Pair kerning of TrueType fonts: the adjustments of the space between
// specific pairs of glyphs that are stored in the "kern" feature of the GPOS
// table of the font.

// maxKernRecords limits the number of records read from a GPOS table. No real
// font comes close to it; it keeps a damaged file, in which every offset can
// point to the same data, from being read forever.
const maxKernRecords = 1 << 22

// pairKerning holds the pair adjustments of a font, in the design units of
// the font. Each lookup is a list of subtables.
type pairKerning struct {
	lookups [][]kernSubtable
}

// kernSubtable holds the horizontal adjustments of one pair adjustment
// subtable, which lists its pairs either one by one or by class of glyph.
type kernSubtable struct {
	// Pairs listed one by one. The key is the first glyph in the high 16
	// bits and the second one in the low 16 bits.
	pairs map[uint32]int16
	// Pairs by class: the first glyphs that the subtable applies to, the
	// class of each glyph in both positions (class 0 is left out) and the
	// adjustment for each combination of classes.
	covered                  map[uint16]bool
	class1, class2           map[uint16]uint16
	class1Count, class2Count int
	values                   []int16
}

// adjustment returns the adjustment between two glyphs. ok is false if the
// subtable does not apply to the pair, in which case the next subtable of the
// lookup is the one to ask.
func (st *kernSubtable) adjustment(first, second uint16) (val int16, ok bool) {
	if st.pairs != nil {
		val, ok = st.pairs[uint32(first)<<16|uint32(second)]
		return
	}
	if !st.covered[first] {
		return 0, false
	}
	// Once the first glyph is covered the subtable applies whatever the
	// second glyph is, even when the adjustment for their classes is zero:
	// the following subtables of the lookup are not asked (this is how
	// HarfBuzz, the text shaper of the browsers, reads these tables).
	c1, c2 := int(st.class1[first]), int(st.class2[second])
	if c1 >= st.class1Count || c2 >= st.class2Count {
		return 0, false
	}
	return st.values[c1*st.class2Count+c2], true
}

// adjustment returns the kerning between two glyphs in the design units of
// the font: the adjustments of all the lookups added up, each lookup
// contributing the first of its subtables that applies to the pair.
func (pk *pairKerning) adjustment(first, second int) (total int) {
	if first < 0 || first > 0xFFFF || second < 0 || second > 0xFFFF {
		return 0
	}
	for _, subtables := range pk.lookups {
		for j := range subtables {
			if val, ok := subtables[j].adjustment(uint16(first), uint16(second)); ok {
				total += int(val)
				break
			}
		}
	}
	return
}

// fontTable reads big-endian values from a table of a font. A font file can
// be cut short or damaged: reading outside of the table returns zero and
// marks the table as bad instead of failing.
type fontTable struct {
	data   []byte
	bad    bool
	budget int // records that can still be read
}

func (t *fontTable) u16(pos int) int {
	if pos < 0 || pos+2 > len(t.data) {
		t.bad = true
		return 0
	}
	return int(t.data[pos])<<8 | int(t.data[pos+1])
}

func (t *fontTable) i16(pos int) int16 {
	return int16(t.u16(pos))
}

func (t *fontTable) u32(pos int) int {
	return t.u16(pos)<<16 | t.u16(pos+2)
}

// spend counts one more record read.
func (t *fontTable) spend() {
	t.budget--
	if t.budget < 0 {
		t.bad = true
	}
}

// valueSize returns the size in bytes of a positioning value that holds the
// fields flagged in format.
func valueSize(format int) (size int) {
	for bit := 1; bit <= 0x80; bit <<= 1 {
		if format&bit != 0 {
			size += 2
		}
	}
	return
}

// coverage returns the glyphs of the coverage table at pos, in the order of
// their coverage index.
func (t *fontTable) coverage(pos int) (glyphs []uint16) {
	count := t.u16(pos + 2)
	switch t.u16(pos) {
	case 1: // list of glyphs
		for j := 0; j < count && !t.bad; j++ {
			glyphs = append(glyphs, uint16(t.u16(pos+4+2*j)))
			t.spend()
		}
	case 2: // ranges of glyphs
		for j := 0; j < count && !t.bad; j++ {
			start, end := t.u16(pos+4+6*j), t.u16(pos+6+6*j)
			for glyph := start; glyph <= end && !t.bad; glyph++ {
				glyphs = append(glyphs, uint16(glyph))
				t.spend()
			}
		}
	}
	return
}

// classes returns the class of each glyph listed in the class definition
// table at pos. Glyphs of class 0, the class of the glyphs that are not
// listed, are left out.
func (t *fontTable) classes(pos int) map[uint16]uint16 {
	classes := make(map[uint16]uint16)
	switch t.u16(pos) {
	case 1: // classes of consecutive glyphs
		start, count := t.u16(pos+2), t.u16(pos+4)
		for j := 0; j < count && start+j <= 0xFFFF && !t.bad; j++ {
			if class := t.u16(pos + 6 + 2*j); class != 0 {
				classes[uint16(start+j)] = uint16(class)
			}
			t.spend()
		}
	case 2: // ranges of glyphs of the same class
		count := t.u16(pos + 2)
		for j := 0; j < count && !t.bad; j++ {
			start, end, class := t.u16(pos+4+6*j), t.u16(pos+6+6*j), t.u16(pos+8+6*j)
			for glyph := start; glyph <= end && !t.bad; glyph++ {
				if class != 0 {
					classes[uint16(glyph)] = uint16(class)
				}
				t.spend()
			}
		}
	}
	return classes
}

// pairSubtable reads the pair adjustment subtable at pos. ok is false if the
// subtable has nothing that can be used.
func (t *fontTable) pairSubtable(pos int) (st kernSubtable, ok bool) {
	format := t.u16(pos)
	coverage := t.coverage(pos + t.u16(pos+2))
	// Each pair has a value for each of its glyphs. Kerning is the
	// horizontal advance of the first one, which comes after the horizontal
	// and vertical placements when they are present.
	format1, format2 := t.u16(pos+4), t.u16(pos+6)
	recordSize := valueSize(format1) + valueSize(format2)
	advance := -1
	if format1&0x4 != 0 {
		advance = valueSize(format1 & 0x3)
	}
	switch format {
	case 1: // pairs listed one by one, a set for each covered glyph
		st.pairs = make(map[uint32]int16)
		setCount := t.u16(pos + 8)
		for j := 0; j < setCount && j < len(coverage) && !t.bad; j++ {
			set := pos + t.u16(pos+10+2*j)
			pairCount := t.u16(set)
			for k := 0; k < pairCount && !t.bad; k++ {
				record := set + 2 + k*(2+recordSize)
				key := uint32(coverage[j])<<16 | uint32(t.u16(record))
				if _, repeated := st.pairs[key]; !repeated {
					var val int16
					if advance >= 0 {
						val = t.i16(record + 2 + advance)
					}
					st.pairs[key] = val
				}
				t.spend()
			}
		}
	case 2: // pairs by class
		st.covered = make(map[uint16]bool, len(coverage))
		for _, glyph := range coverage {
			st.covered[glyph] = true
		}
		st.class1 = t.classes(pos + t.u16(pos+8))
		st.class2 = t.classes(pos + t.u16(pos+10))
		st.class1Count, st.class2Count = t.u16(pos+12), t.u16(pos+14)
		count := st.class1Count * st.class2Count
		if recordSize == 0 || pos+16+count*recordSize > len(t.data) {
			t.bad = t.bad || recordSize != 0
			return st, false
		}
		st.values = make([]int16, count)
		if advance >= 0 {
			for j := range st.values {
				st.values[j] = t.i16(pos + 16 + j*recordSize + advance)
			}
		}
	default:
		return st, false
	}
	return st, !t.bad
}

// parsePairKerning reads the pair adjustments of the "kern" feature from the
// GPOS table of a font. It returns nil if the table has none or cannot be
// read.
//
// A font can register a different kern feature for each script and language.
// The lookups of all of them are used together, which gives the same result
// as choosing the script of each run of text in all but exotic fonts, since
// each lookup lists the glyphs it applies to. Only the adjustments of pairs
// of adjacent glyphs are read: contextual kerning is ignored.
func parsePairKerning(gpos []byte) *pairKerning {
	t := &fontTable{data: gpos, budget: maxKernRecords}
	featureList, lookupList := t.u16(6), t.u16(8)
	// Lookups used by the features tagged "kern"
	wanted := make(map[int]bool)
	featureCount := t.u16(featureList)
	for j := 0; j < featureCount && !t.bad; j++ {
		record := featureList + 2 + 6*j
		if t.u32(record) != 0x6B65726E { // "kern"
			continue
		}
		feature := featureList + t.u16(record+4)
		indexCount := t.u16(feature + 2)
		for k := 0; k < indexCount && !t.bad; k++ {
			wanted[t.u16(feature+4+2*k)] = true
		}
	}
	// Lookups are applied in the order of the lookup list, whatever the
	// order in which the feature names them.
	pk := &pairKerning{}
	lookupCount := t.u16(lookupList)
	for index := 0; index < lookupCount && !t.bad; index++ {
		if !wanted[index] {
			continue
		}
		lookup := lookupList + t.u16(lookupList+2+2*index)
		lookupType := t.u16(lookup)
		subtableCount := t.u16(lookup + 4)
		var subtables []kernSubtable
		for k := 0; k < subtableCount && !t.bad; k++ {
			pos := lookup + t.u16(lookup+6+2*k)
			kind := lookupType
			if kind == 9 { // extension: the subtable itself is further away
				kind = t.u16(pos + 2)
				pos += t.u32(pos + 4)
			}
			if kind != 2 { // not a pair adjustment
				continue
			}
			if st, ok := t.pairSubtable(pos); ok {
				subtables = append(subtables, st)
			}
		}
		if len(subtables) > 0 {
			pk.lookups = append(pk.lookups, subtables)
		}
	}
	if t.bad || len(pk.lookups) == 0 {
		return nil
	}
	return pk
}

// kerning returns the kerning between two characters in thousandths of the
// font size, or zero if the font has none for that pair. The GPOS table is
// read the first time that it is needed.
func (utf *utf8FontFile) kerning(left, right int) int {
	if !utf.kernRead {
		utf.kernRead = true
		utf.pairKerning = parsePairKerning(utf.gpos)
	}
	if utf.pairKerning == nil || utf.fontElementSize == 0 {
		return 0
	}
	first, ok := utf.runeGlyphs[left]
	if !ok {
		return 0
	}
	second, ok := utf.runeGlyphs[right]
	if !ok {
		return 0
	}
	units := utf.pairKerning.adjustment(first, second)
	return int(math.Round(float64(units) * 1000 / float64(utf.fontElementSize)))
}
