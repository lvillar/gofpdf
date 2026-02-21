package reader

import (
	"bytes"
	"crypto/rc4"
	"fmt"
	"io"
	"strconv"
)

// parser is a recursive descent parser for PDF syntax.
type parser struct {
	data   []byte
	pos    int
	cipher *rc4.Cipher // optional: decrypts strings/streams in byte order
}

// newParser creates a parser from a byte slice.
func newParser(data []byte) *parser {
	return &parser{data: data}
}

// newParserFromReader creates a parser by reading all data from a reader.
func newParserFromReader(r io.Reader) (*parser, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reader: reading input: %w", err)
	}
	return &parser{data: data}, nil
}

// remaining returns the number of unread bytes.
func (p *parser) remaining() int {
	return len(p.data) - p.pos
}

// peek returns the byte at the current position without advancing.
func (p *parser) peek() (byte, bool) {
	if p.pos >= len(p.data) {
		return 0, false
	}
	return p.data[p.pos], true
}

// read returns the byte at the current position and advances.
func (p *parser) read() (byte, bool) {
	if p.pos >= len(p.data) {
		return 0, false
	}
	b := p.data[p.pos]
	p.pos++
	return b, true
}

// skipWhitespace advances past whitespace and comments.
func (p *parser) skipWhitespace() {
	for p.pos < len(p.data) {
		b := p.data[p.pos]
		switch b {
		case ' ', '\t', '\n', '\r', '\f', 0:
			p.pos++
		case '%':
			// Comment: skip to end of line
			for p.pos < len(p.data) && p.data[p.pos] != '\n' && p.data[p.pos] != '\r' {
				p.pos++
			}
		default:
			return
		}
	}
}

// isWhitespace returns true if the byte is a PDF whitespace character.
func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f' || b == 0
}

// isDelimiter returns true if the byte is a PDF delimiter character.
func isDelimiter(b byte) bool {
	return b == '(' || b == ')' || b == '<' || b == '>' ||
		b == '[' || b == ']' || b == '{' || b == '}' ||
		b == '/' || b == '%'
}

// isRegular returns true if the byte is a regular (non-whitespace, non-delimiter) character.
func isRegular(b byte) bool {
	return !isWhitespace(b) && !isDelimiter(b)
}

// readToken reads the next token (keyword or number) as a string.
func (p *parser) readToken() string {
	p.skipWhitespace()
	start := p.pos
	for p.pos < len(p.data) && isRegular(p.data[p.pos]) {
		p.pos++
	}
	return string(p.data[start:p.pos])
}

// ParseObject parses the next PDF object from the current position.
func (p *parser) ParseObject() (Object, error) {
	p.skipWhitespace()
	if p.pos >= len(p.data) {
		return nil, io.ErrUnexpectedEOF
	}

	b := p.data[p.pos]

	switch {
	case b == '<':
		// Could be hex string or dictionary
		if p.pos+1 < len(p.data) && p.data[p.pos+1] == '<' {
			return p.parseDict()
		}
		return p.parseHexString()

	case b == '(':
		return p.parseLiteralString()

	case b == '/':
		return p.parseName()

	case b == '[':
		return p.parseArray()

	case b == 't' || b == 'f':
		return p.parseBoolean()

	case b == 'n':
		return p.parseNull()

	case b >= '0' && b <= '9', b == '+', b == '-', b == '.':
		return p.parseNumberOrRef()

	default:
		return nil, fmt.Errorf("reader: unexpected character %q at position %d", b, p.pos)
	}
}

// parseName parses a PDF name object (/Name).
func (p *parser) parseName() (Name, error) {
	if p.data[p.pos] != '/' {
		return "", fmt.Errorf("reader: expected '/' at position %d", p.pos)
	}
	p.pos++ // skip '/'

	var buf bytes.Buffer
	for p.pos < len(p.data) {
		b := p.data[p.pos]
		if isWhitespace(b) || isDelimiter(b) {
			break
		}
		if b == '#' && p.pos+2 < len(p.data) {
			// Hex-encoded character
			hi := unhex(p.data[p.pos+1])
			lo := unhex(p.data[p.pos+2])
			if hi >= 0 && lo >= 0 {
				buf.WriteByte(byte(hi<<4 | lo))
				p.pos += 3
				continue
			}
		}
		buf.WriteByte(b)
		p.pos++
	}
	return Name(buf.String()), nil
}

// parseBoolean parses a PDF boolean (true/false).
func (p *parser) parseBoolean() (Boolean, error) {
	tok := p.readToken()
	switch tok {
	case "true":
		return Boolean(true), nil
	case "false":
		return Boolean(false), nil
	default:
		return false, fmt.Errorf("reader: expected boolean, got %q", tok)
	}
}

