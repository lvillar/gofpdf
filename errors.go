package gofpdf

import (
	"errors"
	"fmt"
)

// Sentinel errors for common PDF generation failure conditions.
var (
	ErrNoFont       = errors.New("gopdf: font has not been set")
	ErrNoPage       = errors.New("gopdf: no page has been added")
	ErrClosed       = errors.New("gopdf: document is closed")
	ErrInvalidParam = errors.New("gopdf: invalid parameter")
	ErrUnsupported  = errors.New("gopdf: unsupported operation")
	ErrEncrypted    = errors.New("gopdf: document is encrypted")
	ErrCorrupted    = errors.New("gopdf: document is corrupted")
)

// PDFError represents an error that occurred during a specific PDF operation.
// It wraps an underlying error and includes the operation name for context.
type PDFError struct {
	Op  string // operation name, e.g. "AddPage", "SetFont"
	Err error  // underlying error
}

func (e *PDFError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("gopdf.%s: %v", e.Op, e.Err)
	}
	return fmt.Sprintf("gopdf.%s: unknown error", e.Op)
}

func (e *PDFError) Unwrap() error {
	return e.Err
}

// newPDFError creates a new PDFError wrapping the given error with operation context.
func newPDFError(op string, err error) *PDFError {
	return &PDFError{Op: op, Err: err}
}
