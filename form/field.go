// Package form provides functionality for creating interactive PDF form fields
// (AcroForm) in documents generated with gofpdf.
//
// It supports text fields, checkboxes, radio buttons, dropdowns, and buttons.
// Forms can be created in new PDFs and fields are added as widget annotations.
package form

import (
	"fmt"
	"strings"

	gofpdf "github.com/lvillar/gofpdf"
)

// FieldType specifies the type of form field.
type FieldType int

const (
	TypeText     FieldType = iota // single or multi-line text input
	TypeCheckbox                  // checkbox (on/off)
	TypeRadio                     // radio button group
	TypeDropdown                  // dropdown/combo box
	TypeButton                    // push button
)

// Field defines a form field to be added to a PDF page.
type Field struct {
	Name     string    // field name (must be unique within the form)
	Type     FieldType // field type
	Page     int       // page number (1-based)
	X, Y     float64   // position in user units
	W, H     float64   // width and height in user units
	Value    string    // default value
	Options  []string  // options for dropdown/radio fields
	FontSize float64   // font size for text display (default: 12)
	MaxLen   int       // maximum text length (0 = unlimited)
	ReadOnly bool      // whether the field is read-only
	Required bool      // whether the field is required
	MultiLine bool     // for text fields: allow multi-line input
}

// FormBuilder manages the creation of interactive form fields on a PDF.
type FormBuilder struct {
	pdf    *gofpdf.Fpdf
	fields []Field
}

// NewFormBuilder creates a new FormBuilder associated with the given PDF.
func NewFormBuilder(pdf *gofpdf.Fpdf) *FormBuilder {
	return &FormBuilder{pdf: pdf}
}

// AddTextField adds a text input field to the form.
func (fb *FormBuilder) AddTextField(name string, page int, x, y, w, h float64) *Field {
	f := Field{
		Name:     name,
		Type:     TypeText,
		Page:     page,
		X:        x,
		Y:        y,
		W:        w,
		H:        h,
		FontSize: 12,
	}
	fb.fields = append(fb.fields, f)
	return &fb.fields[len(fb.fields)-1]
}

// AddCheckbox adds a checkbox field to the form.
func (fb *FormBuilder) AddCheckbox(name string, page int, x, y, size float64) *Field {
	f := Field{
		Name: name,
		Type: TypeCheckbox,
		Page: page,
		X:    x,
		Y:    y,
		W:    size,
		H:    size,
	}
	fb.fields = append(fb.fields, f)
	return &fb.fields[len(fb.fields)-1]
}

// AddDropdown adds a dropdown/combo box field to the form.
func (fb *FormBuilder) AddDropdown(name string, page int, x, y, w, h float64, options []string) *Field {
	f := Field{
		Name:     name,
		Type:     TypeDropdown,
		Page:     page,
		X:        x,
		Y:        y,
		W:        w,
		H:        h,
		Options:  options,
		FontSize: 12,
	}
	fb.fields = append(fb.fields, f)
	return &fb.fields[len(fb.fields)-1]
}

// AddButton adds a push button field to the form.
func (fb *FormBuilder) AddButton(name string, page int, x, y, w, h float64, label string) *Field {
	f := Field{
		Name:  name,
		Type:  TypeButton,
		Page:  page,
		X:     x,
		Y:     y,
		W:     w,
		H:     h,
		Value: label,
	}
	fb.fields = append(fb.fields, f)
	return &fb.fields[len(fb.fields)-1]
}

// SetValue sets the default value for a field. Returns the field for chaining.
func (f *Field) SetValue(v string) *Field {
	f.Value = v
	return f
}

// SetRequired marks the field as required.
func (f *Field) SetRequired(required bool) *Field {
	f.Required = required
	return f
}

// SetReadOnly marks the field as read-only.
func (f *Field) SetReadOnly(readOnly bool) *Field {
	f.ReadOnly = readOnly
	return f
}

// SetMaxLen sets the maximum input length for text fields.
func (f *Field) SetMaxLen(n int) *Field {
	f.MaxLen = n
	return f
}

// SetMultiLine enables multi-line input for text fields.
func (f *Field) SetMultiLine(multiLine bool) *Field {
	f.MultiLine = multiLine
	return f
}

