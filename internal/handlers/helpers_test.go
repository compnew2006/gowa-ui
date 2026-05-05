package handlers

import (
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// --- parsePathUUID ---

func TestParsePathUUID_Valid(t *testing.T) {
	t.Parallel()
	req := testutil.NewGETRequest(t)
	expected := uuid.New()
	testutil.SetPathParam(req, "id", expected.String())

	id, err := parsePathUUID(req, "id", "item")
	require.NoError(t, err)
	assert.Equal(t, expected, id)
}

func TestParsePathUUID_Invalid(t *testing.T) {
	t.Parallel()
	req := testutil.NewGETRequest(t)
	testutil.SetPathParam(req, "id", "not-a-uuid")

	id, err := parsePathUUID(req, "id", "item")
	assert.ErrorIs(t, err, errEnvelopeSent)
	assert.Equal(t, uuid.Nil, id)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

func TestParsePathUUID_Missing(t *testing.T) {
	t.Parallel()
	req := testutil.NewGETRequest(t)

	id, err := parsePathUUID(req, "id", "item")
	assert.ErrorIs(t, err, errEnvelopeSent)
	assert.Equal(t, uuid.Nil, id)
	assert.Equal(t, fasthttp.StatusBadRequest, testutil.GetResponseStatusCode(req))
}

// --- parsePagination ---

func TestParsePagination_Defaults(t *testing.T) {
	t.Parallel()
	req := testutil.NewGETRequest(t)

	p := parsePagination(req)
	assert.Equal(t, 1, p.Page)
	assert.Equal(t, 50, p.Limit)
	assert.Equal(t, 0, p.Offset)
}

func TestParsePagination_CustomValues(t *testing.T) {
	t.Parallel()
	req := testutil.NewGETRequest(t)
	testutil.SetQueryParam(req, "page", 3)
	testutil.SetQueryParam(req, "limit", 20)

	p := parsePagination(req)
	assert.Equal(t, 3, p.Page)
	assert.Equal(t, 20, p.Limit)
	assert.Equal(t, 40, p.Offset) // (3-1)*20
}

func TestParsePagination_MaxLimitCapping(t *testing.T) {
	t.Parallel()
	req := testutil.NewGETRequest(t)
	testutil.SetQueryParam(req, "limit", 500)

	p := parsePagination(req)
	assert.Equal(t, 50, p.Limit) // Exceeds max(100), falls back to default(50)
}

func TestParsePagination_ZeroPageDefaults(t *testing.T) {
	t.Parallel()
	req := testutil.NewGETRequest(t)
	testutil.SetQueryParam(req, "page", 0)
	testutil.SetQueryParam(req, "limit", 10)

	p := parsePagination(req)
	assert.Equal(t, 1, p.Page)
	assert.Equal(t, 10, p.Limit)
	assert.Equal(t, 0, p.Offset)
}

func TestParsePagination_NegativeValues(t *testing.T) {
	t.Parallel()
	req := testutil.NewGETRequest(t)
	testutil.SetQueryParam(req, "page", -1)
	testutil.SetQueryParam(req, "limit", -5)

	p := parsePagination(req)
	assert.Equal(t, 1, p.Page)
	assert.Equal(t, 50, p.Limit)
}

// --- parsePaginationWithDefaults ---

func TestParsePaginationWithDefaults_CustomDefaultAndMax(t *testing.T) {
	t.Parallel()
	req := testutil.NewGETRequest(t)

	p := parsePaginationWithDefaults(req, 25, 200)
	assert.Equal(t, 25, p.Limit)
}

func TestParsePaginationWithDefaults_LimitExceedsMax(t *testing.T) {
	t.Parallel()
	req := testutil.NewGETRequest(t)
	testutil.SetQueryParam(req, "limit", 300)

	p := parsePaginationWithDefaults(req, 25, 200)
	assert.Equal(t, 25, p.Limit)
}

// --- parseDateParam ---

func TestParseDateParam_Valid(t *testing.T) {
	t.Parallel()
	req := testutil.NewGETRequest(t)
	testutil.SetQueryParam(req, "start_date", "2024-06-15")

	result, ok := parseDateParam(req, "start_date")
	assert.True(t, ok)
	assert.Equal(t, 2024, result.Year())
	assert.Equal(t, time.June, result.Month())
	assert.Equal(t, 15, result.Day())
}

func TestParseDateParam_Invalid(t *testing.T) {
	t.Parallel()
	req := testutil.NewGETRequest(t)
	testutil.SetQueryParam(req, "start_date", "not-a-date")

	_, ok := parseDateParam(req, "start_date")
	assert.False(t, ok)
}

func TestParseDateParam_Missing(t *testing.T) {
	t.Parallel()
	req := testutil.NewGETRequest(t)

	_, ok := parseDateParam(req, "start_date")
	assert.False(t, ok)
}

func TestParseDateParam_WrongFormat(t *testing.T) {
	t.Parallel()
	req := testutil.NewGETRequest(t)
	testutil.SetQueryParam(req, "start_date", "15/06/2024")

	_, ok := parseDateParam(req, "start_date")
	assert.False(t, ok)
}

// --- endOfDay ---

func TestEndOfDay(t *testing.T) {
	t.Parallel()
	day := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	end := endOfDay(day)

	assert.Equal(t, 2024, end.Year())
	assert.Equal(t, time.June, end.Month())
	assert.Equal(t, 15, end.Day())
	assert.Equal(t, 23, end.Hour())
	assert.Equal(t, 59, end.Minute())
	assert.Equal(t, 59, end.Second())
}

// --- findByIDAndOrg ---

func TestFindByIDAndOrg_Found(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)

	org := &models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "find-test-" + uuid.New().String()[:8],
		Slug:      "find-test-" + uuid.New().String()[:8],
	}
	require.NoError(t, db.Create(org).Error)

	account := &models.WhatsAppAccount{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org.ID,
		Name:           "test-acct-" + uuid.New().String()[:8],
		PhoneID:        "p-" + uuid.New().String()[:8],
		BusinessID:     "b-" + uuid.New().String()[:8],
		AccessToken:    "tok",
	}
	require.NoError(t, db.Create(account).Error)

	req := testutil.NewGETRequest(t)
	result, err := findByIDAndOrg[models.WhatsAppAccount](db, req, account.ID, org.ID, "Account")
	require.NoError(t, err)
	assert.Equal(t, account.ID, result.ID)
}

