package licenseissuer

import (
	"crypto/ed25519"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/license"
)

const (
	DefaultKeyID           = "vendor-1"
	DefaultPrivateKeyFile  = "tmp/private.key"
	DefaultStudioBindAddr  = "127.0.0.1:41737"
	DefaultStudioDirName   = ".whatomate-license-studio"
	DefaultRegistryName    = "registry.json"
	DefaultKeyRingName     = "keyring.json"
	StatusValidTracked     = "valid_tracked"
	StatusValidUntracked   = "valid_untracked"
	StatusInvalid          = "invalid"
	RegistryStatusActive   = "active"
	RegistryStatusExpired  = "expired"
	RegistryStatusLifetime = "lifetime"
)

type IssueOptions struct {
	KeyID                   string
	PrivateKeyFile          string
	HWID                    string
	Duration                string
	Trial                   string
	Tier                    string
	LicenseID               string
	FamilyID                string
	Revision                uint64
	Organizations           int
	UsersPerOrg             int
	WhatsAppEndpointsPerOrg int
	Workers                 int
	IssuedAt                string
	NotBefore               string
}

type IssuedLicense struct {
	Token       string
	KeyID       string
	PublicKey   ed25519.PublicKey
	PrivateKey  ed25519.PrivateKey
	Claims      license.LicenseClaims
	Trial       string
	Duration    string
	IssuedAt    time.Time
	NotBefore   time.Time
	ExpiresAt   *time.Time
	LicenseKind string
}

func DefaultIssueOptions() IssueOptions {
	return IssueOptions{
		KeyID:                   DefaultKeyID,
		PrivateKeyFile:          DefaultPrivateKeyFile,
		Duration:                "365d",
		Tier:                    "starter",
		Revision:                1,
		Organizations:           1,
		UsersPerOrg:             5,
		WhatsAppEndpointsPerOrg: 5,
		Workers:                 2,
	}
}

func IssueTokenFromOptions(opts IssueOptions, now time.Time) (string, error) {
	result, err := IssueLicenseFromOptions(opts, now)
	if err != nil {
		return "", err
	}
	return result.Token, nil
}

func IssueLicenseFromOptions(opts IssueOptions, now time.Time) (IssuedLicense, error) {
	privateKeyFile := strings.TrimSpace(opts.PrivateKeyFile)
	if privateKeyFile == "" {
		return IssuedLicense{}, fmt.Errorf("private key file is required")
	}

	rawPrivateKey, err := os.ReadFile(privateKeyFile)
	if err != nil {
		return IssuedLicense{}, fmt.Errorf("read private key: %w", err)
	}

	return IssueLicenseFromPrivateKeyText(opts, string(rawPrivateKey), now)
}

func IssueLicenseFromPrivateKeyText(opts IssueOptions, rawPrivateKey string, now time.Time) (IssuedLicense, error) {
	privateKey, err := license.DecodePrivateKey(strings.TrimSpace(rawPrivateKey))
	if err != nil {
		return IssuedLicense{}, fmt.Errorf("decode private key: %w", err)
	}
	return issueLicense(opts, privateKey, now)
}