// parseNull parses a PDF null object.
func (p *parser) parseNull() (Null, error) {
	tok := p.readToken()
	if tok != "null" {
		return Null{}, fmt.Errorf("reader: expected null, got %q", tok)
	}
	return Null{}, nil
}

// parseNumberOrRef parses a number (integer or real) or an indirect reference (N G R).
func (p *parser) parseNumberOrRef() (Object, error) {
	savedPos := p.pos
	tok := p.readToken()

	// Try integer first
	intVal, err := strconv.ParseInt(tok, 10, 64)
	if err == nil {
		// Could be start of an indirect reference "N G R"
		pos2 := p.pos
		p.skipWhitespace()
		if p.pos < len(p.data) && p.data[p.pos] >= '0' && p.data[p.pos] <= '9' {
			tok2 := p.readToken()
			genVal, err2 := strconv.ParseInt(tok2, 10, 64)
			if err2 == nil {
				p.skipWhitespace()
				if p.pos < len(p.data) && p.data[p.pos] == 'R' {
					p.pos++ // consume 'R'
					return Reference{Number: int(intVal), Generation: int(genVal)}, nil
				}
			}
		}
		// Not a reference, restore position after first token
		p.pos = pos2
		return Integer(intVal), nil
	}

	// Try real number
	p.pos = savedPos
	tok = p.readToken()
	realVal, err := strconv.ParseFloat(tok, 64)
	if err != nil {
		return nil, fmt.Errorf("reader: invalid number %q at position %d", tok, savedPos)
	}
	return Real(realVal), nil
}

// parseLiteralString parses a PDF literal string: (text).
func (p *parser) parseLiteralString() (String, error) {
	if p.data[p.pos] != '(' {
		return String{}, fmt.Errorf("reader: expected '(' at position %d", p.pos)
	}
	p.pos++ // skip '('

	var buf bytes.Buffer
	depth := 1

	for p.pos < len(p.data) && depth > 0 {
		b := p.data[p.pos]
		p.pos++

		switch b {
		case '(':
			depth++
			buf.WriteByte(b)
		case ')':
			depth--
			if depth > 0 {
				buf.WriteByte(b)
			}
		case '\\':
			if p.pos >= len(p.data) {
				return String{}, fmt.Errorf("reader: unexpected end of string escape")
			}
			esc := p.data[p.pos]
			p.pos++
			switch esc {
			case 'n':
				buf.WriteByte('\n')
			case 'r':
				buf.WriteByte('\r')
			case 't':
				buf.WriteByte('\t')
			case 'b':
				buf.WriteByte('\b')
			case 'f':
				buf.WriteByte('\f')
			case '(', ')', '\\':
				buf.WriteByte(esc)
			default:
				// Octal escape
				if esc >= '0' && esc <= '7' {
					oct := int(esc - '0')
					for i := 0; i < 2 && p.pos < len(p.data) && p.data[p.pos] >= '0' && p.data[p.pos] <= '7'; i++ {
						oct = oct*8 + int(p.data[p.pos]-'0')
						p.pos++
					}
					buf.WriteByte(byte(oct))
				} else {
					buf.WriteByte(esc)
				}
			}
		default:
			buf.WriteByte(b)
		}
	}

	if depth != 0 {
		return String{}, fmt.Errorf("reader: unterminated literal string")
	}
	data := buf.Bytes()
	if p.cipher != nil {
		p.cipher.XORKeyStream(data, data)
	}
	return String{Value: data}, nil
}

// parseHexString parses a PDF hex string: <hex digits>.
func (p *parser) parseHexString() (String, error) {
	if p.data[p.pos] != '<' {
		return String{}, fmt.Errorf("reader: expected '<' at position %d", p.pos)
	}
	p.pos++ // skip '<'

	var buf bytes.Buffer
	var hi int = -1

	for p.pos < len(p.data) {
		b := p.data[p.pos]
		p.pos++

		if b == '>' {
			if hi >= 0 {
				buf.WriteByte(byte(hi << 4)) // trailing nibble
			}
			data := buf.Bytes()
			if p.cipher != nil {
				p.cipher.XORKeyStream(data, data)
			}
			return String{Value: data, IsHex: true}, nil
		}

		if isWhitespace(b) {
			continue
		}

		v := unhex(b)
		if v < 0 {
			return String{}, fmt.Errorf("reader: invalid hex character %q in hex string", b)
		}

		if hi < 0 {
			hi = v
		} else {
			buf.WriteByte(byte(hi<<4 | v))
			hi = -1
		}
	}

	return String{}, fmt.Errorf("reader: unterminated hex string")
}

