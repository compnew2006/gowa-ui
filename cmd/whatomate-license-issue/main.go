package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/compnew2006/whatomate/internal/licenseissuer"
)

func main() {
	defaults := licenseissuer.DefaultIssueOptions()

	fs := flag.NewFlagSet("whatomate-license-issue", flag.ExitOnError)
	kid := fs.String("kid", defaults.KeyID, "Key ID of the signing public key")
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
	issuedAtFlag := fs.String("issued-at", "", "Optional RFC3339 issued-at timestamp")
	notBeforeFlag := fs.String("not-before", "", "Optional RFC3339 not-before timestamp")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), `whatomate-license-issue - dedicated offline license issuer

Usage:
  whatomate-license-issue -hwid <hash> [options]

Examples:
  whatomate-license-issue -hwid <hash> -trial 7d
  whatomate-license-issue -hwid <hash> -duration 55d -tier starter -orgs 1 -users 5 -wa-endpoints 5 -workers 2

Defaults:
  kid: %s
  private-key-file: %s
  orgs/users/wa-endpoints/workers: %d/%d/%d/%d

`, defaults.KeyID, defaults.PrivateKeyFile, defaults.Organizations, defaults.UsersPerOrg, defaults.WhatsAppEndpointsPerOrg, defaults.Workers)
		fs.PrintDefaults()
	}

	if err := fs.Parse(os.Args[1:]); err != nil {
		fatalf("%v", err)
	}

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
		IssuedAt:                *issuedAtFlag,
		NotBefore:               *notBeforeFlag,
	}, time.Now())
	if err != nil {
		fatalf("%v", err)
	}

	fmt.Println(token)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
