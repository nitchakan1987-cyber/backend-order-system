package service

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"orders_backend/repository"

	"github.com/xuri/excelize/v2"
)

func TestBuildDeliveryXLSX(t *testing.T) {
	start, _ := time.Parse("2006-01-02", "2026-10-01")
	end, _ := time.Parse("2006-01-02", "2026-10-31")
	generatedAt := time.Date(2026, 10, 7, 12, 0, 0, 0, time.FixedZone("Asia/Bangkok", 7*60*60))
	content, err := buildDeliveryXLSX([]repository.DeliveryRow{{
		DeliveryDate: "2026-10-15", OrderID: 103, OrderNo: "BPI-2026100003",
		CustomerID: 1, CustomerName: "ลูกค้า A", ProductID: 1, ProductCode: "PROD-001",
		ProductName: "สินค้า A", QuantityToDeliver: 2, DeliveryStatus: "NOT_SHIPPED",
	}}, start, end, generatedAt)
	if err != nil {
		t.Fatalf("buildDeliveryXLSX returned error: %v", err)
	}
	if !bytes.HasPrefix(content, []byte("PK")) {
		t.Fatal("XLSX output does not have a ZIP signature")
	}
	file, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		t.Fatalf("open generated XLSX: %v", err)
	}
	defer file.Close()
	value, err := file.GetCellValue("Delivery Schedule", "E6")
	if err != nil {
		t.Fatalf("read generated XLSX cell: %v", err)
	}
	if !strings.Contains(value, "สินค้า") {
		t.Fatalf("product name cell = %q, want Thai product name", value)
	}
}

func TestBuildDeliveryPDF(t *testing.T) {
	if _, err := reportFont(); err != nil {
		t.Skip(err)
	}
	start, _ := time.Parse("2006-01-02", "2026-10-01")
	end, _ := time.Parse("2006-01-02", "2026-10-31")
	generatedAt := time.Date(2026, 10, 7, 12, 0, 0, 0, time.FixedZone("Asia/Bangkok", 7*60*60))
	content, err := buildDeliveryPDF([]repository.DeliveryRow{{
		DeliveryDate: "2026-10-15", OrderID: 103, OrderNo: "BPI-2026100003",
		CustomerID: 1, CustomerName: "ลูกค้า A", ProductID: 1, ProductCode: "PROD-001",
		ProductName: "สินค้า A", QuantityToDeliver: 2, DeliveryStatus: "NOT_SHIPPED",
	}}, start, end, generatedAt)
	if err != nil {
		t.Fatalf("buildDeliveryPDF returned error: %v", err)
	}
	if !bytes.HasPrefix(content, []byte("%PDF-")) {
		t.Fatal("PDF output does not have a PDF signature")
	}
}
