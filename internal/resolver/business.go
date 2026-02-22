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
	// 1. Check for deeplink: "unirme:{customer_uuid}"
	if customerID, ok := extractDeeplink(messageText); ok {
		return r.resolveFromDeeplink(ctx, phone, customerID)
	}

	// 2. Lookup phone in collaborators and clients
	return r.resolveFromPhone(ctx, phone)
}

func (r *BusinessResolver) resolveFromDeeplink(ctx context.Context, phone, customerID string) (*BusinessResult, *MultiBusinessResult, error) {
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

// extractDeeplink finds "unirme:{uuid}" anywhere in message text.
func extractDeeplink(text string) (string, bool) {
	idx := strings.Index(text, "unirme:")
	if idx == -1 {
		return "", false
	}
	rest := strings.TrimSpace(text[idx+len("unirme:"):])
	// Take the first word (the UUID)
	if spaceIdx := strings.IndexByte(rest, ' '); spaceIdx > 0 {
		rest = rest[:spaceIdx]
	}
	if len(rest) > 0 {
		return rest, true
	}
	return "", false
}
