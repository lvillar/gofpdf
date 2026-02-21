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

func createPDFTool() Tool {
	return Tool{
		Name:        "create_pdf",
		Description: "Create a PDF document from a JSON template. The template supports headings, paragraphs, tables, images, lists, horizontal rules, and spacers. Returns the PDF as base64.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"template": map[string]interface{}{
					"type":        "object",
					"description": "JSON document template with title, pageSize, pages, and elements",
				},
				"outputPath": map[string]interface{}{
					"type":        "string",
					"description": "Optional file path to save the PDF. If omitted, returns base64.",
				},
			},
			"required": []string{"template"},
		},
		Handler: handleCreatePDF,
	}
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