func TestFindByIDAndOrg_NotFound(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)

	req := testutil.NewGETRequest(t)
	_, err := findByIDAndOrg[models.WhatsAppAccount](db, req, uuid.New(), uuid.New(), "Account")
	assert.ErrorIs(t, err, errEnvelopeSent)
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
}

func TestFindByIDAndOrg_CrossOrgIsolation(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)

	org1 := &models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "org1-" + uuid.New().String()[:8],
		Slug:      "org1-" + uuid.New().String()[:8],
	}
	org2 := &models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "org2-" + uuid.New().String()[:8],
		Slug:      "org2-" + uuid.New().String()[:8],
	}
	require.NoError(t, db.Create(org1).Error)
	require.NoError(t, db.Create(org2).Error)

	account := &models.WhatsAppAccount{
		BaseModel:      models.BaseModel{ID: uuid.New()},
		OrganizationID: org1.ID,
		Name:           "acct-" + uuid.New().String()[:8],
		PhoneID:        "p-" + uuid.New().String()[:8],
		BusinessID:     "b-" + uuid.New().String()[:8],
		AccessToken:    "tok",
	}
	require.NoError(t, db.Create(account).Error)

	// Try to access org1's account with org2's ID
	req := testutil.NewGETRequest(t)
	_, err := findByIDAndOrg[models.WhatsAppAccount](db, req, account.ID, org2.ID, "Account")
	assert.ErrorIs(t, err, errEnvelopeSent)
	assert.Equal(t, fasthttp.StatusNotFound, testutil.GetResponseStatusCode(req))
}

// --- MaskPhoneNumber ---

func TestMaskPhoneNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		phone string
		want  string
	}{
		{"standard phone", "+1234567890", "*******7890"},
		{"short number", "1234", "1234"},
		{"very short", "12", "12"},
		{"empty", "", ""},
		{"exactly 5 chars", "12345", "*2345"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, MaskPhoneNumber(tt.phone))
		})
	}
}

// --- MaskPhoneNumbersInText ---

