package phone

import "strings"

// Variants returns all possible formats of a phone number so that DB lookups
// match regardless of how the number was stored.
//
// Handles the main Mexico issue: WhatsApp sends "521XXXXXXXXXX" (13 digits)
// but users may store "52XXXXXXXXXX" (12), "XXXXXXXXXX" (10), or with "+".
func Variants(phone string) []string {
	p := strings.TrimPrefix(strings.ReplaceAll(phone, " ", ""), "+")
	if p == "" {
		return nil
	}

	seen := map[string]bool{p: true}
	add := func(s string) { seen[s] = true }

	// 521 + 10 digits = Mexican mobile with old prefix
	if strings.HasPrefix(p, "521") && len(p) == 13 {
		add("52" + p[3:])  // without the 1
		add(p[3:])         // just 10 digits
		add("+" + p)       // with +
	}

	// 52 + 10 digits = Mexican without old prefix
	if strings.HasPrefix(p, "52") && len(p) == 12 {
		add("521" + p[2:]) // with the 1
		add(p[2:])         // just 10 digits
		add("+" + p)       // with +
	}

	// 10 digits = local Mexican number
	if len(p) == 10 {
		add("52" + p)
		add("521" + p)
	}

	// 11 digits starting with 1 = Mexican with old prefix, no country code
	if strings.HasPrefix(p, "1") && len(p) == 11 {
		add("52" + p)   // 52 + 1 + 10 = 13
		add("52" + p[1:]) // 52 + 10 = 12
		add(p[1:])        // just 10
	}

	result := make([]string, 0, len(seen))
	for v := range seen {
		result = append(result, v)
	}
	return result
}
