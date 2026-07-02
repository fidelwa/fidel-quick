package admin

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	googleJWKSURL = "https://www.googleapis.com/oauth2/v3/certs"
	googleIssuer1 = "accounts.google.com"
	googleIssuer2 = "https://accounts.google.com"
)

// GoogleProfile holds the user info extracted from a verified Google ID token.
// Email + Sub are always populated. Profile fields (FullName, AvatarURL, ...)
// are only populated when the client requested the `profile` scope; consumers
// must treat them as optional.
type GoogleProfile struct {
	Sub           string // stable Google user id; canonical PK
	Email         string
	EmailVerified bool
	FullName      string // claim "name"
	GivenName     string
	FamilyName    string
	Picture       string // URL to the user's profile picture
	Locale        string // BCP47 (e.g. "en", "es-MX")
	HostedDomain  string // Google Workspace tenant domain (claim "hd"), empty for personal accounts
}

// GoogleVerifier validates Google ID tokens locally using Google's published JWKS.
// It caches the keys in memory and refreshes them when an unknown kid appears
// (key rotation) or when the cache expires.
type GoogleVerifier interface {
	Verify(idToken string) (*GoogleProfile, error)
}

type googleClaims struct {
	jwt.RegisteredClaims
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
	HostedDomain  string `json:"hd"`
}

type jwksKey struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type jwksDoc struct {
	Keys []jwksKey `json:"keys"`
}

type googleVerifier struct {
	clientID string
	jwksURL  string
	httpc    *http.Client

	mu     sync.Mutex
	keys   map[string]*rsa.PublicKey
	expiry time.Time
}

// NewGoogleVerifier returns a verifier configured against Google's public JWKS.
// clientID must be the OAuth Web Client ID; if empty, every Verify call fails.
func NewGoogleVerifier(clientID string) GoogleVerifier {
	return &googleVerifier{
		clientID: clientID,
		jwksURL:  googleJWKSURL,
		httpc:    &http.Client{Timeout: 5 * time.Second},
		keys:     map[string]*rsa.PublicKey{},
	}
}

func (v *googleVerifier) Verify(idToken string) (*GoogleProfile, error) {
	if v.clientID == "" {
		return nil, fmt.Errorf("google login not configured")
	}

	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithIssuer(googleIssuer2),
		jwt.WithAudience(v.clientID),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(30*time.Second),
	)
	claims := &googleClaims{}
	token, err := parser.ParseWithClaims(idToken, claims, v.keyFunc)
	if err != nil {
		// jwt's WithIssuer only accepts a single value; Google emits both
		// "accounts.google.com" and "https://accounts.google.com". Retry with
		// the alternate issuer if the first attempt failed solely on issuer.
		parserAlt := jwt.NewParser(
			jwt.WithValidMethods([]string{"RS256"}),
			jwt.WithIssuer(googleIssuer1),
			jwt.WithAudience(v.clientID),
			jwt.WithExpirationRequired(),
			jwt.WithLeeway(30*time.Second),
		)
		claimsAlt := &googleClaims{}
		tokenAlt, errAlt := parserAlt.ParseWithClaims(idToken, claimsAlt, v.keyFunc)
		if errAlt != nil {
			return nil, fmt.Errorf("verify google token: %w", err)
		}
		token, claims = tokenAlt, claimsAlt
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid google token")
	}
	if !claims.EmailVerified {
		return nil, fmt.Errorf("google email not verified")
	}
	if claims.Subject == "" || claims.Email == "" {
		return nil, fmt.Errorf("google token missing sub or email")
	}
	return &GoogleProfile{
		Sub:           claims.Subject,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
		FullName:      claims.Name,
		GivenName:     claims.GivenName,
		FamilyName:    claims.FamilyName,
		Picture:       claims.Picture,
		Locale:        claims.Locale,
		HostedDomain:  claims.HostedDomain,
	}, nil
}

func (v *googleVerifier) keyFunc(token *jwt.Token) (interface{}, error) {
	kid, _ := token.Header["kid"].(string)
	if kid == "" {
		return nil, fmt.Errorf("token header missing kid")
	}

	if key := v.lookup(kid); key != nil {
		return key, nil
	}
	if err := v.refresh(); err != nil {
		return nil, err
	}
	if key := v.lookup(kid); key != nil {
		return key, nil
	}
	return nil, fmt.Errorf("kid %q not found in JWKS", kid)
}

func (v *googleVerifier) lookup(kid string) *rsa.PublicKey {
	v.mu.Lock()
	defer v.mu.Unlock()
	if time.Now().After(v.expiry) {
		return nil
	}
	return v.keys[kid]
}

func (v *googleVerifier) refresh() error {
	resp, err := v.httpc.Get(v.jwksURL)
	if err != nil {
		return fmt.Errorf("fetch JWKS: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch JWKS: status %d", resp.StatusCode)
	}

	var doc jwksDoc
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return fmt.Errorf("decode JWKS: %w", err)
	}

	keys := map[string]*rsa.PublicKey{}
	for _, k := range doc.Keys {
		if k.Kty != "RSA" {
			continue
		}
		pk, err := rsaPublicKey(k.N, k.E)
		if err != nil {
			continue
		}
		keys[k.Kid] = pk
	}

	v.mu.Lock()
	defer v.mu.Unlock()
	v.keys = keys
	v.expiry = time.Now().Add(time.Hour)
	return nil
}

func rsaPublicKey(nB64, eB64 string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil {
		return nil, fmt.Errorf("decode n: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil {
		return nil, fmt.Errorf("decode e: %w", err)
	}
	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: int(new(big.Int).SetBytes(eBytes).Int64()),
	}, nil
}