func TestMaskPhoneNumbersInText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		text string
		want string
	}{
		{"standard international", "Call me at +1 234 567 8900 tomorrow", "Call me at ***********8900 tomorrow"},
		{"international with 00", "My number is 00447911123456 please", "My number is **********3456 please"},
		{"saudi format 05", "Number is 0561853319", "Number is ******3319"},
		{"saudi format without +", "Number is 966561853319", "Number is ********3319"},
		{"egyptian format 010", "Hit me on 01007181781 later", "Hit me on *******1781 later"},
		{"egyptian format arabic", "Hit me on ٠١٠٠٧١٨١٧٨١ later", "Hit me on *******١٧٨١ later"},
		{"egyptian format without +", "Hit me on 201007181781 later", "Hit me on ********1781 later"},
		{"multiple numbers", "Here: +44 7911 123456 and 01007181781", "Here: ***********3456 and *******1781"},
		{"not a phone number", "My order number is 123456789", "My order number is 123456789"},
		{"national id", "National ID is 10234567890123", "National ID is 10234567890123"},
		{"bank account", "Transfer to 3012345678901234", "Transfer to 3012345678901234"},
		{"short number with plus", "Wait +123 is too short", "Wait +123 is too short"},
		{"with dashes and dots", "Contact: +1-234.567-8900.", "Contact: ***********8900."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, MaskPhoneNumbersInText(tt.text))
		})
	}
}

// --- LooksLikePhoneNumber ---

func TestLooksLikePhoneNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		s    string
		want bool
	}{
		{"standard phone", "+1234567890", true},
		{"digits only", "9876543210", true},
		{"with dashes", "123-456-7890", true},
		{"with spaces", "123 456 7890", true},
		{"too short", "12345", false},
		{"text", "hello world", false},
		{"email", "test@example.com", false},
		{"mixed mostly text", "abc1234567xyz", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, LooksLikePhoneNumber(tt.s))
		})
	}
}

// --- MaskIfPhoneNumber ---

func TestMaskIfPhoneNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		s    string
		want string
	}{
		{"phone number", "+1234567890", "*******7890"},
		{"not phone", "hello", "hello"},
		{"email", "test@example.com", "test@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, MaskIfPhoneNumber(tt.s))
		})
	}
}

// --- parseDateRange ---

func TestParseDateRange_Valid(t *testing.T) {
	t.Parallel()

	start, end, errMsg := parseDateRange("2024-01-15", "2024-01-20")

	assert.Empty(t, errMsg)
	assert.Equal(t, 2024, start.Year())
	assert.Equal(t, time.January, start.Month())
	assert.Equal(t, 15, start.Day())
	assert.Equal(t, 0, start.Hour())

	// End should be end of day
	assert.Equal(t, 2024, end.Year())
	assert.Equal(t, time.January, end.Month())
	assert.Equal(t, 20, end.Day())
	assert.Equal(t, 23, end.Hour())
	assert.Equal(t, 59, end.Minute())
	assert.Equal(t, 59, end.Second())
}

func TestParseDateRange_InvalidStartDate(t *testing.T) {
	t.Parallel()

	_, _, errMsg := parseDateRange("invalid", "2024-01-20")

	assert.Contains(t, errMsg, "Invalid start date")
}

func TestParseDateRange_InvalidEndDate(t *testing.T) {
	t.Parallel()

	_, _, errMsg := parseDateRange("2024-01-15", "invalid")

	assert.Contains(t, errMsg, "Invalid end date")
}

func TestParseDateRange_WrongFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		startStr string
		endStr   string
		wantErr  string
	}{
		{"start DD/MM/YYYY", "15/01/2024", "2024-01-20", "Invalid start date"},
		{"end DD/MM/YYYY", "2024-01-15", "20/01/2024", "Invalid end date"},
		{"start MM-DD-YYYY", "01-15-2024", "2024-01-20", "Invalid start date"},
		{"end MM-DD-YYYY", "2024-01-15", "01-20-2024", "Invalid end date"},
		{"start empty", "", "2024-01-20", "Invalid start date"},
		{"end empty", "2024-01-15", "", "Invalid end date"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, errMsg := parseDateRange(tt.startStr, tt.endStr)
			assert.Contains(t, errMsg, tt.wantErr)
		})
	}
}

func TestParseDateRange_SameDay(t *testing.T) {
	t.Parallel()

	start, end, errMsg := parseDateRange("2024-06-15", "2024-06-15")

	assert.Empty(t, errMsg)
	assert.Equal(t, start.Day(), end.Day())
	// Start at beginning of day, end at end of day
	assert.Equal(t, 0, start.Hour())
	assert.Equal(t, 23, end.Hour())
}

// --- validateExportColumns ---

