// Package reader provides functionality for reading and parsing existing PDF files.
//
// It implements a PDF parser that can extract the object structure, page tree,
// and text content from PDF documents conforming to the PDF specification (ISO 32000).
package reader

import (
	"fmt"
)

// Object is the interface satisfied by all PDF object types.
// The unexported method prevents external types from implementing it.
type Object interface {
	pdfObject()
	String() string
}

// Null represents the PDF null object.
type Null struct{}

func (Null) pdfObject()     {}
func (Null) String() string { return "null" }

// Boolean represents a PDF boolean value.
type Boolean bool

func (Boolean) pdfObject() {}
func (b Boolean) String() string {
	if b {
		return "true"
	}
	return "false"
}

// Integer represents a PDF integer value.
type Integer int64

func (Integer) pdfObject()       {}
func (i Integer) String() string { return fmt.Sprintf("%d", int64(i)) }

// Real represents a PDF real (floating-point) value.
type Real float64

func (Real) pdfObject()       {}
func (r Real) String() string { return fmt.Sprintf("%g", float64(r)) }

// Name represents a PDF name object (e.g., /Type, /Pages).
type Name string

func (Name) pdfObject()       {}
func (n Name) String() string { return "/" + string(n) }

// String represents a PDF string (literal or hexadecimal).
type String struct {
	Value []byte
	IsHex bool
}

func (String) pdfObject() {}
func (s String) String() string {
	if s.IsHex {
		return fmt.Sprintf("<%x>", s.Value)
	}
	return fmt.Sprintf("(%s)", s.Value)
}

// Array represents a PDF array of objects.
type Array []Object

func (Array) pdfObject()       {}
func (a Array) String() string { return fmt.Sprintf("[array len=%d]", len(a)) }

// Dict represents a PDF dictionary mapping names to objects.
type Dict map[Name]Object

func (Dict) pdfObject()       {}
func (d Dict) String() string { return fmt.Sprintf("<<dict len=%d>>", len(d)) }

// GetName returns the value of a name entry, or empty string if not found.
func (d Dict) GetName(key Name) Name {
	if v, ok := d[key]; ok {
		if n, ok := v.(Name); ok {
			return n
		}
	}
	return ""
}

// GetInt returns the value of an integer entry, or 0 if not found.
func (d Dict) GetInt(key Name) (int64, bool) {
	if v, ok := d[key]; ok {
		switch n := v.(type) {
		case Integer:
			return int64(n), true
		case Real:
			return int64(n), true
		}
	}
	return 0, false
}

// GetDict returns a sub-dictionary, or nil if not found.
func (d Dict) GetDict(key Name) Dict {
	if v, ok := d[key]; ok {
		if sub, ok := v.(Dict); ok {
			return sub
		}
	}
	return nil
}

// GetArray returns an array entry, or nil if not found.
func (d Dict) GetArray(key Name) Array {
	if v, ok := d[key]; ok {
		if arr, ok := v.(Array); ok {
			return arr
		}
	}
	return nil
}

// Stream represents a PDF stream object (dictionary + encoded data).
type Stream struct {
	Dict Dict
	Data []byte // raw data (may be compressed)
}

func (Stream) pdfObject()       {}
func (s Stream) String() string { return fmt.Sprintf("<<stream len=%d>>", len(s.Data)) }

// Reference represents an indirect object reference (e.g., "10 0 R").
type Reference struct {
	Number     int
	Generation int
}

func (Reference) pdfObject() {}
func (r Reference) String() string {
	return fmt.Sprintf("%d %d R", r.Number, r.Generation)
}

// IndirectObject represents a PDF indirect object definition (e.g., "10 0 obj ... endobj").
type IndirectObject struct {
	Reference
	Value Object
}

func (IndirectObject) pdfObject() {}
func (o IndirectObject) String() string {
	return fmt.Sprintf("%d %d obj %s", o.Number, o.Generation, o.Value)
}
