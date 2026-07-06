package licenseissuer

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/compnew2006/whatomate/internal/license"
)

type RegistryEntry struct {
	ID                         string     `json:"id"`
	HWID                       string     `json:"hwid"`
	Token                      string     `json:"token"`
	LicenseID                  string     `json:"license_id"`
	LicenseFamilyID            string     `json:"license_family_id"`
	Revision                   uint64     `json:"revision"`
	KeyID                      string     `json:"kid"`
	Tier                       string     `json:"tier"`
	LicenseKind                string     `json:"license_kind"`
	TrialDays                  int        `json:"trial_days"`
	DurationPreset             string     `json:"duration_preset"`
	MaxOrganizations           int        `json:"orgs"`
	MaxUsersPerOrg             int        `json:"users"`
	MaxWhatsAppEndpointsPerOrg int        `json:"wa_endpoints"`
	MaxWorkers                 int        `json:"workers"`
	MaxWorkersPerOrg           int        `json:"workers_per_org"`
	MaxStorageBytesPerOrg      int64      `json:"storage_bytes_per_org"`
	IssuedAt                   time.Time  `json:"issued_at"`
	NotBefore                  time.Time  `json:"not_before"`
	ExpiresAt                  *time.Time `json:"expires_at,omitempty"`
	CreatedAt                  time.Time  `json:"created_at"`
}

type RegistrySummary struct {
	Total    int `json:"total"`
	Trials   int `json:"trials"`
	Paid     int `json:"paid"`
	Active   int `json:"active"`
	Expired  int `json:"expired"`
	Lifetime int `json:"lifetime"`
}

type RegistryFilter struct {
	HWID   string
	Tier   string
	Kind   string
	Status string
}

type RegistryStore struct {
	path string
	mu   sync.Mutex
	now  func() time.Time
}

type KeyRingStore struct {
	path string
	mu   sync.Mutex
}

type VerifyResult struct {
	Status        string          `json:"status"`
	Message       string          `json:"message"`
	Tracked       bool            `json:"tracked"`
	KeyID         string          `json:"kid,omitempty"`
	Claims        *VerifiedClaims `json:"claims,omitempty"`
	RegistryEntry *RegistryEntry  `json:"registry_entry,omitempty"`
	Error         string          `json:"error,omitempty"`
	KnownKeyIDs   []string        `json:"known_key_ids,omitempty"`
}

type VerifiedClaims struct {
	ID                         string     `json:"id"`
	HWID                       string     `json:"hwid"`
	LicenseID                  string     `json:"license_id"`
	LicenseFamilyID            string     `json:"license_family_id"`
	Revision                   uint64     `json:"revision"`
	KeyID                      string     `json:"kid"`
	Tier                       string     `json:"tier"`
	LicenseKind                string     `json:"license_kind"`
	TrialDays                  int        `json:"trial_days"`
	DurationPreset             string     `json:"duration_preset"`
	MaxOrganizations           int        `json:"orgs"`
	MaxUsersPerOrg             int        `json:"users"`
	MaxWhatsAppEndpointsPerOrg int        `json:"wa_endpoints"`
	MaxWorkers                 int        `json:"workers"`
	MaxWorkersPerOrg           int        `json:"workers_per_org"`
	MaxStorageBytesPerOrg      int64      `json:"storage_bytes_per_org"`
	IssuedAt                   time.Time  `json:"issued_at"`
	NotBefore                  time.Time  `json:"not_before"`
	ExpiresAt                  *time.Time `json:"expires_at,omitempty"`
}

func DefaultStudioDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, DefaultStudioDirName), nil
}

func DefaultRegistryPath() (string, error) {
	dir, err := DefaultStudioDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, DefaultRegistryName), nil
}

func DefaultKeyRingPath() (string, error) {
	dir, err := DefaultStudioDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, DefaultKeyRingName), nil
}

func NewRegistryStore(path string) *RegistryStore {
	return &RegistryStore{
		path: strings.TrimSpace(path),
		now:  time.Now,
	}
}

func NewKeyRingStore(path string) *KeyRingStore {
	return &KeyRingStore{path: strings.TrimSpace(path)}
}

func (s *RegistryStore) Save(entry RegistryEntry) (RegistryEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := s.loadLocked()
	if err != nil {
		return RegistryEntry{}, err
	}

	if entry.ID == "" {
		entry.ID = entry.LicenseID
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = s.now().UTC()
	}

	replaced := false
	for i := range entries {
		if entries[i].ID == entry.ID {
			entries[i] = entry
			replaced = true
			break
		}
	}
	if !replaced {
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CreatedAt.After(entries[j].CreatedAt)
	})

	if err := writePrivateJSONFile(s.path, entries); err != nil {
		return RegistryEntry{}, err
	}
	return entry, nil
}