func TestValidateExportColumns_AllColumnsValid(t *testing.T) {
	t.Parallel()

	requested := []string{"name", "email", "phone"}
	allowed := []string{"id", "name", "email", "phone", "created_at"}

	validated, err := validateExportColumns(requested, allowed)

	assert.NoError(t, err)
	assert.Equal(t, requested, validated, "Should preserve order and return all requested columns")
}

func TestValidateExportColumns_InvalidColumn(t *testing.T) {
	t.Parallel()

	requested := []string{"name", "password", "email"}
	allowed := []string{"id", "name", "email", "phone"}

	validated, err := validateExportColumns(requested, allowed)

	assert.Error(t, err)
	assert.Nil(t, validated)
	assert.Contains(t, err.Error(), "password", "Error should mention the invalid column")
	assert.Contains(t, err.Error(), "not allowed for export", "Error should be descriptive")
}

func TestValidateExportColumns_EmptyRequested(t *testing.T) {
	t.Parallel()

	requested := []string{}
	allowed := []string{"id", "name", "email"}

	validated, err := validateExportColumns(requested, allowed)

	assert.NoError(t, err)
	assert.Empty(t, validated, "Empty request should return empty result")
}

func TestValidateExportColumns_DuplicateColumns(t *testing.T) {
	t.Parallel()

	requested := []string{"name", "name", "email"}
	allowed := []string{"id", "name", "email"}

	validated, err := validateExportColumns(requested, allowed)

	assert.NoError(t, err)
	// Should preserve duplicates as that's the caller's responsibility
	assert.Equal(t, []string{"name", "name", "email"}, validated)
}

func TestValidateExportColumns_PartiallyValid(t *testing.T) {
	t.Parallel()

	requested := []string{"name", "invalid_column", "email"}
	allowed := []string{"id", "name", "email", "phone"}

	validated, err := validateExportColumns(requested, allowed)

	assert.Error(t, err)
	assert.Nil(t, validated)
	assert.Contains(t, err.Error(), "invalid_column")
}

func TestValidateExportColumns_OrderPreserved(t *testing.T) {
	t.Parallel()

	requested := []string{"email", "name", "phone"}
	allowed := []string{"id", "phone", "name", "email"}

	validated, err := validateExportColumns(requested, allowed)

	assert.NoError(t, err)
	assert.Equal(t, []string{"email", "name", "phone"}, validated, "Should preserve original order")
}

func TestValidateExportColumns_AllRequestedColumnsValid(t *testing.T) {
	t.Parallel()

	requested := []string{"id", "name", "email", "phone"}
	allowed := []string{"id", "name", "email", "phone"}

	validated, err := validateExportColumns(requested, allowed)

	assert.NoError(t, err)
	assert.Equal(t, requested, validated)
}

func TestValidateExportColumns_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		requested []string
		allowed   []string
		wantErr   bool
		errMsg    string
		want      []string
	}{
		{
			name:      "all valid columns",
			requested: []string{"name", "email"},
			allowed:   []string{"id", "name", "email"},
			wantErr:   false,
			want:      []string{"name", "email"},
		},
		{
			name:      "invalid column present",
			requested: []string{"name", "password"},
			allowed:   []string{"id", "name", "email"},
			wantErr:   true,
			errMsg:    "column 'password' is not allowed for export",
		},
		{
			name:      "empty requested list",
			requested: []string{},
			allowed:   []string{"id", "name"},
			wantErr:   false,
			want:      []string{},
		},
		{
			name:      "duplicates preserved",
			requested: []string{"name", "name"},
			allowed:   []string{"id", "name"},
			wantErr:   false,
			want:      []string{"name", "name"},
		},
		{
			name:      "multiple invalid columns",
			requested: []string{"invalid1", "name", "invalid2"},
			allowed:   []string{"id", "name", "email"},
			wantErr:   true,
			errMsg:    "column 'invalid1' is not allowed for export",
		},
		{
			name:      "case sensitive validation",
			requested: []string{"Name"},       // uppercase
			allowed:   []string{"id", "name"}, // lowercase
			wantErr:   true,
			errMsg:    "column 'Name' is not allowed for export",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			validated, err := validateExportColumns(tt.requested, tt.allowed)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, validated)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, validated)
			}
		})
	}
}

// --- validateRequiredColumns ---

