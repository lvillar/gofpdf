package table_test

import (
	"bytes"
	"testing"

	gofpdf "github.com/lvillar/gofpdf"
	"github.com/lvillar/gofpdf/table"
)

func newTestPDF() *gofpdf.Fpdf {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.AddPage()
	return pdf
}

func TestBasicTable(t *testing.T) {
	pdf := newTestPDF()

	tb := table.New(pdf)
	tb.SetColumnWidths(40, 60, 30, 30)

	h := tb.AddHeaderRow()
	h.AddCell("ID")
	h.AddCell("Name")
	h.AddCell("Qty")
	h.AddCell("Price")

	r := tb.AddRow()
	r.AddCell("1")
	r.AddCell("Widget")
	r.AddCell("10")
	r.AddCell("$5.00")

	r2 := tb.AddRow()
	r2.AddCell("2")
	r2.AddCell("Gadget")
	r2.AddCell("5")
	r2.AddCell("$12.50")

	if err := tb.Render(); err != nil {
		t.Fatalf("render: %v", err)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		t.Fatalf("output: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected non-empty PDF output")
	}
	t.Logf("Basic table PDF: %d bytes", buf.Len())
}

func TestAutoWidthColumns(t *testing.T) {
	pdf := newTestPDF()

	tb := table.New(pdf)
	tb.SetColumnWidths(0, 0, 0) // all auto

	r := tb.AddRow()
	r.AddCell("Auto 1")
	r.AddCell("Auto 2")
	r.AddCell("Auto 3")

	if err := tb.Render(); err != nil {
		t.Fatalf("render: %v", err)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		t.Fatalf("output: %v", err)
	}
	t.Logf("Auto-width table PDF: %d bytes", buf.Len())
}

func TestAlternatingRows(t *testing.T) {
	pdf := newTestPDF()

	tb := table.New(pdf)
	tb.SetColumnWidths(60, 60, 60)
	tb.SetStyle(table.TableStyle{
		AlternateRows: &table.AlternateStyle{
			Even: table.CellStyle{
				FillColor: &table.RGBColor{240, 240, 240},
			},
			Odd: table.CellStyle{
				FillColor: &table.RGBColor{255, 255, 255},
			},
		},
	})

	for i := 0; i < 10; i++ {
		r := tb.AddRow()
		r.AddCellf("Row %d Col 1", i)
		r.AddCellf("Row %d Col 2", i)
		r.AddCellf("Row %d Col 3", i)
	}

	if err := tb.Render(); err != nil {
		t.Fatalf("render: %v", err)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		t.Fatalf("output: %v", err)
	}
	t.Logf("Alternating rows table PDF: %d bytes", buf.Len())
}

func TestHeaderRepeatsOnPageBreak(t *testing.T) {
	pdf := newTestPDF()

	tb := table.New(pdf)
	tb.SetColumnWidths(60, 60, 60)
	tb.SetHeaderRows(1)

	h := tb.AddHeaderRow()
	h.AddCell("ID")
	h.AddCell("Name")
	h.AddCell("Value")

	// Add enough rows to cause page break
	for i := 0; i < 50; i++ {
		r := tb.AddRow()
		r.AddCellf("%d", i+1)
		r.AddCellf("Item %d", i+1)
		r.AddCellf("$%.2f", float64(i+1)*1.5)
	}

	if err := tb.Render(); err != nil {
		t.Fatalf("render: %v", err)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		t.Fatalf("output: %v", err)
	}

	if pdf.PageNo() < 2 {
		t.Error("expected at least 2 pages with 50 rows")
	}
	t.Logf("Multi-page table: %d pages, %d bytes", pdf.PageNo(), buf.Len())
}

func TestColspan(t *testing.T) {
	pdf := newTestPDF()

	tb := table.New(pdf)
	tb.SetColumnWidths(40, 40, 40, 40)

	r1 := tb.AddRow()
	r1.AddCell("Spans 2 cols").SetColspan(2)
	r1.AddCell("Normal")
	r1.AddCell("Normal")

	r2 := tb.AddRow()
	r2.AddCell("A")
	r2.AddCell("B")
	r2.AddCell("C")
	r2.AddCell("D")

	if err := tb.Render(); err != nil {
		t.Fatalf("render: %v", err)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		t.Fatalf("output: %v", err)
	}
	t.Logf("Colspan table PDF: %d bytes", buf.Len())
}

func TestStyledCells(t *testing.T) {
	pdf := newTestPDF()

	tb := table.New(pdf)
	tb.SetColumnWidths(60, 60, 60)
	tb.SetStyle(table.TableStyle{
		HeaderStyle: &table.CellStyle{
			FillColor: &table.RGBColor{0, 51, 102},
			TextColor: &table.RGBColor{255, 255, 255},
			Font:      &table.FontSpec{Family: "Helvetica", Style: "B", Size: 11},
		},
	})
	tb.SetHeaderRows(1)

	h := tb.AddHeaderRow()
	h.AddCell("Product")
	h.AddCell("Category")
	h.AddCell("Price")

	r := tb.AddRow()
	r.AddCell("Widget")
	r.AddCell("Hardware")
	r.AddCell("$5.00").SetAlign("R")

	if err := tb.Render(); err != nil {
		t.Fatalf("render: %v", err)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		t.Fatalf("output: %v", err)
	}
	t.Logf("Styled table PDF: %d bytes", buf.Len())
}

func TestEmptyTable(t *testing.T) {
	pdf := newTestPDF()

	tb := table.New(pdf)
	tb.SetColumnWidths(60, 60)

	// No rows added - should not panic
	if err := tb.Render(); err != nil {
		t.Fatalf("render empty table: %v", err)
	}
}

func TestNewDocumentWithTable(t *testing.T) {
	// Test integration with the new NewDocument constructor
	pdf := gofpdf.NewDocument(
		gofpdf.WithPageSize(gofpdf.PageSizeA4),
		gofpdf.WithUnit(gofpdf.UnitMillimeter),
	)
	pdf.SetFont("Helvetica", "", 10)
	pdf.AddPage()

	tb := table.New(pdf)
	tb.SetColumnWidths(90, 90)

	r := tb.AddRow()
	r.AddCell("Column 1")
	r.AddCell("Column 2")

	if err := tb.Render(); err != nil {
		t.Fatalf("render: %v", err)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		t.Fatalf("output: %v", err)
	}
	t.Logf("NewDocument + Table PDF: %d bytes", buf.Len())
}
