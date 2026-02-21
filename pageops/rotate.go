package pageops

import (
	"fmt"
	"io"

	gofpdf "github.com/lvillar/gofpdf"
	"github.com/lvillar/gofpdf/contrib/gofpdi"
)

// RotatePages rotates specific pages by the given angle (90, 180, or 270 degrees).
// If pages is nil, all pages are rotated.
func RotatePages(w io.Writer, inputPath string, angle int, pages []int) error {
	pdf, err := buildRotatedPDF(inputPath, angle, pages)
	if err != nil {
		return err
	}
	return writePDF(pdf, w)
}

// RotatePagesToFile rotates pages and saves to a file.
func RotatePagesToFile(inputPath, outputPath string, angle int, pages []int) error {
	pdf, err := buildRotatedPDF(inputPath, angle, pages)
	if err != nil {
		return err
	}
	return writePDFToFile(pdf, outputPath)
}

func buildRotatedPDF(inputPath string, angle int, pages []int) (*gofpdf.Fpdf, error) {
	if angle != 90 && angle != 180 && angle != 270 {
		return nil, fmt.Errorf("pageops: rotation angle must be 90, 180, or 270, got %d", angle)
	}

	pageCount, err := getPageCount(inputPath)
	if err != nil {
		return nil, err
	}

	rotatePages := make(map[int]bool)
	if pages == nil {
		for i := 1; i <= pageCount; i++ {
			rotatePages[i] = true
		}
	} else {
		for _, p := range pages {
			rotatePages[p] = true
		}
	}

	pdf := gofpdf.New("P", "pt", "A4", "")
	pdf.SetAutoPageBreak(false, 0)
	imp := gofpdi.NewImporter()

	for i := 1; i <= pageCount; i++ {
		tplID, pw, ph := importPage(pdf, imp, inputPath, i)
		if pw == 0 || ph == 0 {
			pw = 595.28
			ph = 841.89
		}

		if rotatePages[i] {
			// For 90/270 degree rotation, swap page dimensions
			if angle == 90 || angle == 270 {
				pdf.AddPageFormat("P", gofpdf.SizeType{Wd: ph, Ht: pw})
			} else {
				pdf.AddPageFormat("P", gofpdf.SizeType{Wd: pw, Ht: ph})
			}

			// Apply transformation
			cx := pw / 2
			cy := ph / 2

			pdf.TransformBegin()

			switch angle {
			case 90:
				// Rotate 90 degrees: translate to fit new dimensions
				pdf.TransformRotate(-90, 0, 0)
				pdf.TransformTranslate(0, pw)
			case 180:
				pdf.TransformRotate(-180, cx, cy)
			case 270:
				pdf.TransformRotate(-270, 0, 0)
				pdf.TransformTranslate(ph, 0)
			}

			imp.UseImportedTemplate(pdf, tplID, 0, 0, pw, ph)
			pdf.TransformEnd()
		} else {
			pdf.AddPageFormat("P", gofpdf.SizeType{Wd: pw, Ht: ph})
			imp.UseImportedTemplate(pdf, tplID, 0, 0, pw, ph)
		}
	}

	if pdf.Err() {
		return nil, fmt.Errorf("pageops: rotate: %w", pdf.Error())
	}
	return pdf, nil
}