func TestValidateRequiredColumns_AllRequiredPresent(t *testing.T) {
	t.Parallel()

	colIndex := map[string]int{
		"name":  0,
		"email": 1,
		"phone": 2,
	}
	required := []string{"name", "email"}

	err := validateRequiredColumns(colIndex, required)

	assert.NoError(t, err, "Should not error when all required columns present")
}

func TestValidateRequiredColumns_MissingRequiredColumn(t *testing.T) {
	t.Parallel()

	colIndex := map[string]int{
		"name":  0,
		"phone": 1,
	}
	required := []string{"name", "email"}

	err := validateRequiredColumns(colIndex, required)

	assert.Error(t, err, "Should error when required column missing")
	assert.Contains(t, err.Error(), "email", "Error should mention missing column")
	assert.Contains(t, err.Error(), "not found in CSV", "Error should be descriptive")
}

func TestValidateRequiredColumns_EmptyRequiredList(t *testing.T) {
	t.Parallel()

	colIndex := map[string]int{
		"name": 0,
	}
	required := []string{}

	err := validateRequiredColumns(colIndex, required)

	assert.NoError(t, err, "Empty required list should always pass")
}

func TestValidateRequiredColumns_CaseInsensitiveMatch(t *testing.T) {
	t.Parallel()

	colIndex := map[string]int{
		"Name":  0,
		"EMAIL": 1,
		"Phone": 2,
	}
	required := []string{"name", "email", "phone"}

	err := validateRequiredColumns(colIndex, required)

	assert.NoError(t, err, "Should match columns case-insensitively")
}

func TestValidateRequiredColumns_UnderscoreSpaceVariation(t *testing.T) {
	t.Parallel()

	// CSV has spaces instead of underscores
	colIndex := map[string]int{
		"phone number": 0,
		"email":        1,
	}
	required := []string{"phone_number", "email"}

	err := validateRequiredColumns(colIndex, required)

	assert.NoError(t, err, "Should match underscore to space variation")
}

func TestValidateRequiredColumns_SpaceToUnderscoreVariation(t *testing.T) {
	t.Parallel()

	// CSV has underscores instead of spaces
	colIndex := map[string]int{
		"phone_number": 0,
		"email":        1,
	}
	required := []string{"phone number", "email"}

	err := validateRequiredColumns(colIndex, required)

	assert.NoError(t, err, "Should match space to underscore variation")
}

func TestValidateRequiredColumns_MultipleMissingColumns(t *testing.T) {
	t.Parallel()

	colIndex := map[string]int{
		"name": 0,
	}
	required := []string{"name", "email", "phone"}

	err := validateRequiredColumns(colIndex, required)

	assert.Error(t, err, "Should error when multiple columns missing")
	assert.Contains(t, err.Error(), "email", "Should report first missing column")
}

func TestValidateRequiredColumns_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		colIndex  map[string]int
		required  []string
		wantErr   bool
		errSubstr string
	}{
		{
			name: "all required columns present",
			colIndex: map[string]int{
				"name":  0,
				"email": 1,
				"phone": 2,
			},
			required: []string{"name", "email"},
			wantErr:  false,
		},
		{
			name: "missing required column",
			colIndex: map[string]int{
				"name":  0,
				"phone": 1,
			},
			required:  []string{"name", "email"},
			wantErr:   true,
			errSubstr: "email",
		},
		{
			name: "case insensitive match",
			colIndex: map[string]int{
				"NAME":  0,
				"Email": 1,
			},
			required: []string{"name", "email"},
			wantErr:  false,
		},
		{
			name: "underscore space variation",
			colIndex: map[string]int{
				"phone number": 0,
			},
			required: []string{"phone_number"},
			wantErr:  false,
		},
		{
			name: "space underscore variation",
			colIndex: map[string]int{
				"phone_number": 0,
			},
			required: []string{"phone number"},
			wantErr:  false,
		},
		{
			name: "empty required list",
			colIndex: map[string]int{
				"name": 0,
			},
			required: []string{},
			wantErr:  false,
		},
		{
			name:      "empty col index with required columns",
			colIndex:  map[string]int{},
			required:  []string{"name"},
			wantErr:   true,
			errSubstr: "name",
		},
		{
			name: "all variations present",
			colIndex: map[string]int{
				"Name":         0,
				"email":        1,
				"Phone Number": 2,
			},
			required: []string{"name", "email", "phone_number"},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateRequiredColumns(tt.colIndex, tt.required)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
