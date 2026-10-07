package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"orders_backend/repository"

	"github.com/go-pdf/fpdf"
	"github.com/xuri/excelize/v2"
)

var ErrInvalidExportFormat = errors.New("invalid export format")

type ReportExport struct {
	Content     []byte
	ContentType string
	Extension   string
}

func (s *Service) ExportDeliverySchedule(ctx context.Context, principal repository.Principal, start, end, generatedAt time.Time, format string) (ReportExport, error) {
	format = strings.ToLower(strings.TrimSpace(format))
	if format != "pdf" && format != "xlsx" {
		return ReportExport{}, ErrInvalidExportFormat
	}
	items, err := s.repository.ListDeliverySchedule(ctx, principal.SalespersonIDs, start, end)
	if err != nil {
		return ReportExport{}, err
	}
	var (
		content     []byte
		contentType string
	)
	if format == "xlsx" {
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		content, err = buildDeliveryXLSX(items, start, end, generatedAt)
	} else {
		contentType = "application/pdf"
		content, err = buildDeliveryPDF(items, start, end, generatedAt)
	}
	if err != nil {
		return ReportExport{}, err
	}
	return ReportExport{Content: content, ContentType: contentType, Extension: format}, nil
}

func buildDeliveryXLSX(items []repository.DeliveryRow, start, end time.Time, generatedAt time.Time) ([]byte, error) {
	file := excelize.NewFile()
	defer file.Close()
	sheet := "Delivery Schedule"
	if err := file.SetSheetName("Sheet1", sheet); err != nil {
		return nil, err
	}
	rows := [][]any{
		{"Delivery schedule"},
		{"Period", start.Format("2006-01-02"), "to", end.Format("2006-01-02")},
		{"Generated at", generatedAt.Format(time.RFC3339)},
		{},
		{"Delivery date", "Order no", "Customer", "Product code", "Product", "Quantity to deliver", "Delivery status"},
	}
	for _, item := range items {
		rows = append(rows, []any{item.DeliveryDate, item.OrderNo, item.CustomerName, item.ProductCode, item.ProductName, item.QuantityToDeliver, item.DeliveryStatus})
	}
	for rowIndex, values := range rows {
		for columnIndex, value := range values {
			cell, err := excelize.CoordinatesToCellName(columnIndex+1, rowIndex+1)
			if err != nil {
				return nil, err
			}
			if err := file.SetCellValue(sheet, cell, value); err != nil {
				return nil, err
			}
		}
	}
	if err := file.SetColWidth(sheet, "A", "G", 20); err != nil {
		return nil, err
	}
	if err := file.SetColWidth(sheet, "C", "C", 32); err != nil {
		return nil, err
	}
	if err := file.SetColWidth(sheet, "E", "E", 38); err != nil {
		return nil, err
	}
	headerStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}, Fill: excelize.Fill{Type: "pattern", Color: []string{"#DDE8E1"}, Pattern: 1}})
	if err != nil {
		return nil, err
	}
	if err := file.SetCellStyle(sheet, "A5", "G5", headerStyle); err != nil {
		return nil, err
	}
	buffer, err := file.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func buildDeliveryPDF(items []repository.DeliveryRow, start, end time.Time, generatedAt time.Time) ([]byte, error) {
	fontData, err := reportFont()
	if err != nil {
		return nil, err
	}
	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.AddUTF8FontFromBytes("ReportThai", "", fontData)
	if pdf.Err() {
		return nil, errors.New("could not load PDF font")
	}
	pdf.SetTitle("Delivery schedule", false)
	pdf.SetMargins(10, 10, 10)
	pdf.AddPage()
	pdf.SetFont("ReportThai", "", 15)
	pdf.CellFormat(0, 8, "Delivery schedule", "", 1, "L", false, 0, "")
	pdf.SetFont("ReportThai", "", 9)
	pdf.CellFormat(0, 6, fmt.Sprintf("Period: %s to %s", start.Format("2006-01-02"), end.Format("2006-01-02")), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, "Generated at: "+generatedAt.Format(time.RFC3339), "", 1, "L", false, 0, "")
	pdf.Ln(2)
	columns := []struct {
		title string
		width float64
		get   func(repository.DeliveryRow) string
	}{
		{"Delivery date", 28, func(row repository.DeliveryRow) string { return row.DeliveryDate }},
		{"Order no", 35, func(row repository.DeliveryRow) string { return row.OrderNo }},
		{"Customer", 47, func(row repository.DeliveryRow) string { return row.CustomerName }},
		{"Product code", 31, func(row repository.DeliveryRow) string { return row.ProductCode }},
		{"Product", 62, func(row repository.DeliveryRow) string { return row.ProductName }},
		{"Quantity", 25, func(row repository.DeliveryRow) string { return fmt.Sprint(row.QuantityToDeliver) }},
		{"Status", 39, func(row repository.DeliveryRow) string { return row.DeliveryStatus }},
	}
	widths := make([]float64, len(columns))
	for i, column := range columns {
		widths[i] = column.width
	}
	header := make([]string, len(columns))
	for i, column := range columns {
		header[i] = column.title
	}
	drawPDFRow(pdf, widths, header, true)
	for _, item := range items {
		values := make([]string, len(columns))
		for i, column := range columns {
			values[i] = column.get(item)
		}
		drawPDFRow(pdf, widths, values, false)
		if pdf.GetY() > 185 {
			pdf.AddPage()
			drawPDFRow(pdf, widths, header, true)
		}
	}
	var output bytes.Buffer
	if err := pdf.Output(&output); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func drawPDFRow(pdf *fpdf.Fpdf, widths []float64, values []string, header bool) {
	const lineHeight = 4.5
	pdf.SetFont("ReportThai", "", 8)
	if header {
		pdf.SetFillColor(221, 232, 225)
	} else {
		pdf.SetFillColor(255, 255, 255)
	}
	maxLines := 1
	for index, value := range values {
		if lines := len(pdf.SplitText(value, widths[index]-2)); lines > maxLines {
			maxLines = lines
		}
	}
	height := float64(maxLines)*lineHeight + 2
	rowX, startY := pdf.GetX(), pdf.GetY()
	startX := rowX
	for index, value := range values {
		pdf.SetXY(startX, startY)
		style := "D"
		if header {
			style = "FD"
		}
		pdf.Rect(startX, startY, widths[index], height, style)
		pdf.SetXY(startX+1, startY+1)
		pdf.MultiCell(widths[index]-2, lineHeight, value, "", "L", false)
		startX += widths[index]
	}
	pdf.SetXY(rowX, startY+height)
}

func reportFont() ([]byte, error) {
	paths := []string{os.Getenv("PDF_FONT_PATH")}
	if os.PathSeparator == '\\' {
		paths = append(paths, `C:\Windows\Fonts\tahoma.ttf`, `C:\Windows\Fonts\arial.ttf`)
	} else {
		paths = append(paths,
			"/usr/share/fonts/truetype/noto/NotoSansThai-Regular.ttf",
			"/usr/share/fonts/truetype/tlwg/TlwgTypo.ttf",
			"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		)
	}
	for _, path := range paths {
		if path == "" {
			continue
		}
		font, err := os.ReadFile(filepath.Clean(path))
		if err == nil {
			return font, nil
		}
	}
	return nil, fmt.Errorf("no usable PDF font found; set PDF_FONT_PATH to a Unicode TTF font")
}
