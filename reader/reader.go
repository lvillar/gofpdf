package reader

import (
	"fmt"
	"io"
	"iter"
	"os"
	"strings"
)

// Document represents a parsed PDF document.
type Document struct {
	Version string // PDF version from file header (e.g., "1.7")
	xref    xrefTable
	trailer Dict
	data    []byte
	pages   []*Page
	encrypt *encryptInfo // non-nil if document is encrypted and decrypted
}

// Open opens and parses a PDF file from disk.
func Open(filename string) (*Document, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reader: opening %s: %w", filename, err)
	}
	return parse(data)
}

// ReadFrom parses a PDF document from a reader.
// The reader content is read entirely into memory for random access.
func ReadFrom(r io.Reader) (*Document, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reader: reading input: %w", err)
	}
	return parse(data)
}

// OpenWithPassword opens and parses an encrypted PDF file using the given password.
func OpenWithPassword(filename, password string) (*Document, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reader: opening %s: %w", filename, err)
	}
	return parseWithPassword(data, password)
}

// ReadFromWithPassword parses an encrypted PDF from a reader using the given password.
func ReadFromWithPassword(r io.Reader, password string) (*Document, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reader: reading input: %w", err)
	}
	return parseWithPassword(data, password)
}

// parse is the internal entry point that builds a Document from raw PDF bytes.
func parse(data []byte) (*Document, error) {
	return parseWithPassword(data, "")
}

// parseWithPassword parses a PDF, attempting to decrypt if encrypted.
func parseWithPassword(data []byte, password string) (*Document, error) {
	doc := &Document{data: data}

	// Parse PDF version from header
	doc.Version = parseVersion(data)

	// Find and parse cross-reference table
	startXRef, err := findStartXRef(data)
	if err != nil {
		return nil, err
	}

	xref, trailer, err := parseXRefTable(data, startXRef)
	if err != nil {
		return nil, err
	}
	doc.xref = xref
	doc.trailer = trailer

	// Handle encryption
	if doc.isEncrypted() {
		if err := doc.decrypt(password); err != nil {
			return nil, fmt.Errorf("reader: %w", err)
		}
	}

	// Build page list from page tree
	if err := doc.buildPageList(); err != nil {
		return nil, err
	}

	return doc, nil
}

// parseVersion extracts the PDF version from the file header (e.g., "%PDF-1.7").
func parseVersion(data []byte) string {
	if len(data) < 8 {
		return ""
	}
	header := string(data[:min(20, len(data))])
	if idx := strings.Index(header, "%PDF-"); idx >= 0 {
		end := idx + 5
		for end < len(header) && header[end] != '\n' && header[end] != '\r' {
			end++
		}
		return header[idx+5 : end]
	}
	return ""
}

// NumPages returns the total number of pages in the document.
func (d *Document) NumPages() int {
	return len(d.pages)
}

// Page returns the page at the given 1-based index.
func (d *Document) Page(n int) (*Page, error) {
	if n < 1 || n > len(d.pages) {
		return nil, fmt.Errorf("reader: page %d out of range [1, %d]", n, len(d.pages))
	}
	return d.pages[n-1], nil
}

// Pages returns an iterator over all pages. Index is 1-based.
func (d *Document) Pages() iter.Seq2[int, *Page] {
	return func(yield func(int, *Page) bool) {
		for i, page := range d.pages {
			if !yield(i+1, page) {
				return
			}
		}
	}
}

// Metadata returns document metadata from the /Info dictionary.
func (d *Document) Metadata() map[string]string {
	meta := make(map[string]string)

	infoObj, ok := d.trailer["Info"]
	if !ok {
		return meta
	}

	var infoDict Dict
	switch v := infoObj.(type) {
	case Dict:
		infoDict = v
	case Reference:
		resolved, err := d.resolve(v)
		if err != nil {
			return meta
		}
		infoDict, _ = resolved.(Dict)
	}

	if infoDict == nil {
		return meta
	}

	for _, key := range []Name{"Title", "Author", "Subject", "Keywords", "Creator", "Producer"} {
		if v, ok := infoDict[key]; ok {
			if s, ok := v.(String); ok {
				meta[string(key)] = decodePDFString(s.Value)
			}
		}
	}
	return meta
}

// resolve resolves an indirect reference to the actual object.
func (d *Document) resolve(ref Reference) (Object, error) {
	entry, ok := d.xref[ref.Number]
	if !ok || !entry.InUse {
		return Null{}, nil
	}

	if entry.Offset < 0 || int(entry.Offset) >= len(d.data) {
		return nil, fmt.Errorf("reader: object %d offset %d out of bounds", ref.Number, entry.Offset)
	}

	p := newParser(d.data[entry.Offset:])

	// Set up per-object RC4 cipher for decryption.
	// gofpdf reuses cipher state across strings in the same object,
	// so we must decrypt strings in byte order during parsing.
	if d.encrypt != nil && d.encrypt.key != nil {
		p.cipher = d.makeObjectCipher(ref.Number, ref.Generation)
	}

	obj, err := p.ParseIndirectObject()
	if err != nil {
		return nil, fmt.Errorf("reader: parsing object %d: %w", ref.Number, err)
	}

	return obj.Value, nil
}

// resolveIfRef resolves an object if it is a Reference, otherwise returns it as-is.
func (d *Document) resolveIfRef(obj Object) (Object, error) {
	if ref, ok := obj.(Reference); ok {
		return d.resolve(ref)
	}
	return obj, nil
}

// ResolveReference resolves an indirect reference to the actual object.
// This is the public API for resolving references.
func (d *Document) ResolveReference(ref Reference) (Object, error) {
	return d.resolve(ref)
}
