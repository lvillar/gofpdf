package mcp

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/lvillar/gofpdf/doctpl"
	"github.com/lvillar/gofpdf/form"
	"github.com/lvillar/gofpdf/pageops"
	"github.com/lvillar/gofpdf/reader"
)

// RegisterDefaultTools adds all built-in PDF tools to the server.
func RegisterDefaultTools(s *Server) {
	s.AddTool(createPDFTool())
	s.AddTool(validateTemplateTool())
	s.AddTool(readPDFTool())
	s.AddTool(readPDFTextTool())
	s.AddTool(mergePDFsTool())
	s.AddTool(addWatermarkTool())
	s.AddTool(addPageNumbersTool())
	s.AddTool(fillFormTool())
	s.AddTool(flattenFormTool())
	s.AddTool(rotatePDFTool())
	s.AddTool(pdfInfoTool())
}

// docTemplateSchema is the JSON Schema describing a doctpl Document. Exposing
// this in tool InputSchemas lets LLMs author templates correctly on the first
// try instead of guessing field names.
func docTemplateSchema() map[string]interface{} {
	color := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"r": map[string]interface{}{"type": "integer", "minimum": 0, "maximum": 255},
			"g": map[string]interface{}{"type": "integer", "minimum": 0, "maximum": 255},
			"b": map[string]interface{}{"type": "integer", "minimum": 0, "maximum": 255},
		},
		"required": []string{"r", "g", "b"},
	}
	font := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"family": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"Helvetica", "Courier", "Times"},
				"description": "Standard PDF font family.",
			},
			"style": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"", "B", "I", "BI"},
				"description": "Empty=regular, B=bold, I=italic, BI=bold italic.",
			},
			"size": map[string]interface{}{"type": "number", "description": "Size in points."},
		},
	}
	cellStyle := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"fillColor": color,
			"textColor": color,
			"font":      font,
		},
	}
	tableColumn := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"header": map[string]interface{}{"type": "string"},
			"width":  map[string]interface{}{"type": "number", "description": "Width in document units. 0 (or omitted) = auto."},
			"align":  map[string]interface{}{"type": "string", "enum": []string{"L", "C", "R"}},
		},
		"required": []string{"header"},
	}
	element := map[string]interface{}{
		"type":        "object",
		"description": "A single visual block on a page. The 'type' field selects which other fields apply.",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"heading", "paragraph", "text", "table", "image", "line", "rect", "spacer", "hr", "list"},
				"description": "Element kind.",
			},
			"text":  map[string]interface{}{"type": "string", "description": "Used by heading, paragraph, text."},
			"level": map[string]interface{}{"type": "integer", "minimum": 1, "maximum": 6, "description": "Heading level (1=largest)."},
			"align": map[string]interface{}{"type": "string", "enum": []string{"L", "C", "R"}},
			"font":  font,
			"color": color,

			"columns":     map[string]interface{}{"type": "array", "items": tableColumn, "description": "Table columns."},
			"rows":        map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}}, "description": "Table data rows; each row must have one cell per column."},
			"headerStyle": cellStyle,
			"cellStyle":   cellStyle,

			"src":    map[string]interface{}{"type": "string", "description": "Image file path or URL."},
			"x":      map[string]interface{}{"type": "number"},
			"y":      map[string]interface{}{"type": "number"},
			"width":  map[string]interface{}{"type": "number"},
			"height": map[string]interface{}{"type": "number"},

			"x1": map[string]interface{}{"type": "number"},
			"y1": map[string]interface{}{"type": "number"},
			"x2": map[string]interface{}{"type": "number"},
			"y2": map[string]interface{}{"type": "number"},

			"spacerHeight": map[string]interface{}{"type": "number", "description": "Vertical whitespace for spacer."},
			"lineWidth":    map[string]interface{}{"type": "number"},

			"items":   map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "List items."},
			"ordered": map[string]interface{}{"type": "boolean", "description": "Use numbers (1. 2. 3.) instead of bullets."},
			"bullet":  map[string]interface{}{"type": "string", "description": "Custom bullet character for unordered lists."},

			"fillColor": color,
			"border":    map[string]interface{}{"type": "boolean"},
		},
		"required": []string{"type"},
	}
	page := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"size":     map[string]interface{}{"type": "string", "description": "Per-page override of pageSize."},
			"elements": map[string]interface{}{"type": "array", "items": element},
		},
		"required": []string{"elements"},
	}
	return map[string]interface{}{
		"type":        "object",
		"description": "Declarative PDF document. Top-level fields configure the document; 'pages' contains the visual content.",
		"properties": map[string]interface{}{
			"title":    map[string]interface{}{"type": "string"},
			"author":   map[string]interface{}{"type": "string"},
			"subject":  map[string]interface{}{"type": "string"},
			"pageSize": map[string]interface{}{"type": "string", "enum": []string{"A4", "Letter", "Legal"}, "description": "Default page size."},
			"unit":     map[string]interface{}{"type": "string", "enum": []string{"mm", "cm", "in", "pt"}, "description": "Measurement unit for sizes/positions (default: mm)."},
			"margin": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"top":    map[string]interface{}{"type": "number"},
					"right":  map[string]interface{}{"type": "number"},
					"bottom": map[string]interface{}{"type": "number"},
					"left":   map[string]interface{}{"type": "number"},
				},
			},
			"font": font,
			"header": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"text":  map[string]interface{}{"type": "string"},
					"align": map[string]interface{}{"type": "string", "enum": []string{"L", "C", "R"}},
					"font":  font,
					"color": color,
				},
			},
			"footer": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"text":  map[string]interface{}{"type": "string", "description": "Supports '{page}' placeholder for current page number."},
					"align": map[string]interface{}{"type": "string", "enum": []string{"L", "C", "R"}},
					"font":  font,
					"color": color,
				},
			},
			"pages": map[string]interface{}{"type": "array", "items": page, "description": "One or more pages of content. Required."},
		},
		"required": []string{"pages"},
	}
}