// Build generates the AcroForm structure and injects it into the PDF.
// This must be called after all pages have been added but before Output().
func (fb *FormBuilder) Build() error {
	if len(fb.fields) == 0 {
		return nil
	}

	k := fb.pdf.GetScaleFactor()

	// Collect field reference strings for the AcroForm /Fields array
	var fieldRefs []string

	for i, f := range fb.fields {
		annot, fieldRef := buildFieldAnnotation(f, i, k)
		fb.pdf.AddPageAnnotation(f.Page, annot)
		fieldRefs = append(fieldRefs, fieldRef)
	}

	// Build AcroForm catalog entry
	acroForm := fmt.Sprintf("/AcroForm <</Fields [%s] /DR <</Font <</Helv <</Type /Font /Subtype /Type1 /BaseFont /Helvetica>>>>>> /DA (/Helv 0 Tf 0 g) /NeedAppearances true>>",
		strings.Join(fieldRefs, " "))
	fb.pdf.AddCatalogEntry(acroForm)

	return fb.pdf.Error()
}

// buildFieldAnnotation constructs the PDF annotation string for a field.
func buildFieldAnnotation(f Field, index int, k float64) (annot string, fieldRef string) {
	// Convert user units to points
	x := f.X * k
	y := f.Y * k
	w := f.W * k
	h := f.H * k

	// Field flags
	var ff int
	if f.ReadOnly {
		ff |= 1 // Bit 1: ReadOnly
	}
	if f.Required {
		ff |= 2 // Bit 2: Required
	}

	// The field reference will be inline (annotation IS the field)
	// We use the annotation directly in /Fields
	fieldRef = fmt.Sprintf("<</Type /Annot /Subtype /Widget /T (%s) /Rect [%.2f %.2f %.2f %.2f]",
		escapePDFString(f.Name), x, y, x+w, y+h)

	switch f.Type {
	case TypeText:
		fieldRef += " /FT /Tx"
		if f.FontSize > 0 {
			fieldRef += fmt.Sprintf(" /DA (/Helv %.1f Tf 0 g)", f.FontSize)
		}
		if f.Value != "" {
			fieldRef += fmt.Sprintf(" /V (%s)", escapePDFString(f.Value))
		}
		if f.MaxLen > 0 {
			fieldRef += fmt.Sprintf(" /MaxLen %d", f.MaxLen)
		}
		if f.MultiLine {
			ff |= 1 << 12 // Bit 13: Multiline
		}

	case TypeCheckbox:
		fieldRef += " /FT /Btn"
		if f.Value == "Yes" || f.Value == "true" || f.Value == "on" {
			fieldRef += " /V /Yes /AS /Yes"
		} else {
			fieldRef += " /V /Off /AS /Off"
		}

	case TypeDropdown:
		fieldRef += " /FT /Ch"
		ff |= 1 << 17 // Bit 18: Combo (dropdown)
		if len(f.Options) > 0 {
			opts := make([]string, len(f.Options))
			for i, opt := range f.Options {
				opts[i] = fmt.Sprintf("(%s)", escapePDFString(opt))
			}
			fieldRef += fmt.Sprintf(" /Opt [%s]", strings.Join(opts, " "))
		}
		if f.Value != "" {
			fieldRef += fmt.Sprintf(" /V (%s)", escapePDFString(f.Value))
		}
		if f.FontSize > 0 {
			fieldRef += fmt.Sprintf(" /DA (/Helv %.1f Tf 0 g)", f.FontSize)
		}

	case TypeButton:
		fieldRef += " /FT /Btn"
		ff |= 1 << 16 // Bit 17: Pushbutton
		if f.Value != "" {
			fieldRef += fmt.Sprintf(" /MK <</CA (%s)>>", escapePDFString(f.Value))
		}
	}

	if ff != 0 {
		fieldRef += fmt.Sprintf(" /Ff %d", ff)
	}

	fieldRef += ">>"

	// The annotation is the same as the field (inline widget)
	annot = fieldRef
	return annot, fieldRef
}

// escapePDFString escapes special characters in a PDF string.
func escapePDFString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `(`, `\(`)
	s = strings.ReplaceAll(s, `)`, `\)`)
	return s
}
