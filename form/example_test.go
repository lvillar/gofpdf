package form_test

import (
	gofpdf "github.com/lvillar/gofpdf"
	"github.com/lvillar/gofpdf/form"
	"github.com/lvillar/gofpdf/internal/example"
)

// ExampleFormBuilder demonstrates creating an interactive PDF form with
// text fields, checkboxes, and dropdown menus.
func ExampleFormBuilder() {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.AddPage()

	// Title
	pdf.SetFont("Helvetica", "B", 16)
	pdf.Cell(0, 10, "Registration Form")
	pdf.Ln(14)

	// Labels
	pdf.SetFont("Helvetica", "", 11)
	labelX := 20.0
	fieldX := 60.0
	y := 35.0
	lineH := 12.0

	pdf.Text(labelX, y+5, "Full Name:")
	pdf.Text(labelX, y+lineH+5, "Email:")
	pdf.Text(labelX, y+2*lineH+5, "Country:")
	pdf.Text(labelX, y+3*lineH+5, "Comments:")
	pdf.Text(labelX, y+6*lineH+5, "Subscribe:")

	// Create form builder
	fb := form.NewFormBuilder(pdf)

	// Text fields
	fb.AddTextField("fullname", 1, fieldX, y, 80, 8).SetRequired(true)
	fb.AddTextField("email", 1, fieldX, y+lineH, 80, 8).SetRequired(true)

	// Dropdown
	fb.AddDropdown("country", 1, fieldX, y+2*lineH, 80, 8,
		[]string{"United States", "Canada", "United Kingdom", "Germany", "France", "Japan", "Other"})

	// Multi-line text area
	fb.AddTextField("comments", 1, fieldX, y+3*lineH, 80, 28).SetMultiLine(true).SetMaxLen(500)

	// Checkbox
	fb.AddCheckbox("subscribe", 1, fieldX, y+6*lineH, 5)

	if err := fb.Build(); err != nil {
		panic(err)
	}

	fileStr := example.Filename("FormBuilder")
	err := pdf.OutputFileAndClose(fileStr)
	example.SummaryCompare(err, fileStr)
	// Output:
	// Successfully generated ../pdf/FormBuilder.pdf
}
