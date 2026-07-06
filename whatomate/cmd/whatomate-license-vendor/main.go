package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/license"
	"github.com/compnew2006/whatomate/internal/licenseissuer"
)

const defaultKeyID = licenseissuer.DefaultKeyID

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "keygen":
		runKeygen(os.Args[2:])
	case "issue":
		runIssue(os.Args[2:])
	case "help", "-h", "--help":
		printUsage()
	default:
		fatalf("unknown command %q", os.Args[1])
	}
}

func printUsage() {
	fmt.Print(`whatomate-license-vendor - offline license issuer

Usage:
  whatomate-license-vendor <command> [options]

Commands:
  keygen   Generate a new Ed25519 keypair
  issue    Issue a signed offline license or trial token

Examples:
  whatomate-license-vendor keygen -kid vendor-1 -public-key-file public.key -private-key-file private.key
  whatomate-license-vendor issue -kid vendor-1 -private-key-file private.key -hwid <hash> -duration 55d -tier starter -orgs 1 -users 5 -wa-endpoints 5 -workers 2
  whatomate-license-vendor issue -kid vendor-1 -private-key-file private.key -hwid <hash> -trial 7d
`)
}

func runKeygen(args []string) {
	fs := flag.NewFlagSet("keygen", flag.ExitOnError)
	kid := fs.String("kid", defaultKeyID, "Key ID to associate with this public key")
	publicKeyFile := fs.String("public-key-file", "", "Optional path to write the base64 public key")
	privateKeyFile := fs.String("private-key-file", "", "Optional path to write the base64 private key")
	_ = fs.Parse(args)

	publicKey, privateKey, err := license.GenerateKeyPair()
	if err != nil {
		fatalf("generate keypair: %v", err)
	}

	if *publicKeyFile != "" {
		if err := os.WriteFile(*publicKeyFile, []byte(publicKey+"\n"), 0o644); err != nil {
			fatalf("write public key: %v", err)
		}
	}
	if *privateKeyFile != "" {
		if err := os.WriteFile(*privateKeyFile, []byte(privateKey+"\n"), 0o600); err != nil {
			fatalf("write private key: %v", err)
		}
	}

	output := map[string]any{
		"kid":         strings.TrimSpace(*kid),
		"public_key":  publicKey,
		"private_key": privateKey,
		"key_ring_json": []license.KeyRingEntry{
			{
				KID:       strings.TrimSpace(*kid),
				PublicKey: publicKey,
			},
		},
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		fatalf("encode output: %v", err)
	}
}

func runIssue(args []string) {
	defaults := licenseissuer.DefaultIssueOptions()
	fs := flag.NewFlagSet("issue", flag.ExitOnError)
	kid := fs.String("kid", defaultKeyID, "Key ID of the signing public key")
	privateKeyFile := fs.String("private-key-file", defaults.PrivateKeyFile, "Path to the base64 private key file")
	hwid := fs.String("hwid", "", "Target server HWID hash")
	duration := fs.String("duration", defaults.Duration, "Paid license duration: any positive day count like 55d, 365d, or lifetime")
	trial := fs.String("trial", "", "Trial duration: 7d or 14d")
	tier := fs.String("tier", defaults.Tier, "Tier label")
	licenseID := fs.String("license-id", "", "Optional explicit license_id")
	familyID := fs.String("family-id", "", "Optional explicit license_family_id")
	revision := fs.Uint64("revision", defaults.Revision, "License revision within the family")
	orgs := fs.Int("orgs", defaults.Organizations, "Maximum organizations")
	users := fs.Int("users", defaults.UsersPerOrg, "Maximum users per organization")
	endpoints := fs.Int("wa-endpoints", defaults.WhatsAppEndpointsPerOrg, "Maximum WhatsApp endpoints per organization")
	workers := fs.Int("workers", defaults.Workers, "Maximum workers")
	workersPerOrg := fs.Int("workers-per-org", defaults.WorkersPerOrg, "Maximum workers per organization (0 = unlimited)")
	storageBytesPerOrg := fs.Int64("storage-bytes", defaults.StorageBytesPerOrg, "Maximum stored bytes per organization (0 = unlimited)")
	issuedAtFlag := fs.String("issued-at", "", "Optional RFC3339 issued-at timestamp")
	notBeforeFlag := fs.String("not-before", "", "Optional RFC3339 not-before timestamp")
	_ = fs.Parse(args)

	token, err := licenseissuer.IssueTokenFromOptions(licenseissuer.IssueOptions{
		KeyID:                   *kid,
		PrivateKeyFile:          *privateKeyFile,
		HWID:                    *hwid,
		Duration:                *duration,
		Trial:                   *trial,
		Tier:                    *tier,
		LicenseID:               *licenseID,
		FamilyID:                *familyID,
		Revision:                *revision,
		Organizations:           *orgs,
		UsersPerOrg:             *users,
		WhatsAppEndpointsPerOrg: *endpoints,
		Workers:                 *workers,
		WorkersPerOrg:           *workersPerOrg,
		StorageBytesPerOrg:      *storageBytesPerOrg,
		IssuedAt:                *issuedAtFlag,
		NotBefore:               *notBeforeFlag,
	}, time.Now())
	if err != nil {
		fatalf("issue token: %v", err)
	}

	fmt.Println(token)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
