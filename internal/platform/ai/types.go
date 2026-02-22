package ai

// PhotoResult holds the extracted data from a ticket photo.
type PhotoResult struct {
	Amount    float64 // Extracted total amount
	Currency  string  // Detected currency (e.g., "MXN")
	Confident bool    // Whether the extraction is reliable
	RawText   string  // OCR text for debugging
}