func (s *RegistryStore) List(filter RegistryFilter) ([]RegistryEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := s.loadLocked()
	if err != nil {
		return nil, err
	}

	filtered := make([]RegistryEntry, 0, len(entries))
	for _, entry := range entries {
		if !matchesRegistryFilter(entry, filter, s.now()) {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered, nil
}

func (s *RegistryStore) Summary() (RegistrySummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := s.loadLocked()
	if err != nil {
		return RegistrySummary{}, err
	}

	now := s.now()
	var summary RegistrySummary
	summary.Total = len(entries)
	for _, entry := range entries {
		switch entry.LicenseKind {
		case license.KindTrial:
			summary.Trials++
		case license.KindPaid:
			summary.Paid++
		}

		switch entry.Status(now) {
		case RegistryStatusActive:
			summary.Active++
		case RegistryStatusExpired:
			summary.Expired++
		case RegistryStatusLifetime:
			summary.Lifetime++
		}
	}
	return summary, nil
}

func (s *RegistryStore) FindByID(id string) (*RegistryEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.ID == strings.TrimSpace(id) {
			copy := entry
			return &copy, nil
		}
	}
	return nil, os.ErrNotExist
}

func (s *RegistryStore) FindByToken(token string) (*RegistryEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(token)
	for _, entry := range entries {
		if strings.TrimSpace(entry.Token) == trimmed {
			copy := entry
			return &copy, nil
		}
	}
	return nil, os.ErrNotExist
}

func (s *RegistryStore) loadLocked() ([]RegistryEntry, error) {
	if strings.TrimSpace(s.path) == "" {
		return nil, fmt.Errorf("registry path is required")
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []RegistryEntry{}, nil
		}
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return []RegistryEntry{}, nil
	}
	var entries []RegistryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CreatedAt.After(entries[j].CreatedAt)
	})
	return entries, nil
}

func (e RegistryEntry) Status(now time.Time) string {
	if e.ExpiresAt == nil {
		return RegistryStatusLifetime
	}
	if now.UTC().After(e.ExpiresAt.UTC()) {
		return RegistryStatusExpired
	}
	return RegistryStatusActive
}

func BuildRegistryEntry(result IssuedLicense) RegistryEntry {
	durationPreset := result.Duration
	if result.LicenseKind == license.KindTrial {
		durationPreset = result.Trial
	}
	return RegistryEntry{
		ID:                         result.Claims.LicenseID,
		HWID:                       result.Claims.HWIDHash,
		Token:                      result.Token,
		LicenseID:                  result.Claims.LicenseID,
		LicenseFamilyID:            result.Claims.LicenseFamilyID,
		Revision:                   result.Claims.Revision,
		KeyID:                      result.KeyID,
		Tier:                       result.Claims.Tier,
		LicenseKind:                result.Claims.LicenseKind,
		TrialDays:                  result.Claims.TrialDays,
		DurationPreset:             durationPreset,
		MaxOrganizations:           result.Claims.MaxOrganizations,
		MaxUsersPerOrg:             result.Claims.MaxUsersPerOrg,
		MaxWhatsAppEndpointsPerOrg: result.Claims.MaxWhatsAppEndpointsPerOrg,
		MaxWorkers:                 result.Claims.MaxWorkers,
		MaxWorkersPerOrg:           result.Claims.MaxWorkersPerOrg,
		MaxStorageBytesPerOrg:      result.Claims.MaxStorageBytesPerOrg,
		IssuedAt:                   result.IssuedAt.UTC(),
		NotBefore:                  result.NotBefore.UTC(),
		ExpiresAt:                  result.ExpiresAt,
		CreatedAt:                  time.Now().UTC(),
	}
}

func (s *KeyRingStore) UpsertPublicKey(kid string, publicKey ed25519.PublicKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := s.loadLocked()
	if err != nil {
		return err
	}

	encoded := base64.StdEncoding.EncodeToString(publicKey)
	found := false
	for i := range entries {
		if strings.TrimSpace(entries[i].KID) == strings.TrimSpace(kid) {
			entries[i].PublicKey = encoded
			found = true
			break
		}
	}
	if !found {
		entries = append(entries, license.KeyRingEntry{
			KID:       strings.TrimSpace(kid),
			PublicKey: encoded,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].KID < entries[j].KID
	})
	return writePrivateJSONFile(s.path, entries)
}

func (s *KeyRingStore) KnownKeyIDs() ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	kids := make([]string, 0, len(entries))
	for _, entry := range entries {
		kids = append(kids, entry.KID)
	}
	sort.Strings(kids)
	return kids, nil
}

func (s *KeyRingStore) LoadMap() (map[string]ed25519.PublicKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	return license.ParseKeyRing(entries)
}

