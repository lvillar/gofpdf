package doctpl_test

import (
	"bytes"
	"fmt"

	"github.com/lvillar/gofpdf/doctpl"
)

func ExampleRender() {
	template := `{
		"title": "Invoice #1234",
		"author": "Acme Corp",
		"pageSize": "Letter",
		"margin": {"top": 20, "right": 15, "bottom": 20, "left": 15},
		"font": {"family": "Helvetica", "size": 11},
		"header": {"text": "Acme Corp - Invoice", "align": "R"},
		"footer": {"text": "Page {page}", "align": "C"},
		"pages": [{
			"elements": [
				{"type": "heading", "text": "Invoice #1234", "level": 1},
				{"type": "paragraph", "text": "Date: 2024-01-15\nBill To: John Doe\n123 Main St"},
				{"type": "hr"},
				{
					"type": "table",
					"columns": [
						{"header": "Item", "width": 80},
						{"header": "Description"},
						{"header": "Qty", "width": 20, "align": "C"},
						{"header": "Price", "width": 30, "align": "R"}
					],
					"rows": [
						["WDG-001", "Premium Widget", "10", "$5.00"],
						["WDG-002", "Deluxe Widget", "5", "$12.00"],
						["SVC-001", "Installation Service", "1", "$50.00"]
					]
				},
				{"type": "spacer", "spacerHeight": 10},
				{"type": "paragraph", "text": "Total: $160.00", "align": "R", "font": {"style": "B", "size": 14}},
				{"type": "hr"},
				{"type": "paragraph", "text": "Thank you for your business!", "align": "C", "color": {"r": 100, "g": 100, "b": 100}}
			]
		}]
	}`

	var buf bytes.Buffer
	if err := doctpl.Render(&buf, []byte(template)); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Generated PDF: %d bytes\n", buf.Len())
	// Output pattern: Generated PDF: NNNN bytes
}

func ExampleRenderDocument() {
	doc := &doctpl.Document{
		Title:    "Quick Report",
		PageSize: "A4",
		Pages: []doctpl.Page{{
			Elements: []doctpl.Element{
				{Type: "heading", Text: "Monthly Report", Level: 1},
				{Type: "paragraph", Text: "This report covers the activities for the current month."},
				{
					Type: "list",
					Items: []string{
						"Revenue increased by 15%",
						"New customer acquisitions up 20%",
						"Customer satisfaction at 94%",
					},
				},
				{Type: "hr"},
				{Type: "paragraph", Text: "Prepared by: Analytics Team", Align: "R"},
			},
		}},
	}

	var buf bytes.Buffer
	if err := doctpl.RenderDocument(&buf, doc); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Generated PDF: %d bytes\n", buf.Len())
	// Output pattern: Generated PDF: NNNN bytes
}
