package ai

import "testing"

func confidentInvoice() *InvoiceResult {
	return &InvoiceResult{
		Total:         1234.50,
		Currency:      "MXN",
		BusinessName:  "Café Central S.A. de C.V.",
		BusinessRFC:   "GODE561231GR8",
		InvoiceNumber: "A-000123",
		Date:          "2026-06-25",
		Confident:     true,
	}
}

func TestComputeFingerprint_StableForSameTicket(t *testing.T) {
	a, err := ComputeFingerprint(confidentInvoice())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, err := ComputeFingerprint(confidentInvoice())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Hash == "" {
		t.Fatal("expected a non-empty hash for a confident invoice with folio")
	}
	if a.Hash != b.Hash {
		t.Fatalf("same ticket produced different hashes: %s != %s", a.Hash, b.Hash)
	}
	// RFC present → business_rfc must be the field used, not business_name.
	if got := a.HashFields; len(got) != 4 || got[0] != "business_rfc" ||
		got[1] != "invoice_number" || got[2] != "date" || got[3] != "total" {
		t.Fatalf("unexpected hash fields: %v", got)
	}
	if !a.Confident {
		t.Fatal("expected Confident to mirror the extract flag")
	}
	if len(a.Data) == 0 {
		t.Fatal("expected receipt data JSON to be populated")
	}
}

func TestComputeFingerprint_NormalizesOCRVariance(t *testing.T) {
	base := confidentInvoice()

	// A second scan of the same physical ticket with OCR noise: extra spaces,
	// different casing, RFC with separators, total with a trailing zero, and a
	// date in a different but equivalent representation.
	noisy := &InvoiceResult{
		Total:         1234.5, // 1234.5 == 1234.50 after rounding
		Currency:      "mxn",
		BusinessName:  "  café central s.a. DE c.v. ",
		BusinessRFC:   " GODE 561231 GR8 ",
		InvoiceNumber: "A-000123",
		Date:          "2026-06-25T10:30:00Z",
		Confident:     true,
	}

	a, _ := ComputeFingerprint(base)
	b, _ := ComputeFingerprint(noisy)
	if a.Hash != b.Hash {
		t.Fatalf("OCR variants of the same ticket hashed differently:\n base=%s\n noisy=%s", a.Hash, b.Hash)
	}
}

func TestComputeFingerprint_DifferentTicketsDiffer(t *testing.T) {
	a, _ := ComputeFingerprint(confidentInvoice())

	other := confidentInvoice()
	other.InvoiceNumber = "A-000999" // different folio → different ticket
	b, _ := ComputeFingerprint(other)

	if a.Hash == b.Hash {
		t.Fatal("different folios must produce different hashes")
	}

	other2 := confidentInvoice()
	other2.Total = 1234.51 // one cent apart → different ticket
	c, _ := ComputeFingerprint(other2)
	if a.Hash == c.Hash {
		t.Fatal("different totals must produce different hashes")
	}
}

func TestComputeFingerprint_FallsBackToBusinessNameWithoutRFC(t *testing.T) {
	inv := confidentInvoice()
	inv.BusinessRFC = ""
	fp, _ := ComputeFingerprint(inv)
	if fp.Hash == "" {
		t.Fatal("expected a hash using business_name fallback")
	}
	if fp.HashFields[0] != "business_name" {
		t.Fatalf("expected business_name fallback, got %v", fp.HashFields)
	}
}

func TestComputeFingerprint_NoHashWhenNotConfident(t *testing.T) {
	inv := confidentInvoice()
	inv.Confident = false
	fp, err := ComputeFingerprint(inv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fp.Hash != "" || len(fp.HashFields) != 0 {
		t.Fatalf("expected no hash for a non-confident extract, got hash=%q fields=%v", fp.Hash, fp.HashFields)
	}
	// Data is still persisted for auditing.
	if len(fp.Data) == 0 {
		t.Fatal("expected data to be persisted even without a hash")
	}
}

func TestComputeFingerprint_NoHashWhenMissingFolio(t *testing.T) {
	inv := confidentInvoice()
	inv.InvoiceNumber = "   " // whitespace only
	fp, _ := ComputeFingerprint(inv)
	if fp.Hash != "" {
		t.Fatalf("expected no hash when folio is missing, got %q", fp.Hash)
	}
}

// A failed / low-signal Gemini analysis produces a zero-value InvoiceResult.
// We must NOT serialize it: Data stays nil so receipt_data ends up NULL rather
// than a meaningless empty JSON blob. The transaction is still credited (no hash).
func TestComputeFingerprint_NoDataForZeroValueInvoice(t *testing.T) {
	fp, err := ComputeFingerprint(&InvoiceResult{}) // zero-value, !Confident
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fp.Data != nil {
		t.Fatalf("expected nil Data for a zero-value invoice, got %q", fp.Data)
	}
	if fp.Hash != "" || len(fp.HashFields) != 0 {
		t.Fatalf("expected no hash for a zero-value invoice, got hash=%q fields=%v", fp.Hash, fp.HashFields)
	}
}

// A non-confident extract that still carries real data (e.g. a total and a
// business name Gemini was unsure about) must persist for auditing.
func TestComputeFingerprint_KeepsDataWhenNotConfidentButHasContent(t *testing.T) {
	inv := &InvoiceResult{Total: 250, BusinessName: "Taquería El Fogón", Confident: false}
	fp, err := ComputeFingerprint(inv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fp.Data) == 0 {
		t.Fatal("expected data to be persisted for a non-confident extract with real content")
	}
	if fp.Hash != "" {
		t.Fatalf("expected no hash for a non-confident extract, got %q", fp.Hash)
	}
}

func TestComputeFingerprint_NilInvoice(t *testing.T) {
	fp, err := ComputeFingerprint(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fp.Hash != "" || fp.Data != nil {
		t.Fatal("expected empty fingerprint for nil invoice")
	}
}
