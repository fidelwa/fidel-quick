package resolver

import (
	"context"
	"strings"

	"github.com/theluisbolivar/fidel-quick/internal/session"
)

// BusinessResult represents the outcome of business resolution.
type BusinessResult struct {
	CustomerID   string
	BusinessName string
	IsNew        bool // true if user is not registered in this business
}

// MultiBusinessResult is returned when a user belongs to multiple businesses.
type MultiBusinessResult struct {
	Options []session.SelectionOption
}

type BusinessResolver struct {
	repo Repository
}

func NewBusinessResolver(repo Repository) *BusinessResolver {
	return &BusinessResolver{repo: repo}
}

// Resolve determines which business the message belongs to.
// Returns:
//   - (*BusinessResult, nil, nil) — single business found
//   - (nil, *MultiBusinessResult, nil) — multiple businesses, user must select
//   - (nil, nil, nil) — user not registered anywhere
//   - (nil, nil, error) — internal error
func (r *BusinessResolver) Resolve(ctx context.Context, phone, messageText string) (*BusinessResult, *MultiBusinessResult, error) {
	// 1. Deeplink legacy: "unirme:{customer_uuid}". Mantenemos compat con
	// QRs antiguos que aún tengan el UUID en el texto.
	if customerID, ok := extractDeeplinkUUID(messageText); ok {
		return r.resolveFromDeeplinkID(ctx, phone, customerID)
	}

	// 2. Deeplink nuevo: "unirme a <NombreNegocio>". El wizard genera el
	// QR con este formato — mucho más legible para el usuario que ve el
	// mensaje pre-rellenado en WhatsApp.
	if name, ok := extractDeeplinkName(messageText); ok {
		if result, multi, err := r.resolveFromName(ctx, phone, name); err != nil || result != nil || multi != nil {
			return result, multi, err
		}
		// si no matchea ningún negocio por nombre, fallback a phone
	}

	// 3. Lookup phone in collaborators and clients
	return r.resolveFromPhone(ctx, phone)
}

func (r *BusinessResolver) resolveFromDeeplinkID(ctx context.Context, phone, customerID string) (*BusinessResult, *MultiBusinessResult, error) {
	name, err := r.repo.GetActiveCustomerByID(ctx, customerID)
	if err != nil {
		return nil, nil, err
	}
	if name == "" {
		return nil, nil, nil
	}

	exists, err := r.repo.UserExistsInBusiness(ctx, phone, customerID)
	if err != nil {
		return nil, nil, err
	}

	return &BusinessResult{CustomerID: customerID, BusinessName: name, IsNew: !exists}, nil, nil
}

func (r *BusinessResolver) resolveFromName(ctx context.Context, phone, name string) (*BusinessResult, *MultiBusinessResult, error) {
	matches, err := r.repo.FindActiveCustomersByName(ctx, name)
	if err != nil {
		return nil, nil, err
	}
	switch len(matches) {
	case 0:
		return nil, nil, nil
	case 1:
		exists, err := r.repo.UserExistsInBusiness(ctx, phone, matches[0].CustomerID)
		if err != nil {
			return nil, nil, err
		}
		return &BusinessResult{
			CustomerID:   matches[0].CustomerID,
			BusinessName: matches[0].Name,
			IsNew:        !exists,
		}, nil, nil
	default:
		// Múltiples negocios con el mismo nombre — el bot le va a pedir
		// al usuario que elija. El IsNew se resolverá una vez que pique.
		return nil, &MultiBusinessResult{Options: matches}, nil
	}
}

func (r *BusinessResolver) resolveFromPhone(ctx context.Context, phone string) (*BusinessResult, *MultiBusinessResult, error) {
	options, err := r.repo.FindBusinessesByPhone(ctx, phone)
	if err != nil {
		return nil, nil, err
	}

	switch len(options) {
	case 0:
		return nil, nil, nil
	case 1:
		return &BusinessResult{
			CustomerID:   options[0].CustomerID,
			BusinessName: options[0].Name,
		}, nil, nil
	default:
		return nil, &MultiBusinessResult{Options: options}, nil
	}
}

// extractDeeplinkUUID busca el token legacy "unirme:{uuid}" en el mensaje.
// Devuelve el UUID si lo encuentra. Solo se usa para back-compat con QRs
// generados antes de la migración a deeplink-por-nombre.
func extractDeeplinkUUID(text string) (string, bool) {
	idx := strings.Index(text, "unirme:")
	if idx == -1 {
		return "", false
	}
	rest := strings.TrimSpace(text[idx+len("unirme:"):])
	if spaceIdx := strings.IndexByte(rest, ' '); spaceIdx > 0 {
		rest = rest[:spaceIdx]
	}
	if len(rest) > 0 {
		return rest, true
	}
	return "", false
}

// extractDeeplinkName extrae el nombre del negocio desde un mensaje con
// formato "Quiero unirme a {NombreNegocio}". Insensible a case del prefix,
// trim de signos de puntuación finales. Si encuentra " unirme:" después
// del nombre (mezcla legacy + nuevo), corta ahí.
//
// Ejemplos:
//
//	"Hola! Quiero unirme a Santas Conchas"        → "Santas Conchas"
//	"hola, quiero unirme a Café del Sol."         → "Café del Sol"
//	"Quiero unirme a Foo unirme:abc"              → "Foo"  (back-compat híbrido)
func extractDeeplinkName(text string) (string, bool) {
	const marker = "unirme a "
	lower := strings.ToLower(text)
	idx := strings.Index(lower, marker)
	if idx == -1 {
		return "", false
	}
	rest := text[idx+len(marker):]
	// Si todavía contiene el token legacy " unirme:", cortar ahí.
	if i := strings.Index(rest, " unirme:"); i >= 0 {
		rest = rest[:i]
	}
	// Trim puntuación final típica de WhatsApp ("!", ".", "?").
	rest = strings.TrimRight(rest, " \t\n\r!.?,;:")
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return "", false
	}
	return rest, true
}