// parseArray parses a PDF array: [obj1 obj2 ...].
func (p *parser) parseArray() (Array, error) {
	if p.data[p.pos] != '[' {
		return nil, fmt.Errorf("reader: expected '[' at position %d", p.pos)
	}
	p.pos++ // skip '['

	var arr Array
	for {
		p.skipWhitespace()
		if p.pos >= len(p.data) {
			return nil, fmt.Errorf("reader: unterminated array")
		}
		if p.data[p.pos] == ']' {
			p.pos++ // skip ']'
			return arr, nil
		}
		obj, err := p.ParseObject()
		if err != nil {
			return nil, fmt.Errorf("reader: in array: %w", err)
		}
		arr = append(arr, obj)
	}
}

// parseDict parses a PDF dictionary: << /Key Value ... >>.
func (p *parser) parseDict() (Dict, error) {
	if p.pos+1 >= len(p.data) || p.data[p.pos] != '<' || p.data[p.pos+1] != '<' {
		return nil, fmt.Errorf("reader: expected '<<' at position %d", p.pos)
	}
	p.pos += 2 // skip '<<'

	d := make(Dict)
	for {
		p.skipWhitespace()
		if p.pos >= len(p.data) {
			return nil, fmt.Errorf("reader: unterminated dictionary")
		}
		if p.pos+1 < len(p.data) && p.data[p.pos] == '>' && p.data[p.pos+1] == '>' {
			p.pos += 2 // skip '>>'
			return d, nil
		}
		// Key must be a name
		key, err := p.parseName()
		if err != nil {
			return nil, fmt.Errorf("reader: dict key: %w", err)
		}
		// Value is any object
		val, err := p.ParseObject()
		if err != nil {
			return nil, fmt.Errorf("reader: dict value for %s: %w", key, err)
		}
		d[key] = val
	}
}

// ParseIndirectObject parses "N G obj ... endobj".
func (p *parser) ParseIndirectObject() (*IndirectObject, error) {
	p.skipWhitespace()

	// Read object number
	numTok := p.readToken()
	num, err := strconv.ParseInt(numTok, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("reader: expected object number, got %q", numTok)
	}

	// Read generation number
	p.skipWhitespace()
	genTok := p.readToken()
	gen, err := strconv.ParseInt(genTok, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("reader: expected generation number, got %q", genTok)
	}

	// Read "obj" keyword
	p.skipWhitespace()
	objTok := p.readToken()
	if objTok != "obj" {
		return nil, fmt.Errorf("reader: expected 'obj', got %q", objTok)
	}

	// Parse the object value
	val, err := p.ParseObject()
	if err != nil {
		return nil, fmt.Errorf("reader: object %d %d: %w", num, gen, err)
	}

	// Check for stream
	p.skipWhitespace()
	if p.pos+6 <= len(p.data) && string(p.data[p.pos:p.pos+6]) == "stream" {
		dict, ok := val.(Dict)
		if !ok {
			return nil, fmt.Errorf("reader: stream object %d %d has non-dict header", num, gen)
		}

		p.pos += 6 // skip "stream"
		// Skip single \r\n or \n after "stream"
		if p.pos < len(p.data) && p.data[p.pos] == '\r' {
			p.pos++
		}
		if p.pos < len(p.data) && p.data[p.pos] == '\n' {
			p.pos++
		}

		// Read stream data using /Length
		length := 0
		if lenVal, ok := dict.GetInt("Length"); ok {
			length = int(lenVal)
		}

		if p.pos+length > len(p.data) {
			return nil, fmt.Errorf("reader: stream data exceeds file bounds for object %d %d", num, gen)
		}

		streamData := make([]byte, length)
		copy(streamData, p.data[p.pos:p.pos+length])
		p.pos += length

		if p.cipher != nil {
			p.cipher.XORKeyStream(streamData, streamData)
		}

		// Skip "endstream"
		p.skipWhitespace()
		if p.pos+9 <= len(p.data) && string(p.data[p.pos:p.pos+9]) == "endstream" {
			p.pos += 9
		}

		val = Stream{Dict: dict, Data: streamData}
	}

	// Skip "endobj"
	p.skipWhitespace()
	if p.pos+6 <= len(p.data) && string(p.data[p.pos:p.pos+6]) == "endobj" {
		p.pos += 6
	}

	return &IndirectObject{
		Reference: Reference{Number: int(num), Generation: int(gen)},
		Value:     val,
	}, nil
}

// unhex returns the numeric value of a hex digit, or -1 if not valid.
func unhex(b byte) int {
	switch {
	case b >= '0' && b <= '9':
		return int(b - '0')
	case b >= 'a' && b <= 'f':
		return int(b-'a') + 10
	case b >= 'A' && b <= 'F':
		return int(b-'A') + 10
	default:
		return -1
	}
}
