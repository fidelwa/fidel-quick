package ai

// InvoiceResult holds comprehensive data extracted from an invoice/receipt photo.
type InvoiceResult struct {
	// Core amounts
	Total    float64 `json:"total"`
	Subtotal float64 `json:"subtotal"`
	Tax      float64 `json:"tax"`
	Tip      float64 `json:"tip"`
	Currency string  `json:"currency"`

	// Business info
	BusinessName    string `json:"business_name"`
	BusinessRFC     string `json:"business_rfc"`
	BusinessAddress string `json:"business_address"`

	// Invoice metadata
	InvoiceNumber string `json:"invoice_number"`
	Date          string `json:"date"`
	PaymentMethod string `json:"payment_method"`

	// Line items
	Items []InvoiceItem `json:"items"`

	// Confidence and debug
	Confident   bool   `json:"confident"`
	RawResponse string `json:"-"`
}

// InvoiceItem represents a single line item from the invoice.
type InvoiceItem struct {
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	Total       float64 `json:"total"`
}
