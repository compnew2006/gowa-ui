package license

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	TokenType        = "WHM-LICENSE"
	TokenIssuer      = "whatomate-license-vendor"
	TokenAudience    = "whatomate-server"
	ProductWhatomate = "whatomate"
	KindTrial        = "trial"
	KindPaid         = "paid"
)

// EmbeddedPublicKeyRingJSON is the legacy build-time JSON injection hook.
var EmbeddedPublicKeyRingJSON = "[]"

// EmbeddedPublicKeyRingBase64 is the preferred build-time key-ring payload.
var EmbeddedPublicKeyRingBase64 = ""

type KeyRingEntry struct {
	KID       string `json:"kid"`
	PublicKey string `json:"public_key"`
}

// LicenseClaims is the canonical signed payload for offline licenses.
type LicenseClaims struct {
	LicenseID                  string `json:"license_id"`
	LicenseFamilyID            string `json:"license_family_id"`
	Revision                   uint64 `json:"revision"`
	Product                    string `json:"product"`
	HWIDHash                   string `json:"hwid_hash"`
	Tier                       string `json:"tier"`
	LicenseKind                string `json:"license_kind"`
	TrialDays                  int    `json:"trial_days"`
	MaxOrganizations           int    `json:"max_organizations"`
	MaxUsersPerOrg             int    `json:"max_users_per_org"`
	MaxWhatsAppEndpointsPerOrg int    `json:"max_whatsapp_endpoints_per_org"`
	MaxWorkers                 int    `json:"max_workers"`
	jwt.RegisteredClaims
}

// Validate performs custom semantic validation beyond JWT signature and time checks.
func (c *LicenseClaims) Validate() error {
	if strings.TrimSpace(c.Product) != ProductWhatomate {
		return fmt.Errorf("unexpected product")
	}
	if strings.TrimSpace(c.LicenseID) == "" {
		return fmt.Errorf("license_id is required")
	}
	if strings.TrimSpace(c.LicenseFamilyID) == "" {
		return fmt.Errorf("license_family_id is required")
	}
	if strings.TrimSpace(c.HWIDHash) == "" {
		return fmt.Errorf("hwid_hash is required")
	}
	if strings.TrimSpace(c.Tier) == "" {
		return fmt.Errorf("tier is required")
	}
	if c.LicenseKind != KindTrial && c.LicenseKind != KindPaid {
		return fmt.Errorf("license_kind must be trial or paid")
	}
	if c.LicenseKind == KindTrial {
		if c.TrialDays != 7 && c.TrialDays != 14 {
			return fmt.Errorf("trial_days must be 7 or 14 for trial licenses")
		}
	} else if c.TrialDays != 0 {
		return fmt.Errorf("trial_days must be 0 for paid licenses")
	}
	if c.MaxOrganizations < 0 || c.MaxUsersPerOrg < 0 || c.MaxWhatsAppEndpointsPerOrg < 0 || c.MaxWorkers < 0 {
		return fmt.Errorf("license limits cannot be negative")
	}
	if c.Issuer != TokenIssuer {
		return fmt.Errorf("unexpected issuer")
	}
	hasAudience := false
	for _, audience := range c.Audience {
		if audience == TokenAudience {
			hasAudience = true
			break
		}
	}
	if !hasAudience {
		return fmt.Errorf("unexpected audience")
	}
	return nil
}

type IssueRequest struct {
	KeyID                      string
	PrivateKey                 ed25519.PrivateKey
	LicenseID                  string
	LicenseFamilyID            string
	Revision                   uint64
	HWIDHash                   string
	Tier                       string
	LicenseKind                string
	TrialDays                  int
	MaxOrganizations           int
	MaxUsersPerOrg             int
	MaxWhatsAppEndpointsPerOrg int
	MaxWorkers                 int
	IssuedAt                   time.Time
	NotBefore                  time.Time
	ExpiresAt                  *time.Time
}

// GenerateKeyPair creates a new Ed25519 keypair and returns base64-encoded values.
func GenerateKeyPair() (publicKey, privateKey string, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", err
	}
	return base64.StdEncoding.EncodeToString(pub), base64.StdEncoding.EncodeToString(priv), nil
}

func DecodePublicKey(value string) (ed25519.PublicKey, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return nil, fmt.Errorf("decode public key: %w", err)
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("public key must be %d bytes", ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(raw), nil
}

func DecodePrivateKey(value string) (ed25519.PrivateKey, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return nil, fmt.Errorf("decode private key: %w", err)
	}
	if len(raw) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("private key must be %d bytes", ed25519.PrivateKeySize)
	}
	return ed25519.PrivateKey(raw), nil
}

func ParseKeyRing(entries []KeyRingEntry) (map[string]ed25519.PublicKey, error) {
	keyRing := make(map[string]ed25519.PublicKey, len(entries))
	for _, entry := range entries {
		kid := strings.TrimSpace(entry.KID)
		if kid == "" {
			return nil, fmt.Errorf("key entry missing kid")
		}
		key, err := DecodePublicKey(entry.PublicKey)
		if err != nil {
			return nil, fmt.Errorf("decode key %q: %w", kid, err)
		}
		keyRing[kid] = key
	}
	return keyRing, nil
}

