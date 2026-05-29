package handlers

import (
	"strings"
	"testing"
	"time"
)

// csvBytes joins lines with \n and returns UTF-8 bytes suitable for parseBatchCSV.
func csvBytes(lines ...string) []byte {
	return []byte(strings.Join(lines, "\n"))
}

func TestParseBatchCSV_ISODate(t *testing.T) {
	data := csvBytes(
		"product_name,batch_number,expiry_date,quantity,purchase_price,selling_price",
		"Aspirin 500mg,BTH-001,2026-05-30,100,2.50,4.50",
	)
	batches, total, errors := parseBatchCSV(data)
	if len(errors) != 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}
	if total != 1 || len(batches) != 1 {
		t.Fatalf("expected 1 batch, got %d (total=%d)", len(batches), total)
	}
	want := time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC)
	if !batches[0].ExpiryDate.Equal(want) {
		t.Errorf("expiry date: got %v, want %v", batches[0].ExpiryDate, want)
	}
}

func TestParseBatchCSV_EuropeanDateSlash(t *testing.T) {
	data := csvBytes(
		"product_name,expiry_date,quantity,purchase_price,selling_price",
		"Ibuprofen 400mg,30/05/2026,50,0.80,1.80",
	)
	batches, _, errors := parseBatchCSV(data)
	if len(errors) != 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}
	want := time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC)
	if !batches[0].ExpiryDate.Equal(want) {
		t.Errorf("expiry date: got %v, want %v", batches[0].ExpiryDate, want)
	}
}

func TestParseBatchCSV_EuropeanDateDash(t *testing.T) {
	data := csvBytes(
		"product_name,expiry_date,quantity,purchase_price,selling_price",
		"Paracetamol,30-05-2026,20,1.00,2.00",
	)
	batches, _, errors := parseBatchCSV(data)
	if len(errors) != 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}
	want := time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC)
	if !batches[0].ExpiryDate.Equal(want) {
		t.Errorf("expiry date: got %v, want %v", batches[0].ExpiryDate, want)
	}
}

func TestParseBatchCSV_USDate(t *testing.T) {
	data := csvBytes(
		"product_name,expiry_date,quantity,purchase_price,selling_price",
		"Amoxicillin,05/30/2026,30,1.50,3.00",
	)
	batches, _, errors := parseBatchCSV(data)
	if len(errors) != 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}
	want := time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC)
	if !batches[0].ExpiryDate.Equal(want) {
		t.Errorf("expiry date: got %v, want %v", batches[0].ExpiryDate, want)
	}
}

func TestParseBatchCSV_DateWithWhitespace(t *testing.T) {
	data := csvBytes(
		"product_name,expiry_date,quantity,purchase_price,selling_price",
		"Aspirin,  2026-05-30  ,100,2.50,4.50",
	)
	batches, _, errors := parseBatchCSV(data)
	if len(errors) != 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}
	want := time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC)
	if !batches[0].ExpiryDate.Equal(want) {
		t.Errorf("expiry date: got %v, want %v", batches[0].ExpiryDate, want)
	}
}

func TestParseBatchCSV_BOM(t *testing.T) {
	// UTF-8 BOM prefix on first header field (Excel artefact)
	data := []byte("\xef\xbb\xbfproduct_name,expiry_date,quantity,purchase_price,selling_price\nAspirin,2026-05-30,100,2.50,4.50")
	batches, _, errors := parseBatchCSV(data)
	if len(errors) != 0 {
		t.Fatalf("BOM CSV: unexpected errors: %v", errors)
	}
	if len(batches) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(batches))
	}
	if batches[0].ProductName != "Aspirin" {
		t.Errorf("product name: got %q", batches[0].ProductName)
	}
}

func TestParseBatchCSV_MixedCaseHeaders(t *testing.T) {
	data := csvBytes(
		"Product_Name,Expiry_Date,Quantity,Purchase_Price,Selling_Price",
		"Aspirin,2026-05-30,100,2.50,4.50",
	)
	batches, _, errors := parseBatchCSV(data)
	if len(errors) != 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}
	if len(batches) != 1 {
		t.Fatalf("expected 1 batch")
	}
}

func TestParseBatchCSV_AliasHeaders(t *testing.T) {
	// "name", "expires", "qty", "cost", "price" are all valid aliases
	data := csvBytes(
		"name,batch,expires,qty,cost,price",
		"Aspirin,BTH-001,2026-12-31,50,1.00,2.00",
	)
	batches, _, errors := parseBatchCSV(data)
	if len(errors) != 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}
	if len(batches) != 1 {
		t.Fatalf("expected 1 batch")
	}
	if batches[0].BatchNumber != "BTH-001" {
		t.Errorf("batch number: got %q", batches[0].BatchNumber)
	}
}

func TestParseBatchCSV_MissingRequiredColumn(t *testing.T) {
	// No expiry_date column
	data := csvBytes(
		"product_name,quantity,purchase_price,selling_price",
		"Aspirin,100,2.50,4.50",
	)
	_, total, errors := parseBatchCSV(data)
	if total != 0 || len(errors) == 0 {
		t.Fatalf("expected fatal error for missing expiry_date, got total=%d errors=%v", total, errors)
	}
	if !strings.Contains(errors[0], "expiry_date") {
		t.Errorf("error should mention missing column, got: %s", errors[0])
	}
}