func createPDFTool() Tool {
	return Tool{
		Name: "create_pdf",
		Description: "Create a PDF document from a declarative JSON template. " +
			"Templates support headings, paragraphs, tables, images, lists, lines, rects, spacers, and horizontal rules. " +
			"For an example invoice, see the doctpl.Render godoc. Returns the PDF as base64 by default; pass outputPath to write to disk instead. " +
			"Use validate_template first to lint a template without rendering.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"template": docTemplateSchema(),
				"outputPath": map[string]interface{}{
					"type":        "string",
					"description": "Optional file path to save the PDF. If omitted, the PDF is returned as base64 in the response.",
				},
			},
			"required": []string{"template"},
		},
		Handler: handleCreatePDF,
	}
}

func validateTemplateTool() Tool {
	return Tool{
		Name: "validate_template",
		Description: "Validate a doctpl JSON template without rendering. " +
			"Returns either a confirmation that the template is valid or a structured list of problems " +
			"(path, field, message) so an LLM can self-correct before calling create_pdf.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"template": docTemplateSchema(),
			},
			"required": []string{"template"},
		},
		Handler: handleValidateTemplate,
	}
}

func handleValidateTemplate(args map[string]interface{}) (ToolResult, error) {
	templateData, ok := args["template"]
	if !ok {
		return ToolResult{}, fmt.Errorf("missing 'template' argument")
	}

	jsonBytes, err := json.Marshal(templateData)
	if err != nil {
		return ToolResult{}, fmt.Errorf("encoding template: %w", err)
	}

	errs, err := doctpl.Validate(jsonBytes)
	if err != nil {
		return ToolResult{
			Content: []ContentBlock{{
				Type: "text",
				Text: fmt.Sprintf("Template is not valid JSON: %v", err),
			}},
			IsError: true,
		}, nil
	}

	if len(errs) == 0 {
		return ToolResult{
			Content: []ContentBlock{{
				Type: "text",
				Text: "Template is valid. You can pass it to create_pdf.",
			}},
		}, nil
	}

	report := map[string]interface{}{
		"valid":  false,
		"errors": errs,
	}
	out, _ := json.MarshalIndent(report, "", "  ")
	return ToolResult{
		Content: []ContentBlock{{
			Type: "text",
			Text: fmt.Sprintf("Template has %d validation error(s):\n%s", len(errs), string(out)),
		}},
	}, nil
}

