package main

import (
	"crypto/ed25519"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/license"
	"github.com/golang-jwt/jwt/v5"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "inspect":
		runInspect(os.Args[2:])
	case "verify":
		runVerify(os.Args[2:])
	case "help", "-h", "--help":
		printUsage()
	default:
		fatalf("unknown command %q", os.Args[1])
	}
}

func printUsage() {
	fmt.Print(`whatomate-license-admin - inspect and verify offline license tokens

Usage:
  whatomate-license-admin <command> [options]

Commands:
  inspect   Decode a token and optionally verify it
  verify    Verify a token against a public key or key ring

Examples:
  whatomate-license-admin inspect -token-file license.txt
  whatomate-license-admin inspect -token-file license.txt -public-key-file public.key -kid vendor-1
  whatomate-license-admin verify -token-file license.txt -public-key-file public.key -kid vendor-1
`)
}

func runInspect(args []string) {
	fs := flag.NewFlagSet("inspect", flag.ExitOnError)
	tokenFlag := fs.String("token", "", "Inline token value")
	tokenFile := fs.String("token-file", "", "Path to a token file")
	publicKeyFile := fs.String("public-key-file", "", "Optional base64 public key file for verification")
	kid := fs.String("kid", "", "Key ID for -public-key-file")
	keyRingFile := fs.String("key-ring-file", "", "Optional JSON key ring file")
	nowFlag := fs.String("now", "", "Optional RFC3339 time used for verification")
	hwid := fs.String("hwid", "", "Optional expected HWID hash")
	_ = fs.Parse(args)

	rawToken := mustReadToken(*tokenFlag, *tokenFile)
	claims, header, err := parseTokenUnverified(rawToken)
	if err != nil {
		fatalf("inspect token: %v", err)
	}

	result := map[string]any{
		"header": header,
		"claims": claims,
	}

	if *publicKeyFile != "" || *keyRingFile != "" {
		keyRing, loadErr := loadKeyRing(*publicKeyFile, *kid, *keyRingFile)
		if loadErr != nil {
			fatalf("load keys: %v", loadErr)
		}
		now, parseErr := parseNow(*nowFlag)
		if parseErr != nil {
			fatalf("parse now: %v", parseErr)
		}
		verifiedClaims, verifiedKID, verifyErr := license.VerifyToken(rawToken, keyRing, now)
		result["verified"] = verifyErr == nil
		if verifyErr != nil {
			result["verify_error"] = verifyErr.Error()
		} else {
			result["verified_kid"] = verifiedKID
			if expected := strings.TrimSpace(*hwid); expected != "" {
				result["hwid_match"] = verifiedClaims.HWIDHash == expected
			}
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		fatalf("encode output: %v", err)
	}
}

func runVerify(args []string) {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	tokenFlag := fs.String("token", "", "Inline token value")
	tokenFile := fs.String("token-file", "", "Path to a token file")
	publicKeyFile := fs.String("public-key-file", "", "Base64 public key file")
	kid := fs.String("kid", "", "Key ID for -public-key-file")
	keyRingFile := fs.String("key-ring-file", "", "JSON key ring file")
	nowFlag := fs.String("now", "", "Optional RFC3339 verification time")
	hwid := fs.String("hwid", "", "Optional expected HWID hash")
	_ = fs.Parse(args)

	rawToken := mustReadToken(*tokenFlag, *tokenFile)
	keyRing, err := loadKeyRing(*publicKeyFile, *kid, *keyRingFile)
	if err != nil {
		fatalf("load keys: %v", err)
	}
	now, err := parseNow(*nowFlag)
	if err != nil {
		fatalf("parse now: %v", err)
	}

	claims, verifiedKID, err := license.VerifyToken(rawToken, keyRing, now)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid: %v\n", err)
		os.Exit(1)
	}
	if expected := strings.TrimSpace(*hwid); expected != "" && claims.HWIDHash != expected {
		fmt.Fprintln(os.Stderr, "invalid: hwid mismatch")
		os.Exit(1)
	}

	result := map[string]any{
		"valid":  true,
		"kid":    verifiedKID,
		"expiry": claims.ExpiresAt,
		"tier":   claims.Tier,
		"kind":   claims.LicenseKind,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		fatalf("encode output: %v", err)
	}
}

func mustReadToken(tokenValue, tokenFile string) string {
	if trimmed := strings.TrimSpace(tokenValue); trimmed != "" {
		return trimmed
	}
	if strings.TrimSpace(tokenFile) == "" {
		fatalf("either -token or -token-file is required")
	}
	data, err := os.ReadFile(tokenFile)
	if err != nil {
		fatalf("read token file: %v", err)
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		fatalf("token is empty")
	}
	return trimmed
}

func loadKeyRing(publicKeyFile, kid, keyRingFile string) (map[string]ed25519.PublicKey, error) {
	if strings.TrimSpace(keyRingFile) != "" {
		data, err := os.ReadFile(keyRingFile)
		if err != nil {
			return nil, err
		}
		var entries []license.KeyRingEntry
		if err := json.Unmarshal(data, &entries); err != nil {
			return nil, err
		}
		return license.ParseKeyRing(entries)
	}

	if strings.TrimSpace(publicKeyFile) == "" {
		return nil, fmt.Errorf("either -key-ring-file or -public-key-file is required")
	}
	if strings.TrimSpace(kid) == "" {
		return nil, fmt.Errorf("-kid is required when using -public-key-file")
	}

	data, err := os.ReadFile(publicKeyFile)
	if err != nil {
		return nil, err
	}
	publicKey, err := license.DecodePublicKey(strings.TrimSpace(string(data)))
	if err != nil {
		return nil, err
	}
	return map[string]ed25519.PublicKey{
		strings.TrimSpace(kid): publicKey,
	}, nil
}

func parseNow(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Now().UTC(), nil
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}

func parseTokenUnverified(raw string) (*license.LicenseClaims, map[string]any, error) {
	parser := jwt.NewParser()
	claims := &license.LicenseClaims{}
	token, _, err := parser.ParseUnverified(raw, claims)
	if err != nil {
		return nil, nil, err
	}
	header := make(map[string]any, len(token.Header))
	for key, value := range token.Header {
		header[key] = value
	}
	return claims, header, nil
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