func TestParseBatchCSV_InvalidDate(t *testing.T) {
	data := csvBytes(
		"product_name,expiry_date,quantity,purchase_price,selling_price",
		"Aspirin 500mg,not-a-date,100,2.50,4.50",
	)
	_, total, errors := parseBatchCSV(data)
	if total != 1 || len(errors) != 1 {
		t.Fatalf("expected 1 row error, got total=%d errors=%v", total, errors)
	}
	// Error must mention the bad value AND the product name
	if !strings.Contains(errors[0], "not-a-date") {
		t.Errorf("error should show the bad date value; got: %s", errors[0])
	}
	if !strings.Contains(errors[0], "Aspirin 500mg") {
		t.Errorf("error should show the product name; got: %s", errors[0])
	}
}

func TestParseBatchCSV_RichTemplateFormat(t *testing.T) {
	// 11-column format as per the Download Template
	data := csvBytes(
		"barcode,name,name_el,category,manufacturer,batch_number,expiry_date,quantity,purchase_price,selling_price,supplier",
		"5290000099001,Aspirin 500mg,Ασπιρίνη 500mg,Painkillers,Bayer,BTH-EXAMPLE,2026-12-31,100,2.50,4.50,MedSupply Cyprus",
	)
	batches, _, errors := parseBatchCSV(data)
	if len(errors) != 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}
	b := batches[0]
	if b.ProductName != "Aspirin 500mg" {
		t.Errorf("name: %q", b.ProductName)
	}
	if b.ProductNameEl != "Ασπιρίνη 500mg" {
		t.Errorf("name_el: %q", b.ProductNameEl)
	}
	if b.Barcode != "5290000099001" {
		t.Errorf("barcode: %q", b.Barcode)
	}
	if b.Supplier != "MedSupply Cyprus" {
		t.Errorf("supplier: %q", b.Supplier)
	}
	if b.Category != "Painkillers" {
		t.Errorf("category: %q", b.Category)
	}
}

func TestParseBatchCSV_CommaDecimalSeparator(t *testing.T) {
	// Quoted fields with comma decimal separator (e.g. "2,50" → 2.50)
	data := csvBytes(
		"product_name,expiry_date,quantity,purchase_price,selling_price",
		`Aspirin,2026-12-31,100,"2,50","4,50"`,
	)
	batches, _, errors := parseBatchCSV(data)
	if len(errors) != 0 {
		t.Fatalf("unexpected errors: %v", errors)
	}
	if batches[0].PurchasePrice != 2.50 {
		t.Errorf("purchase_price: got %v, want 2.50", batches[0].PurchasePrice)
	}
}

func TestParseBatchCSV_MultipleRowErrors(t *testing.T) {
	data := csvBytes(
		"product_name,expiry_date,quantity,purchase_price,selling_price",
		"Aspirin,2026-05-30,100,2.50,4.50",     // ok
		"Ibuprofen,bad-date,50,0.80,1.80",       // bad date
		"Paracetamol,2026-06-01,abc,1.00,2.00",  // bad qty
		"Omeprazole,2026-07-15,20,1.50,3.00",    // ok
	)
	batches, total, errors := parseBatchCSV(data)
	if total != 4 {
		t.Errorf("total: got %d, want 4", total)
	}
	if len(batches) != 2 {
		t.Errorf("valid batches: got %d, want 2", len(batches))
	}
	if len(errors) != 2 {
		t.Errorf("errors: got %d, want 2 — %v", len(errors), errors)
	}
}

func TestParseDate_AllFormats(t *testing.T) {
	cases := []struct {
		input string
		year  int
		month time.Month
		day   int
	}{
		{"2026-05-30", 2026, time.May, 30},
		{"30/05/2026", 2026, time.May, 30},
		{"30-05-2026", 2026, time.May, 30},
		{"05/30/2026", 2026, time.May, 30},
		{"2026/05/30", 2026, time.May, 30},
		{"30 May 2026", 2026, time.May, 30},
		{"30 May 2026", 2026, time.May, 30},
		{"  2026-05-30  ", 2026, time.May, 30}, // whitespace
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got, err := parseDate(tc.input)
			if err != nil {
				t.Fatalf("parseDate(%q) error: %v", tc.input, err)
			}
			if got.Year() != tc.year || got.Month() != tc.month || got.Day() != tc.day {
				t.Errorf("got %v, want %d-%02d-%02d", got, tc.year, tc.month, tc.day)
			}
		})
	}
}

func TestParseDate_Invalid(t *testing.T) {
	bad := []string{"not-a-date", "2026", "99/99/9999", "", "Aspirin 500mg"}
	for _, v := range bad {
		if _, err := parseDate(v); err == nil {
			t.Errorf("parseDate(%q) should have failed", v)
		}
	}
}