func handleCreatePDF(args map[string]interface{}) (ToolResult, error) {
	templateData, ok := args["template"]
	if !ok {
		return ToolResult{}, fmt.Errorf("missing 'template' argument")
	}

	jsonBytes, err := json.Marshal(templateData)
	if err != nil {
		return ToolResult{}, fmt.Errorf("encoding template: %w", err)
	}

	var buf bytes.Buffer
	if err := doctpl.Render(&buf, jsonBytes); err != nil {
		return ToolResult{}, fmt.Errorf("rendering PDF: %w", err)
	}

	// Save to file if outputPath specified
	if outputPath, ok := args["outputPath"].(string); ok && outputPath != "" {
		if err := os.WriteFile(outputPath, buf.Bytes(), 0644); err != nil {
			return ToolResult{}, fmt.Errorf("writing file: %w", err)
		}
		return ToolResult{
			Content: []ContentBlock{{
				Type: "text",
				Text: fmt.Sprintf("PDF created successfully: %s (%d bytes)", outputPath, buf.Len()),
			}},
		}, nil
	}

	// Return as base64
	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	return ToolResult{
		Content: []ContentBlock{{
			Type: "text",
			Text: fmt.Sprintf("PDF created successfully (%d bytes). Base64 data:\n%s", buf.Len(), encoded),
		}},
	}, nil
}

func readPDFTool() Tool {
	return Tool{
		Name:        "read_pdf",
		Description: "Read a PDF file and return its metadata (title, author, page count, version).",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the PDF file",
				},
			},
			"required": []string{"path"},
		},
		Handler: handleReadPDF,
	}
}

func handleReadPDF(args map[string]interface{}) (ToolResult, error) {
	path, ok := args["path"].(string)
	if !ok {
		return ToolResult{}, fmt.Errorf("missing 'path' argument")
	}

	doc, err := reader.Open(path)
	if err != nil {
		return ToolResult{}, fmt.Errorf("opening PDF: %w", err)
	}

	meta := doc.Metadata()
	info := map[string]interface{}{
		"version":  doc.Version,
		"numPages": doc.NumPages(),
		"metadata": meta,
	}

	jsonBytes, _ := json.MarshalIndent(info, "", "  ")
	return ToolResult{
		Content: []ContentBlock{{Type: "text", Text: string(jsonBytes)}},
	}, nil
}

func readPDFTextTool() Tool {
	return Tool{
		Name:        "read_pdf_text",
		Description: "Extract text content from a PDF file. Returns the text from all pages or specific pages.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the PDF file",
				},
				"pages": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "number"},
					"description": "Specific page numbers to extract (1-based). Omit for all pages.",
				},
			},
			"required": []string{"path"},
		},
		Handler: handleReadPDFText,
	}
}

func handleReadPDFText(args map[string]interface{}) (ToolResult, error) {
	path, ok := args["path"].(string)
	if !ok {
		return ToolResult{}, fmt.Errorf("missing 'path' argument")
	}

	doc, err := reader.Open(path)
	if err != nil {
		return ToolResult{}, fmt.Errorf("opening PDF: %w", err)
	}

	// Determine which pages to extract
	pageSet := make(map[int]bool)
	if pagesArg, ok := args["pages"].([]interface{}); ok {
		for _, p := range pagesArg {
			if num, ok := p.(float64); ok {
				pageSet[int(num)] = true
			}
		}
	}

	var result strings.Builder
	for pageNum, page := range doc.Pages() {
		if len(pageSet) > 0 && !pageSet[pageNum] {
			continue
		}

		text, err := page.ExtractText()
		if err != nil {
			fmt.Fprintf(&result, "--- Page %d (error: %v) ---\n", pageNum, err)
			continue
		}

		fmt.Fprintf(&result, "--- Page %d ---\n%s\n\n", pageNum, text)
	}

	return ToolResult{
		Content: []ContentBlock{{Type: "text", Text: result.String()}},
	}, nil
}

