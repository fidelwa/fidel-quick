package deeplink

import (
	"fmt"
	"net/url"
)

// WhatsAppURL genera un wa.me deeplink con el texto pre-rellenado para
// el usuario. El resolver del backend identifica al negocio a partir del
// nombre que aparece tras "unirme a " — más legible que un UUID/slug.
//
// El parámetro customerID se mantiene en la firma para no romper callers
// existentes pero ya no se usa (back-compat: el resolver acepta también
// el formato legacy "unirme:{uuid}" si algún QR viejo lo lleva).
func WhatsAppURL(displayPhone, _customerID, businessName string) string {
	text := fmt.Sprintf("Hola! Quiero unirme a %s", businessName)
	return fmt.Sprintf("https://wa.me/%s?text=%s", displayPhone, url.QueryEscape(text))
}
