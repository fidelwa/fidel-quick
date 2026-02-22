package deeplink

import (
	"fmt"
	"net/url"
)

// WhatsAppURL generates a wa.me deeplink that embeds the business context.
// When the user taps "Send", the webhook receives the prefilled text
// containing the customer UUID for business resolution.
func WhatsAppURL(displayPhone, customerID, businessName string) string {
	text := fmt.Sprintf("Hola! Quiero unirme a %s unirme:%s", businessName, customerID)
	return fmt.Sprintf("https://wa.me/%s?text=%s", displayPhone, url.QueryEscape(text))
}