func mergePDFsTool() Tool {
	return Tool{
		Name:        "merge_pdfs",
		Description: "Merge multiple PDF files into a single PDF.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"inputPaths": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Paths to PDF files to merge, in order",
				},
				"outputPath": map[string]interface{}{
					"type":        "string",
					"description": "Path for the merged output PDF",
				},
			},
			"required": []string{"inputPaths", "outputPath"},
		},
		Handler: handleMergePDFs,
	}
}

func handleMergePDFs(args map[string]interface{}) (ToolResult, error) {
	pathsRaw, ok := args["inputPaths"].([]interface{})
	if !ok {
		return ToolResult{}, fmt.Errorf("missing 'inputPaths' argument")
	}
	outputPath, ok := args["outputPath"].(string)
	if !ok {
		return ToolResult{}, fmt.Errorf("missing 'outputPath' argument")
	}

	paths := make([]string, len(pathsRaw))
	for i, p := range pathsRaw {
		paths[i], _ = p.(string)
	}

	if err := pageops.MergeFiles(outputPath, paths...); err != nil {
		return ToolResult{}, fmt.Errorf("merging: %w", err)
	}

	return ToolResult{
		Content: []ContentBlock{{
			Type: "text",
			Text: fmt.Sprintf("Merged %d PDFs into %s", len(paths), outputPath),
		}},
	}, nil
}

func addWatermarkTool() Tool {
	return Tool{
		Name:        "add_watermark",
		Description: "Add a text watermark to a PDF file.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"inputPath": map[string]interface{}{
					"type":        "string",
					"description": "Path to the input PDF",
				},
				"outputPath": map[string]interface{}{
					"type":        "string",
					"description": "Path for the output PDF",
				},
				"text": map[string]interface{}{
					"type":        "string",
					"description": "Watermark text (e.g. 'CONFIDENTIAL', 'DRAFT')",
				},
				"fontSize": map[string]interface{}{
					"type":        "number",
					"description": "Font size in points (default: 60)",
				},
				"opacity": map[string]interface{}{
					"type":        "number",
					"description": "Opacity from 0.0 to 1.0 (default: 0.3)",
				},
				"angle": map[string]interface{}{
					"type":        "number",
					"description": "Rotation angle in degrees (default: 45)",
				},
			},
			"required": []string{"inputPath", "outputPath", "text"},
		},
		Handler: handleAddWatermark,
	}
}

func handleAddWatermark(args map[string]interface{}) (ToolResult, error) {
	inputPath, _ := args["inputPath"].(string)
	outputPath, _ := args["outputPath"].(string)
	text, _ := args["text"].(string)

	if inputPath == "" || outputPath == "" || text == "" {
		return ToolResult{}, fmt.Errorf("inputPath, outputPath, and text are required")
	}

	wm := pageops.TextWatermark{Text: text}
	if fs, ok := args["fontSize"].(float64); ok {
		wm.FontSize = fs
	}
	if op, ok := args["opacity"].(float64); ok {
		wm.Opacity = op
	}
	if angle, ok := args["angle"].(float64); ok {
		wm.Angle = angle
	}

	if err := pageops.AddTextWatermarkToFile(inputPath, outputPath, wm); err != nil {
		return ToolResult{}, err
	}

	return ToolResult{
		Content: []ContentBlock{{
			Type: "text",
			Text: fmt.Sprintf("Watermark '%s' added to %s -> %s", text, inputPath, outputPath),
		}},
	}, nil
}

func addPageNumbersTool() Tool {
	return Tool{
		Name:        "add_page_numbers",
		Description: "Add page numbers to a PDF file.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"inputPath": map[string]interface{}{
					"type":        "string",
					"description": "Path to the input PDF",
				},
				"outputPath": map[string]interface{}{
					"type":        "string",
					"description": "Path for the output PDF",
				},
				"format": map[string]interface{}{
					"type":        "string",
					"description": "Format string, e.g. 'Page %d of %d' (default: 'Page %d of %d')",
				},
				"position": map[string]interface{}{
					"type":        "string",
					"description": "Position: bottom-center, bottom-left, bottom-right, top-center, top-left, top-right",
				},
			},
			"required": []string{"inputPath", "outputPath"},
		},
		Handler: handleAddPageNumbers,
	}
}

