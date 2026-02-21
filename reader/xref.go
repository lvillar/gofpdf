package reader

import (
	"bytes"
	"fmt"
	"strconv"
)

// xrefEntry represents a single cross-reference table entry.
type xrefEntry struct {
	Offset     int64
	Generation int
	InUse      bool
}

// xrefTable maps object numbers to their file offsets.
type xrefTable map[int]xrefEntry

// findStartXRef locates the "startxref" position from the end of the file.
func findStartXRef(data []byte) (int64, error) {
	// Search backward from end of file for "startxref"
	searchLen := 1024
	if len(data) < searchLen {
		searchLen = len(data)
	}
	tail := data[len(data)-searchLen:]

	idx := bytes.LastIndex(tail, []byte("startxref"))
	if idx < 0 {
		return 0, fmt.Errorf("reader: startxref not found")
	}

	// Parse the offset value after "startxref"
	p := newParser(tail[idx+9:]) // 9 = len("startxref")
	p.skipWhitespace()
	tok := p.readToken()
	offset, err := strconv.ParseInt(tok, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("reader: invalid startxref offset %q: %w", tok, err)
	}
	return offset, nil
}

// parseXRefTable parses a traditional cross-reference table starting at the given offset.
// Returns the xref entries and the trailer dictionary.
func parseXRefTable(data []byte, offset int64) (xrefTable, Dict, error) {
	if offset < 0 || int(offset) >= len(data) {
		return nil, nil, fmt.Errorf("reader: xref offset %d out of bounds", offset)
	}

	p := newParser(data[offset:])
	table := make(xrefTable)

	// Expect "xref" keyword
	tok := p.readToken()
	if tok != "xref" {
		// Could be a cross-reference stream (PDF 1.5+)
		return parseXRefStream(data, offset)
	}

	// Parse subsections: startObj count
	for {
		p.skipWhitespace()
		if p.pos >= len(p.data) {
			break
		}

		// Check if we've reached the trailer
		savedPos := p.pos
		tok = p.readToken()
		if tok == "trailer" {
			break
		}
		p.pos = savedPos

		// Read start object number
		startTok := p.readToken()
		startObj, err := strconv.ParseInt(startTok, 10, 64)
		if err != nil {
			return nil, nil, fmt.Errorf("reader: xref start obj %q: %w", startTok, err)
		}

		// Read count
		p.skipWhitespace()
		countTok := p.readToken()
		count, err := strconv.ParseInt(countTok, 10, 64)
		if err != nil {
			return nil, nil, fmt.Errorf("reader: xref count %q: %w", countTok, err)
		}

		// Read entries
		for i := int64(0); i < count; i++ {
			p.skipWhitespace()
			offsetTok := p.readToken()
			entryOffset, err := strconv.ParseInt(offsetTok, 10, 64)
			if err != nil {
				return nil, nil, fmt.Errorf("reader: xref entry offset: %w", err)
			}

			p.skipWhitespace()
			genTok := p.readToken()
			gen, err := strconv.ParseInt(genTok, 10, 64)
			if err != nil {
				return nil, nil, fmt.Errorf("reader: xref entry generation: %w", err)
			}

			p.skipWhitespace()
			typeTok := p.readToken()

			objNum := int(startObj + i)
			// Only add if not already present (first definition wins for incremental updates)
			if _, exists := table[objNum]; !exists {
				table[objNum] = xrefEntry{
					Offset:     entryOffset,
					Generation: int(gen),
					InUse:      typeTok == "n",
				}
			}
		}
	}

	// Parse trailer dictionary
	p.skipWhitespace()
	obj, err := p.ParseObject()
	if err != nil {
		return nil, nil, fmt.Errorf("reader: trailer dict: %w", err)
	}
	trailer, ok := obj.(Dict)
	if !ok {
		return nil, nil, fmt.Errorf("reader: trailer is not a dictionary")
	}

	// Follow /Prev link for incremental updates
	if prevVal, ok := trailer.GetInt("Prev"); ok {
		prevTable, _, err := parseXRefTable(data, prevVal)
		if err != nil {
			return nil, nil, fmt.Errorf("reader: previous xref: %w", err)
		}
		// Merge: current entries take precedence
		for num, entry := range prevTable {
			if _, exists := table[num]; !exists {
				table[num] = entry
			}
		}
	}

	return table, trailer, nil
}

// parseXRefStream parses a cross-reference stream (PDF 1.5+).
func parseXRefStream(data []byte, offset int64) (xrefTable, Dict, error) {
	p := newParser(data[offset:])
	obj, err := p.ParseIndirectObject()
	if err != nil {
		return nil, nil, fmt.Errorf("reader: xref stream object: %w", err)
	}

	stream, ok := obj.Value.(Stream)
	if !ok {
		return nil, nil, fmt.Errorf("reader: xref stream is not a stream object")
	}

	// Decode the stream
	decoded, err := decodeStream(stream)
	if err != nil {
		return nil, nil, fmt.Errorf("reader: decoding xref stream: %w", err)
	}

	// Get /W array (field widths)
	wArr := stream.Dict.GetArray("W")
	if len(wArr) != 3 {
		return nil, nil, fmt.Errorf("reader: xref stream /W must have 3 elements")
	}

	widths := make([]int, 3)
	for i, w := range wArr {
		if intVal, ok := w.(Integer); ok {
			widths[i] = int(intVal)
		}
	}
	entrySize := widths[0] + widths[1] + widths[2]

	// Get /Index array (default: [0 Size])
	var indices []int
	if idxArr := stream.Dict.GetArray("Index"); idxArr != nil {
		for _, v := range idxArr {
			if intVal, ok := v.(Integer); ok {
				indices = append(indices, int(intVal))
			}
		}
	} else {
		size, _ := stream.Dict.GetInt("Size")
		indices = []int{0, int(size)}
	}

	table := make(xrefTable)
	dataPos := 0

	for i := 0; i+1 < len(indices); i += 2 {
		startObj := indices[i]
		count := indices[i+1]

		for j := 0; j < count; j++ {
			if dataPos+entrySize > len(decoded) {
				break
			}

			fields := make([]int64, 3)
			pos := dataPos
			for f := 0; f < 3; f++ {
				var val int64
				for k := 0; k < widths[f]; k++ {
					val = val<<8 | int64(decoded[pos])
					pos++
				}
				fields[f] = val
			}
			dataPos += entrySize

			objNum := startObj + j

			// Default type is 1 if width[0] is 0
			fieldType := fields[0]
			if widths[0] == 0 {
				fieldType = 1
			}

			switch fieldType {
			case 0: // free object
				table[objNum] = xrefEntry{InUse: false, Generation: int(fields[2])}
			case 1: // in-use object
				table[objNum] = xrefEntry{
					Offset:     fields[1],
					Generation: int(fields[2]),
					InUse:      true,
				}
			case 2: // compressed object (in object stream)
				// fields[1] = object stream number, fields[2] = index within stream
				table[objNum] = xrefEntry{
					Offset:     fields[1], // store stream object number in Offset
					Generation: int(fields[2]),
					InUse:      true,
				}
			}
		}
	}

	return table, stream.Dict, nil
}