func ParseEmbeddedKeyRing() (map[string]ed25519.PublicKey, error) {
	payload, err := embeddedKeyRingPayload()
	if err != nil {
		return nil, err
	}
	var entries []KeyRingEntry
	if err := json.Unmarshal(payload, &entries); err != nil {
		return nil, fmt.Errorf("parse embedded key ring: %w", err)
	}
	return ParseKeyRing(entries)
}

func embeddedKeyRingPayload() ([]byte, error) {
	if trimmed := strings.TrimSpace(EmbeddedPublicKeyRingBase64); trimmed != "" {
		decoded, err := base64.StdEncoding.DecodeString(trimmed)
		if err != nil {
			return nil, fmt.Errorf("decode embedded key ring base64: %w", err)
		}
		decoded = bytes.TrimSpace(decoded)
		if len(decoded) == 0 {
			return []byte("[]"), nil
		}
		return decoded, nil
	}

	trimmed := strings.TrimSpace(EmbeddedPublicKeyRingJSON)
	if trimmed == "" {
		trimmed = "[]"
	}
	return []byte(trimmed), nil
}

func IssueToken(req IssueRequest) (string, error) {
	if len(req.PrivateKey) != ed25519.PrivateKeySize {
		return "", fmt.Errorf("private key must be %d bytes", ed25519.PrivateKeySize)
	}
	if strings.TrimSpace(req.KeyID) == "" {
		return "", fmt.Errorf("kid is required")
	}
	issuedAt := req.IssuedAt.UTC()
	if issuedAt.IsZero() {
		issuedAt = time.Now().UTC()
	}
	notBefore := req.NotBefore.UTC()
	if notBefore.IsZero() {
		notBefore = issuedAt
	}

	licenseID := strings.TrimSpace(req.LicenseID)
	if licenseID == "" {
		licenseID = uuid.NewString()
	}
	licenseFamilyID := strings.TrimSpace(req.LicenseFamilyID)
	if licenseFamilyID == "" {
		licenseFamilyID = licenseID
	}

	claims := LicenseClaims{
		LicenseID:                  licenseID,
		LicenseFamilyID:            licenseFamilyID,
		Revision:                   req.Revision,
		Product:                    ProductWhatomate,
		HWIDHash:                   strings.TrimSpace(req.HWIDHash),
		Tier:                       strings.TrimSpace(req.Tier),
		LicenseKind:                strings.TrimSpace(req.LicenseKind),
		TrialDays:                  req.TrialDays,
		MaxOrganizations:           req.MaxOrganizations,
		MaxUsersPerOrg:             req.MaxUsersPerOrg,
		MaxWhatsAppEndpointsPerOrg: req.MaxWhatsAppEndpointsPerOrg,
		MaxWorkers:                 req.MaxWorkers,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    TokenIssuer,
			Subject:   ProductWhatomate,
			Audience:  jwt.ClaimStrings{TokenAudience},
			ExpiresAt: jwt.NewNumericDate(zeroTimeIfNil(req.ExpiresAt)),
			NotBefore: jwt.NewNumericDate(notBefore),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ID:        licenseID,
		},
	}
	if req.ExpiresAt == nil {
		claims.ExpiresAt = nil
	}
	if err := claims.Validate(); err != nil {
		return "", err
	}

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	token.Header["kid"] = strings.TrimSpace(req.KeyID)
	token.Header["typ"] = TokenType
	return token.SignedString(req.PrivateKey)
}

func zeroTimeIfNil(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return value.UTC()
}

// VerifyToken verifies a signed license token against a key ring.
func VerifyToken(raw string, keyRing map[string]ed25519.PublicKey, now time.Time) (*LicenseClaims, string, error) {
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}),
		jwt.WithIssuer(TokenIssuer),
		jwt.WithAudience(TokenAudience),
		jwt.WithTimeFunc(func() time.Time { return now.UTC() }),
	)

	return parseTokenWithParser(raw, keyRing, parser)
}

// VerifyTokenSignatureOnly verifies the token signature and semantic claims
// without enforcing exp/nbf time windows. It is used for persisted license
// records so the runtime can still apply grace-period policy after expiry.
func VerifyTokenSignatureOnly(raw string, keyRing map[string]ed25519.PublicKey) (*LicenseClaims, string, error) {
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}),
		jwt.WithoutClaimsValidation(),
	)

	return parseTokenWithParser(raw, keyRing, parser)
}

func parseTokenWithParser(raw string, keyRing map[string]ed25519.PublicKey, parser *jwt.Parser) (*LicenseClaims, string, error) {
	claims := &LicenseClaims{}
	token, err := parser.ParseWithClaims(raw, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodEdDSA.Alg() {
			return nil, fmt.Errorf("unexpected signing method %s", token.Method.Alg())
		}
		kid, _ := token.Header["kid"].(string)
		kid = strings.TrimSpace(kid)
		if kid == "" {
			return nil, fmt.Errorf("missing kid header")
		}
		if typ, _ := token.Header["typ"].(string); strings.TrimSpace(typ) != TokenType {
			return nil, fmt.Errorf("unexpected token type")
		}
		key, ok := keyRing[kid]
		if !ok {
			return nil, fmt.Errorf("unknown key id")
		}
		return key, nil
	})
	if err != nil {
		return nil, "", err
	}
	if !token.Valid {
		return nil, "", fmt.Errorf("invalid token")
	}
	if err := claims.Validate(); err != nil {
		return nil, "", err
	}
	kid, _ := token.Header["kid"].(string)
	return claims, strings.TrimSpace(kid), nil
}