func handleAddPageNumbers(args map[string]interface{}) (ToolResult, error) {
	inputPath, _ := args["inputPath"].(string)
	outputPath, _ := args["outputPath"].(string)

	if inputPath == "" || outputPath == "" {
		return ToolResult{}, fmt.Errorf("inputPath and outputPath are required")
	}

	style := pageops.PageNumberStyle{}
	if f, ok := args["format"].(string); ok {
		style.Format = f
	}
	if pos, ok := args["position"].(string); ok {
		style.Position = parsePosition(pos)
	}

	if err := pageops.AddPageNumbersToFile(inputPath, outputPath, style); err != nil {
		return ToolResult{}, err
	}

	return ToolResult{
		Content: []ContentBlock{{
			Type: "text",
			Text: fmt.Sprintf("Page numbers added to %s -> %s", inputPath, outputPath),
		}},
	}, nil
}

func fillFormTool() Tool {
	return Tool{
		Name:        "fill_form",
		Description: "Fill form fields in a PDF with provided values.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"inputPath": map[string]interface{}{
					"type":        "string",
					"description": "Path to the input PDF with form fields",
				},
				"outputPath": map[string]interface{}{
					"type":        "string",
					"description": "Path for the filled output PDF",
				},
				"values": map[string]interface{}{
					"type":        "object",
					"description": "Map of field names to values",
				},
			},
			"required": []string{"inputPath", "outputPath", "values"},
		},
		Handler: handleFillForm,
	}
}

func handleFillForm(args map[string]interface{}) (ToolResult, error) {
	inputPath, _ := args["inputPath"].(string)
	outputPath, _ := args["outputPath"].(string)
	valuesRaw, _ := args["values"].(map[string]interface{})

	if inputPath == "" || outputPath == "" {
		return ToolResult{}, fmt.Errorf("inputPath and outputPath are required")
	}

	values := make(map[string]string)
	for k, v := range valuesRaw {
		values[k] = fmt.Sprintf("%v", v)
	}

	if err := form.FillFile(inputPath, outputPath, values); err != nil {
		return ToolResult{}, err
	}

	return ToolResult{
		Content: []ContentBlock{{
			Type: "text",
			Text: fmt.Sprintf("Filled %d form fields in %s -> %s", len(values), inputPath, outputPath),
		}},
	}, nil
}

func flattenFormTool() Tool {
	return Tool{
		Name:        "flatten_form",
		Description: "Flatten a PDF form, making form fields non-editable and embedding their values as static content.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"inputPath": map[string]interface{}{
					"type":        "string",
					"description": "Path to the input PDF with form fields",
				},
				"outputPath": map[string]interface{}{
					"type":        "string",
					"description": "Path for the flattened output PDF",
				},
			},
			"required": []string{"inputPath", "outputPath"},
		},
		Handler: handleFlattenForm,
	}
}

func handleFlattenForm(args map[string]interface{}) (ToolResult, error) {
	inputPath, _ := args["inputPath"].(string)
	outputPath, _ := args["outputPath"].(string)

	if inputPath == "" || outputPath == "" {
		return ToolResult{}, fmt.Errorf("inputPath and outputPath are required")
	}

	input, err := os.Open(inputPath)
	if err != nil {
		return ToolResult{}, err
	}
	defer input.Close()

	output, err := os.Create(outputPath)
	if err != nil {
		return ToolResult{}, err
	}
	defer output.Close()

	if err := form.Flatten(input, output); err != nil {
		return ToolResult{}, err
	}

	return ToolResult{
		Content: []ContentBlock{{
			Type: "text",
			Text: fmt.Sprintf("Form flattened: %s -> %s", inputPath, outputPath),
		}},
	}, nil
}

func rotatePDFTool() Tool {
	return Tool{
		Name:        "rotate_pages",
		Description: "Rotate pages in a PDF by a specified angle (90, 180, or 270 degrees).",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"inputPath": map[string]interface{}{
					"type":        "string",
					"description": "Path to the input PDF",
				},
				"outputPath": map[string]interface{}{
					"type":        "string",
					"description": "Path for the output PDF",
				},
				"angle": map[string]interface{}{
					"type":        "number",
					"description": "Rotation angle: 90, 180, or 270",
				},
				"pages": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "number"},
					"description": "Page numbers to rotate (1-based). Omit for all pages.",
				},
			},
			"required": []string{"inputPath", "outputPath", "angle"},
		},
		Handler: handleRotatePDF,
	}
}

