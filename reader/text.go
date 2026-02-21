package reader

import (
	"bytes"
	"strings"
	"unicode/utf16"
)

// ExtractText extracts the text content from this page.
// It parses the content stream and extracts text from BT/ET blocks
// using the Tj, TJ, ', and " operators.
//
// Note: This is a basic extraction that handles common cases. Complex text
// with custom encodings, CIDFonts, or ToUnicode CMaps may not be fully supported.
func (p *Page) ExtractText() (string, error) {
	data, err := p.ContentStream()
	if err != nil {
		return "", err
	}
	return extractTextFromContentStream(data), nil
}

// extractTextFromContentStream parses text operators from a PDF content stream.
func extractTextFromContentStream(data []byte) string {
	var result strings.Builder
	var inText bool

	i := 0
	for i < len(data) {
		// Skip whitespace
		for i < len(data) && isWhitespace(data[i]) {
			i++
		}
		if i >= len(data) {
			break
		}

		// Check for BT (begin text) / ET (end text)
		if i+2 <= len(data) && data[i] == 'B' && data[i+1] == 'T' &&
			(i+2 >= len(data) || isWhitespace(data[i+2]) || isDelimiter(data[i+2])) {
			inText = true
			i += 2
			continue
		}
		if i+2 <= len(data) && data[i] == 'E' && data[i+1] == 'T' &&
			(i+2 >= len(data) || isWhitespace(data[i+2]) || isDelimiter(data[i+2])) {
			inText = false
			result.WriteByte(' ')
			i += 2
			continue
		}

		if !inText {
			// Skip until next token
			if data[i] == '(' {
				i = skipLiteralString(data, i)
			} else if data[i] == '<' {
				i = skipAngleBrackets(data, i)
			} else if data[i] == '[' {
				i = skipArray(data, i)
			} else {
				i++
			}
			continue
		}

		// Inside BT...ET block: look for text operators
		if data[i] == '(' {
			// Literal string - extract text
			text, end := parseLiteralStringRaw(data, i)
			result.WriteString(decodePDFString(text))
			i = end
			continue
		}

		if data[i] == '<' && (i+1 >= len(data) || data[i+1] != '<') {
			// Hex string - extract text
			text, end := parseHexStringRaw(data, i)
			result.WriteString(decodePDFString(text))
			i = end
			continue
		}

		if data[i] == '[' {
			// TJ array - extract text from strings within
			i++ // skip '['
			for i < len(data) && data[i] != ']' {
				if data[i] == '(' {
					text, end := parseLiteralStringRaw(data, i)
					result.WriteString(decodePDFString(text))
					i = end
				} else if data[i] == '<' {
					text, end := parseHexStringRaw(data, i)
					result.WriteString(decodePDFString(text))
					i = end
				} else {
					i++
				}
			}
			if i < len(data) {
				i++ // skip ']'
			}
			continue
		}

		// Check for text positioning operators that imply space/newline
		if i+2 <= len(data) {
			op := string(data[i:min(i+3, len(data))])
			if (op[:2] == "Td" || op[:2] == "TD" || op[:2] == "T*") &&
				(len(op) < 3 || isWhitespace(op[2]) || isDelimiter(op[2])) {
				result.WriteByte(' ')
				i += 2
				continue
			}
		}

		i++
	}

	return strings.TrimSpace(result.String())
}

// parseLiteralStringRaw extracts raw bytes from a literal string starting at pos.
// Returns the bytes and the position after the closing ')'.
func parseLiteralStringRaw(data []byte, pos int) ([]byte, int) {
	if pos >= len(data) || data[pos] != '(' {
		return nil, pos
	}
	pos++ // skip '('

	var buf bytes.Buffer
	depth := 1

	for pos < len(data) && depth > 0 {
		b := data[pos]
		pos++
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
			if pos < len(data) {
				esc := data[pos]
				pos++
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
					if esc >= '0' && esc <= '7' {
						oct := int(esc - '0')
						for j := 0; j < 2 && pos < len(data) && data[pos] >= '0' && data[pos] <= '7'; j++ {
							oct = oct*8 + int(data[pos]-'0')
							pos++
						}
						buf.WriteByte(byte(oct))
					} else {
						buf.WriteByte(esc)
					}
				}
			}
		default:
			buf.WriteByte(b)
		}
	}
	return buf.Bytes(), pos
}

// parseHexStringRaw extracts raw bytes from a hex string starting at pos.
func parseHexStringRaw(data []byte, pos int) ([]byte, int) {
	if pos >= len(data) || data[pos] != '<' {
		return nil, pos
	}
	pos++ // skip '<'

	var buf bytes.Buffer
	hi := -1

	for pos < len(data) {
		b := data[pos]
		pos++
		if b == '>' {
			if hi >= 0 {
				buf.WriteByte(byte(hi << 4))
			}
			return buf.Bytes(), pos
		}
		if isWhitespace(b) {
			continue
		}
		v := unhex(b)
		if v < 0 {
			continue
		}
		if hi < 0 {
			hi = v
		} else {
			buf.WriteByte(byte(hi<<4 | v))
			hi = -1
		}
	}
	return buf.Bytes(), pos
}

// decodePDFString attempts to decode a PDF string to a Go string.
// Handles UTF-16BE BOM and falls back to Latin-1.
func decodePDFString(data []byte) string {
	// Check for UTF-16BE BOM
	if len(data) >= 2 && data[0] == 0xFE && data[1] == 0xFF {
		return decodeUTF16BE(data[2:])
	}
	// Assume PDFDocEncoding (similar to Latin-1 for printable chars)
	var buf strings.Builder
	for _, b := range data {
		buf.WriteRune(rune(b))
	}
	return buf.String()
}

// decodeUTF16BE decodes UTF-16BE encoded bytes to a Go string.
func decodeUTF16BE(data []byte) string {
	if len(data)%2 != 0 {
		data = append(data, 0) // pad
	}
	u16s := make([]uint16, len(data)/2)
	for i := range u16s {
		u16s[i] = uint16(data[2*i])<<8 | uint16(data[2*i+1])
	}
	return string(utf16.Decode(u16s))
}

// skipLiteralString advances past a literal string at pos.
func skipLiteralString(data []byte, pos int) int {
	if pos >= len(data) || data[pos] != '(' {
		return pos + 1
	}
	pos++
	depth := 1
	for pos < len(data) && depth > 0 {
		switch data[pos] {
		case '(':
			depth++
		case ')':
			depth--
		case '\\':
			pos++ // skip escaped character
		}
		pos++
	}
	return pos
}

// skipAngleBrackets advances past angle brackets at pos.
func skipAngleBrackets(data []byte, pos int) int {
	pos++ // skip '<'
	for pos < len(data) && data[pos] != '>' {
		pos++
	}
	if pos < len(data) {
		pos++ // skip '>'
	}
	return pos
}

// skipArray advances past an array at pos.
func skipArray(data []byte, pos int) int {
	pos++ // skip '['
	depth := 1
	for pos < len(data) && depth > 0 {
		switch data[pos] {
		case '[':
			depth++
		case ']':
			depth--
		case '(':
			pos = skipLiteralString(data, pos)
			continue
		}
		pos++
	}
	return pos
}
