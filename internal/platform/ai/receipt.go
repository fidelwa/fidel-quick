package ai

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ReceiptFingerprint is the anti-fraud fingerprint of an extracted invoice/receipt.
// It carries the full extracted payload (for auditing / dispute review) plus a
// canonical SHA-256 hash used to deduplicate the same physical ticket within a
// business. When the extract is not trustworthy enough to identify the ticket
// uniquely (see ComputeFingerprint), Hash is empty and the transaction is still
// credited but is NOT protected against duplicates.
type ReceiptFingerprint struct {
	// Data is the JSON serialization of the full InvoiceResult, destined for the
	// receipt_data JSONB column.
	Data []byte
	// Hash is the hex SHA-256 of the canonical subset, or "" when no reliable
	// hash could be computed.
	Hash string
	// HashFields lists the canonical field names that fed the hash, in order.
	// Empty when Hash is "".
	HashFields []string
	// Confident mirrors InvoiceResult.Confident (the extractor's own flag).
	Confident bool
}

var (
	// amountRe extracts the leading numeric part of a money-ish string so OCR
	// noise like "$1,234.50 MXN" normalizes to a plain number.
	nonAmountRe = regexp.MustCompile(`[^0-9.\-]`)
	spaceRe     = regexp.MustCompile(`\s+`)
)

// ComputeFingerprint builds the anti-fraud fingerprint for an extracted invoice.
//
// The canonical hash is SHA-256 over a normalized subset that identifies a
// physical ticket for a given business:
//
//	business_rfc | business_name (RFC preferred; name as fallback when RFC absent)
//	invoice_number
//	date
//	total
//
// Normalization removes OCR variance: strings are trimmed, lowercased and have
// internal whitespace collapsed; RFC is uppercased then lowercased consistently;
// amounts are parsed and rounded to 2 decimals; the date is coerced to ISO
// (YYYY-MM-DD) when parseable.
//
// The hash is only computed when the ticket can be reliably identified:
//   - the extract is Confident, AND
//   - InvoiceNumber is present (the folio is what makes two tickets from the same
//     business on the same day for the same total distinguishable).
//
// Otherwise Hash/HashFields are left empty (provisional policy — pending Pablo):
// the caller should persist the data and credit without dedup protection.
func ComputeFingerprint(inv *InvoiceResult) (ReceiptFingerprint, error) {
	fp := ReceiptFingerprint{}
	if inv == nil {
		return fp, nil
	}

	data, err := json.Marshal(inv)
	if err != nil {
		return fp, fmt.Errorf("marshal invoice: %w", err)
	}
	fp.Data = data
	fp.Confident = inv.Confident

	// Ticket no identificable de forma confiable → sin hash (se acredita igual).
	if !inv.Confident || strings.TrimSpace(inv.InvoiceNumber) == "" {
		return fp, nil
	}

	// Business identity: prefer RFC (canonical, unique per business); fall back
	// to the business name when no RFC was extracted.
	businessField := "business_name"
	businessVal := normalizeText(inv.BusinessName)
	if rfc := normalizeRFC(inv.BusinessRFC); rfc != "" {
		businessField = "business_rfc"
		businessVal = rfc
	}

	parts := []string{
		businessVal,
		normalizeText(inv.InvoiceNumber),
		normalizeDate(inv.Date),
		normalizeAmount(inv.Total),
	}
	fp.HashFields = []string{businessField, "invoice_number", "date", "total"}

	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	fp.Hash = hex.EncodeToString(sum[:])
	return fp, nil
}

// normalizeText trims, lowercases and collapses internal whitespace.
func normalizeText(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	return spaceRe.ReplaceAllString(s, " ")
}

// normalizeRFC strips whitespace and non-alphanumeric noise, then lowercases so
// "GODE 561231-GR8 " and "gode561231gr8" hash identically.
func normalizeRFC(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// normalizeAmount parses a monetary float and renders it rounded to 2 decimals,
// so 1234.5 and 1234.50 and OCR "1,234.50" all collapse to "1234.50".
func normalizeAmount(v float64) string {
	return fmt.Sprintf("%.2f", v)
}

// normalizeDate coerces common date shapes to ISO (YYYY-MM-DD). If the input is
// not parseable it falls back to normalized text so distinct raw dates still
// differ in the hash.
func normalizeDate(s string) string {
	raw := strings.TrimSpace(s)
	if raw == "" {
		return ""
	}
	layouts := []string{
		"2006-01-02",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"02/01/2006",
		"01/02/2006",
		"02-01-2006",
		"2006/01/02",
		time.RFC3339,
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, raw); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return normalizeText(raw)
}