func handleRotatePDF(args map[string]interface{}) (ToolResult, error) {
	inputPath, _ := args["inputPath"].(string)
	outputPath, _ := args["outputPath"].(string)
	angleF, _ := args["angle"].(float64)
	angle := int(angleF)

	if inputPath == "" || outputPath == "" {
		return ToolResult{}, fmt.Errorf("inputPath, outputPath, and angle are required")
	}

	var pages []int
	if pagesRaw, ok := args["pages"].([]interface{}); ok {
		for _, p := range pagesRaw {
			if num, ok := p.(float64); ok {
				pages = append(pages, int(num))
			}
		}
	}

	if err := pageops.RotatePagesToFile(inputPath, outputPath, angle, pages); err != nil {
		return ToolResult{}, err
	}

	pagesDesc := "all pages"
	if len(pages) > 0 {
		pagesDesc = fmt.Sprintf("pages %v", pages)
	}

	return ToolResult{
		Content: []ContentBlock{{
			Type: "text",
			Text: fmt.Sprintf("Rotated %s by %d degrees in %s -> %s", pagesDesc, angle, inputPath, outputPath),
		}},
	}, nil
}

func pdfInfoTool() Tool {
	return Tool{
		Name:        "pdf_info",
		Description: "Get detailed information about a PDF file including page count, form fields, version, and metadata.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the PDF file",
				},
			},
			"required": []string{"path"},
		},
		Handler: handlePDFInfo,
	}
}

func handlePDFInfo(args map[string]interface{}) (ToolResult, error) {
	path, ok := args["path"].(string)
	if !ok {
		return ToolResult{}, fmt.Errorf("missing 'path' argument")
	}

	doc, err := reader.Open(path)
	if err != nil {
		return ToolResult{}, fmt.Errorf("opening PDF: %w", err)
	}

	info := map[string]interface{}{
		"version":  doc.Version,
		"numPages": doc.NumPages(),
		"metadata": doc.Metadata(),
	}

	// Check for form fields
	fields, err := doc.FormFields()
	if err == nil && len(fields) > 0 {
		fieldInfo := make([]map[string]interface{}, 0)
		for _, f := range flattenFormFields(fields) {
			fieldInfo = append(fieldInfo, map[string]interface{}{
				"name":  f.FullName,
				"type":  f.Type,
				"value": f.Value,
			})
		}
		info["formFields"] = fieldInfo
	}

	// Page dimensions
	pageInfos := make([]map[string]interface{}, 0)
	for pageNum, page := range doc.Pages() {
		mb := page.MediaBox
		pageInfos = append(pageInfos, map[string]interface{}{
			"page":   pageNum,
			"width":  mb.Width(),
			"height": mb.Height(),
		})
	}
	info["pages"] = pageInfos

	jsonBytes, _ := json.MarshalIndent(info, "", "  ")
	return ToolResult{
		Content: []ContentBlock{{Type: "text", Text: string(jsonBytes)}},
	}, nil
}

// flattenFormFields recursively collects all form fields.
func flattenFormFields(fields []*reader.FormField) []*reader.FormField {
	var result []*reader.FormField
	for _, f := range fields {
		result = append(result, f)
		if len(f.Kids) > 0 {
			result = append(result, flattenFormFields(f.Kids)...)
		}
	}
	return result
}

func parsePosition(s string) pageops.Position {
	switch strings.ToLower(strings.ReplaceAll(s, "-", "")) {
	case "topleft":
		return pageops.TopLeft
	case "topcenter":
		return pageops.TopCenter
	case "topright":
		return pageops.TopRight
	case "bottomleft":
		return pageops.BottomLeft
	case "bottomright":
		return pageops.BottomRight
	case "center":
		return pageops.Center
	default:
		return pageops.BottomCenter
	}
}
