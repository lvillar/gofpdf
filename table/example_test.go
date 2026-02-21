package table_test

import (
	gofpdf "github.com/lvillar/gofpdf"
	"github.com/lvillar/gofpdf/internal/example"
	"github.com/lvillar/gofpdf/table"
)

// ExampleTable demonstrates creating a styled data table with headers,
// alternating row colors, and custom column widths.
func ExampleTable() {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.AddPage()

	pdf.SetFont("Helvetica", "B", 14)
	pdf.Cell(0, 10, "Employee Directory")
	pdf.Ln(12)
	pdf.SetFont("Helvetica", "", 10)

	tbl := table.New(pdf)
	tbl.SetColumnWidths(10, 40, 50, 30, 30)
	tbl.SetStyle(table.TableStyle{
		CellPadding: table.UniformPadding(2),
		Border:      &table.BorderStyle{Width: 0.3, Color: table.RGBColor{R: 180, G: 180, B: 180}},
		HeaderStyle: &table.CellStyle{
			FillColor: &table.RGBColor{R: 41, G: 128, B: 185},
			TextColor: &table.RGBColor{R: 255, G: 255, B: 255},
			Font:      &table.FontSpec{Family: "Helvetica", Style: "B", Size: 10},
			Align:     "C",
		},
		AlternateRows: &table.AlternateStyle{
			Even: table.CellStyle{FillColor: &table.RGBColor{R: 245, G: 245, B: 245}},
			Odd:  table.CellStyle{FillColor: &table.RGBColor{R: 255, G: 255, B: 255}},
		},
	})

	header := tbl.AddHeaderRow()
	header.AddCell("#")
	header.AddCell("Name")
	header.AddCell("Email")
	header.AddCell("Department")
	header.AddCell("Status")

	data := [][]string{
		{"1", "Alice Johnson", "alice@example.com", "Engineering", "Active"},
		{"2", "Bob Smith", "bob@example.com", "Design", "Active"},
		{"3", "Carol White", "carol@example.com", "Marketing", "On Leave"},
		{"4", "Dave Brown", "dave@example.com", "Engineering", "Active"},
		{"5", "Eve Davis", "eve@example.com", "Sales", "Active"},
		{"6", "Frank Miller", "frank@example.com", "Design", "Inactive"},
		{"7", "Grace Lee", "grace@example.com", "Engineering", "Active"},
		{"8", "Henry Wilson", "henry@example.com", "Marketing", "Active"},
	}

	for _, d := range data {
		row := tbl.AddRow()
		row.AddCell(d[0]).SetAlign("C")
		row.AddCell(d[1])
		row.AddCell(d[2])
		row.AddCell(d[3]).SetAlign("C")
		cell := row.AddCell(d[4]).SetAlign("C")
		if d[4] == "Active" {
			cell.SetFillColor(212, 237, 218)
		} else if d[4] == "Inactive" {
			cell.SetFillColor(248, 215, 218)
		}
	}

	tbl.Render()

	fileStr := example.Filename("Table")
	err := pdf.OutputFileAndClose(fileStr)
	example.SummaryCompare(err, fileStr)
	// Output:
	// Successfully generated ../pdf/Table.pdf
}