func (s *KeyRingStore) loadLocked() ([]license.KeyRingEntry, error) {
	if strings.TrimSpace(s.path) == "" {
		return nil, fmt.Errorf("keyring path is required")
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []license.KeyRingEntry{}, nil
		}
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return []license.KeyRingEntry{}, nil
	}
	var entries []license.KeyRingEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func VerifyToken(rawToken string, keyRing *KeyRingStore, registry *RegistryStore, now time.Time) VerifyResult {
	knownIDs, _ := keyRing.KnownKeyIDs()
	keyMap, err := keyRing.LoadMap()
	if err != nil {
		return VerifyResult{
			Status:      StatusInvalid,
			Message:     "Verification failed",
			Error:       err.Error(),
			KnownKeyIDs: knownIDs,
		}
	}
	if len(keyMap) == 0 {
		return VerifyResult{
			Status:      StatusInvalid,
			Message:     "No public keys are registered locally",
			Error:       "empty key ring",
			KnownKeyIDs: knownIDs,
		}
	}

	claims, kid, err := license.VerifyToken(strings.TrimSpace(rawToken), keyMap, now.UTC())
	if err != nil {
		return VerifyResult{
			Status:      StatusInvalid,
			Message:     "Token is invalid",
			Error:       err.Error(),
			KnownKeyIDs: knownIDs,
		}
	}

	entry := verifiedClaimsFromClaims(*claims)
	entry.KeyID = kid
	entry.LicenseKind = claims.LicenseKind
	entry.TrialDays = claims.TrialDays

	trackedEntry, trackErr := registry.FindByToken(rawToken)
	if trackErr == nil {
		return VerifyResult{
			Status:        StatusValidTracked,
			Message:       "Token is valid and tracked in the local registry",
			Tracked:       true,
			KeyID:         kid,
			Claims:        &entry,
			RegistryEntry: trackedEntry,
			KnownKeyIDs:   knownIDs,
		}
	}

	if trackErr != nil && !os.IsNotExist(trackErr) {
		return VerifyResult{
			Status:      StatusInvalid,
			Message:     "Verification failed",
			Tracked:     false,
			KeyID:       kid,
			Claims:      &entry,
			Error:       trackErr.Error(),
			KnownKeyIDs: knownIDs,
		}
	}

	return VerifyResult{
		Status:      StatusValidUntracked,
		Message:     "Token is valid but not found in the local registry",
		Tracked:     false,
		KeyID:       kid,
		Claims:      &entry,
		KnownKeyIDs: knownIDs,
	}
}

func verifiedClaimsFromClaims(claims license.LicenseClaims) VerifiedClaims {
	var expiresAt *time.Time
	if claims.ExpiresAt != nil {
		value := claims.ExpiresAt.Time.UTC()
		expiresAt = &value
	}

	var issuedAt time.Time
	if claims.IssuedAt != nil {
		issuedAt = claims.IssuedAt.Time.UTC()
	}
	var notBefore time.Time
	if claims.NotBefore != nil {
		notBefore = claims.NotBefore.Time.UTC()
	}

	durationPreset := "lifetime"
	if claims.LicenseKind == license.KindTrial && claims.TrialDays > 0 {
		durationPreset = fmt.Sprintf("%dd", claims.TrialDays)
	} else if expiresAt != nil && !issuedAt.IsZero() {
		days := int(expiresAt.Sub(issuedAt).Hours() / 24)
		if days > 0 {
			durationPreset = fmt.Sprintf("%dd", days)
		}
	}

	return VerifiedClaims{
		ID:                         claims.LicenseID,
		HWID:                       claims.HWIDHash,
		LicenseID:                  claims.LicenseID,
		LicenseFamilyID:            claims.LicenseFamilyID,
		Revision:                   claims.Revision,
		Tier:                       claims.Tier,
		LicenseKind:                claims.LicenseKind,
		TrialDays:                  claims.TrialDays,
		DurationPreset:             durationPreset,
		MaxOrganizations:           claims.MaxOrganizations,
		MaxUsersPerOrg:             claims.MaxUsersPerOrg,
		MaxWhatsAppEndpointsPerOrg: claims.MaxWhatsAppEndpointsPerOrg,
		MaxWorkers:                 claims.MaxWorkers,
		MaxWorkersPerOrg:           claims.MaxWorkersPerOrg,
		MaxStorageBytesPerOrg:      claims.MaxStorageBytesPerOrg,
		IssuedAt:                   issuedAt,
		NotBefore:                  notBefore,
		ExpiresAt:                  expiresAt,
	}
}

func matchesRegistryFilter(entry RegistryEntry, filter RegistryFilter, now time.Time) bool {
	if hwid := strings.TrimSpace(strings.ToLower(filter.HWID)); hwid != "" {
		if !strings.Contains(strings.ToLower(entry.HWID), hwid) {
			return false
		}
	}
	if tier := strings.TrimSpace(strings.ToLower(filter.Tier)); tier != "" {
		if strings.ToLower(entry.Tier) != tier {
			return false
		}
	}
	if kind := strings.TrimSpace(strings.ToLower(filter.Kind)); kind != "" {
		if strings.ToLower(entry.LicenseKind) != kind {
			return false
		}
	}
	if status := strings.TrimSpace(strings.ToLower(filter.Status)); status != "" {
		if strings.ToLower(entry.Status(now)) != status {
			return false
		}
	}
	return true
}

func writePrivateJSONFile(path string, value any) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("path is required")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tempName := tempFile.Name()
	defer func() { _ = os.Remove(tempName) }()

	if err := tempFile.Chmod(0o600); err != nil {
		_ = tempFile.Close()
		return err
	}
	if _, err := tempFile.Write(append(data, '\n')); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempName, path); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}