func issueLicense(opts IssueOptions, privateKey ed25519.PrivateKey, now time.Time) (IssuedLicense, error) {
	hwid := strings.TrimSpace(opts.HWID)
	if hwid == "" {
		return IssuedLicense{}, fmt.Errorf("hwid is required")
	}

	issuedAt, err := parseOptionalTime(opts.IssuedAt)
	if err != nil {
		return IssuedLicense{}, fmt.Errorf("parse issued-at: %w", err)
	}
	notBefore, err := parseOptionalTime(opts.NotBefore)
	if err != nil {
		return IssuedLicense{}, fmt.Errorf("parse not-before: %w", err)
	}
	if issuedAt.IsZero() {
		issuedAt = now.UTC()
	}
	if notBefore.IsZero() {
		notBefore = issuedAt
	}

	kind := license.KindPaid
	trialDays := 0
	expiresAt, err := parseLicenseExpiry(opts.Duration, opts.Trial, issuedAt)
	if err != nil {
		return IssuedLicense{}, err
	}

	tier := strings.TrimSpace(opts.Tier)
	if strings.TrimSpace(opts.Trial) != "" {
		kind = license.KindTrial
		trialDays, _ = trialPresetDays(opts.Trial)
		if tier == "" || tier == "starter" {
			tier = "trial"
		}
	}

	keyID := strings.TrimSpace(opts.KeyID)
	if keyID == "" {
		keyID = DefaultKeyID
	}

	token, err := license.IssueToken(license.IssueRequest{
		KeyID:                      keyID,
		PrivateKey:                 privateKey,
		LicenseID:                  strings.TrimSpace(opts.LicenseID),
		LicenseFamilyID:            strings.TrimSpace(opts.FamilyID),
		Revision:                   opts.Revision,
		HWIDHash:                   hwid,
		Tier:                       tier,
		LicenseKind:                kind,
		TrialDays:                  trialDays,
		MaxOrganizations:           opts.Organizations,
		MaxUsersPerOrg:             opts.UsersPerOrg,
		MaxWhatsAppEndpointsPerOrg: opts.WhatsAppEndpointsPerOrg,
		MaxWorkers:                 opts.Workers,
		IssuedAt:                   issuedAt,
		NotBefore:                  notBefore,
		ExpiresAt:                  expiresAt,
	})
	if err != nil {
		return IssuedLicense{}, err
	}

	result := IssuedLicense{
		Token:       token,
		KeyID:       keyID,
		PublicKey:   privateKey.Public().(ed25519.PublicKey),
		PrivateKey:  privateKey,
		Trial:       normalizeDurationPreset(opts.Trial),
		Duration:    normalizeDurationPreset(opts.Duration),
		IssuedAt:    issuedAt,
		NotBefore:   notBefore,
		ExpiresAt:   expiresAt,
		LicenseKind: kind,
		Claims: license.LicenseClaims{
			LicenseID:                  strings.TrimSpace(opts.LicenseID),
			LicenseFamilyID:            strings.TrimSpace(opts.FamilyID),
			Revision:                   opts.Revision,
			Product:                    license.ProductWhatomate,
			HWIDHash:                   hwid,
			Tier:                       tier,
			LicenseKind:                kind,
			TrialDays:                  trialDays,
			MaxOrganizations:           opts.Organizations,
			MaxUsersPerOrg:             opts.UsersPerOrg,
			MaxWhatsAppEndpointsPerOrg: opts.WhatsAppEndpointsPerOrg,
			MaxWorkers:                 opts.Workers,
		},
	}

	verifiedClaims, _, err := license.VerifyToken(token, map[string]ed25519.PublicKey{keyID: result.PublicKey}, issuedAt)
	if err != nil {
		return IssuedLicense{}, fmt.Errorf("verify issued token: %w", err)
	}
	result.Claims = *verifiedClaims

	return result, nil
}

func normalizeDurationPreset(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return ""
	}
	if trimmed == "lifetime" {
		return trimmed
	}
	if days, ok := parseDurationDays(trimmed); ok {
		return fmt.Sprintf("%dd", days)
	}
	return trimmed
}

func parseLicenseExpiry(durationPreset, trialPreset string, issuedAt time.Time) (*time.Time, error) {
	if strings.TrimSpace(trialPreset) != "" {
		days, ok := trialPresetDays(trialPreset)
		if !ok {
			return nil, fmt.Errorf("trial must be 7d or 14d")
		}
		expiresAt := issuedAt.AddDate(0, 0, days)
		return &expiresAt, nil
	}

	normalized := normalizeDurationPreset(durationPreset)
	if normalized == "lifetime" {
		return nil, nil
	}
	if days, ok := parseDurationDays(normalized); ok {
		expiresAt := issuedAt.AddDate(0, 0, days)
		return &expiresAt, nil
	}
	return nil, fmt.Errorf("duration must be a positive number of days like 55d or lifetime")
}

func trialPresetDays(value string) (int, bool) {
	switch normalizeDurationPreset(value) {
	case "7d":
		return 7, true
	case "14d":
		return 14, true
	default:
		return 0, false
	}
}

func parseOptionalTime(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}

func parseDurationDays(value string) (int, bool) {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return 0, false
	}
	trimmed = strings.TrimSuffix(trimmed, "days")
	trimmed = strings.TrimSuffix(trimmed, "day")
	trimmed = strings.TrimSuffix(trimmed, "d")
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		return 0, false
	}
	days, err := strconv.Atoi(trimmed)
	if err != nil || days <= 0 {
		return 0, false
	}
	return days, true
}
