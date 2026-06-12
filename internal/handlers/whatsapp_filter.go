package handlers

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

// WhatsAppFilterBatchRequest represents a JSON request to create a filter batch
type WhatsAppFilterBatchRequest struct {
	ConnectionID string   `json:"connection_id"`
	Phones       []string `json:"phones"`
	Names        []string `json:"names,omitempty"`
}

// CreateWhatsAppFilterBatch creates a new filter batch and triggers validation
func (a *App) CreateWhatsAppFilterBatch(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if err := a.requirePermission(r, userID, models.ResourceWhatsAppFilter, models.ActionWrite); err != nil {
		return err
	}

	var connectionIDStr string
	var inputPhones []string
	var inputNames []string

	// Determine request type: multipart form (file upload) or JSON
	contentType := string(r.RequestCtx.Request.Header.ContentType())
	if strings.Contains(contentType, "multipart/form-data") {
		// Multipart Form
		form, err := r.RequestCtx.MultipartForm()
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid multipart form", nil, "")
		}

		connValues := form.Value["connection_id"]
		if len(connValues) == 0 {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "connection_id is required", nil, "")
		}
		connectionIDStr = connValues[0]

		files := form.File["file"]
		if len(files) == 0 {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "file is required", nil, "")
		}
		fileHeader := files[0]

		file, err := fileHeader.Open()
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Failed to read file", nil, "")
		}
		defer file.Close()

		// Limit CSV file size to 10MB
		const maxCSVSize = 10 << 20
		limitedReader := io.LimitReader(file, maxCSVSize+1)

		reader := csv.NewReader(limitedReader)
		header, err := reader.Read()
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Failed to read CSV header", nil, "")
		}

		// Find relevant column indexes
		phoneIdx := -1
		nameIdx := -1

		for i, h := range header {
			h = strings.ToLower(strings.TrimSpace(h))
			if h == "phone" || h == "phone_number" || h == "number" || h == "telephone" || h == "الرقم" || h == "هاتف" {
				phoneIdx = i
			} else if h == "name" || h == "contact_name" || h == "full_name" || h == "username" || h == "الاسم" {
				nameIdx = i
			}
		}

		// If no matching headers are found, assume first column is phone, second is name
		if phoneIdx == -1 {
			phoneIdx = 0
			if len(header) > 1 {
				nameIdx = 1
			}
		}

		for {
			row, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				continue
			}

			if phoneIdx < len(row) {
				phone := strings.TrimSpace(row[phoneIdx])
				if phone != "" {
					inputPhones = append(inputPhones, phone)
					name := ""
					if nameIdx != -1 && nameIdx < len(row) {
						name = strings.TrimSpace(row[nameIdx])
					}
					inputNames = append(inputNames, name)
				}
			}
		}
	} else {
		// JSON payload
		var req WhatsAppFilterBatchRequest
		if err := json.Unmarshal(r.RequestCtx.PostBody(), &req); err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
		}
		connectionIDStr = req.ConnectionID
		inputPhones = req.Phones
		inputNames = req.Names
	}

	connectionID, err := uuid.Parse(connectionIDStr)
	if err != nil || connectionID == uuid.Nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "connection_id is invalid", nil, "connection_id")
	}

	// Clean and normalize phone inputs
	var validPhones []string
	var validNames []string

	for i, phone := range inputPhones {
		phone = cleanPhoneNumberDigits(phone)
		if phone == "" {
			continue
		}
		validPhones = append(validPhones, phone)
		name := ""
		if i < len(inputNames) {
			name = inputNames[i]
		}
		validNames = append(validNames, name)
	}

	if len(validPhones) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "No valid phone numbers found in input", nil, "")
	}

	// 10,000 limit per batch to protect performance
	if len(validPhones) > 10000 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "A maximum of 10,000 numbers can be checked per batch", nil, "")
	}

	// Resolve the active provider configuration and verify connection is connected/available
	var whatsappAccount string
	var instanceID *uuid.UUID

	if a.Config.WhatsApp.Provider == "whatsmeow" {
		var instance models.WhatsAppInstance
		if err := requestDB.Where("id = ? AND organization_id = ?", connectionID, orgID).First(&instance).Error; err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "WhatsApp connection instance not found", nil, "connection_id")
		}
		if instance.Status != models.InstanceStatusConnected {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "WhatsApp instance is disconnected. Please connect it first.", nil, "connection_id")
		}
		instanceID = &instance.ID
		whatsappAccount = instance.Name
	} else {
		// Try whatsmeow instance first, then fall back to Meta account
		var instance models.WhatsAppInstance
		if err := requestDB.Where("id = ? AND organization_id = ?", connectionID, orgID).First(&instance).Error; err == nil {
			if instance.Status != models.InstanceStatusConnected {
				return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "WhatsApp instance is disconnected. Please connect it first.", nil, "connection_id")
			}
			instanceID = &instance.ID
			whatsappAccount = instance.Name
		} else {
			var account models.WhatsAppAccount
			if err := requestDB.Where("id = ? AND organization_id = ?", connectionID, orgID).First(&account).Error; err != nil {
				return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Meta WhatsApp account configuration not found", nil, "connection_id")
			}
			whatsappAccount = account.Name
		}
	}

	// Create Batch & Result records in GORM transaction
	batch := models.WhatsAppFilterBatch{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  orgID,
		CreatedBy:       userID,
		WhatsAppAccount: whatsappAccount,
		InstanceID:      instanceID,
		Status:          models.FilterStatusPending,
		TotalNumbers:    len(validPhones),
		ValidNumbers:    0,
		InvalidNumbers:  0,
	}

	results := make([]models.WhatsAppFilterResult, len(validPhones))
	for i, phone := range validPhones {
		results[i] = models.WhatsAppFilterResult{
			BaseModel:   models.BaseModel{ID: uuid.New()},
			BatchID:     batch.ID,
			PhoneNumber: phone,
			ContactName: validNames[i],
			IsValid:     false,
		}
	}

	tx := requestDB.Session(&gorm.Session{NewDB: true}).Begin()
	if err := tx.Create(&batch).Error; err != nil {
		tx.Rollback()
		a.Log.Error("Failed to create filter batch in DB", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to initialize verification batch", nil, "")
	}

	// Insert results in batches of 100
	if err := tx.CreateInBatches(&results, 100).Error; err != nil {
		tx.Rollback()
		a.Log.Error("Failed to create filter results in DB", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to initialize verification batch details", nil, "")
	}

	if err := tx.Commit().Error; err != nil {
		a.Log.Error("Failed to commit filter transaction", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to save verification batch", nil, "")
	}

	// Enqueue redis job
	job := &queue.WhatsAppFilterJob{
		BatchID:           batch.ID,
		OrganizationID:    orgID,
		WhatsAppAccountID: connectionID,
		InstanceID:        instanceID,
		EnqueuedAt:        time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.Queue.EnqueueWhatsAppFilter(ctx, job); err != nil {
		a.Log.Error("Failed to enqueue whatsapp filter job", "error", err, "batch_id", batch.ID)
		// Set status to failed
		requestDB.Model(&batch).Update("status", models.FilterStatusFailed)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to enqueue validation job", nil, "")
	}

	return r.SendEnvelope(batch)
}

// ListWhatsAppFilterBatches returns paginated filter batches scoped to org
func (a *App) ListWhatsAppFilterBatches(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if err := a.requirePermission(r, userID, models.ResourceWhatsAppFilter, models.ActionRead); err != nil {
		return err
	}

	pg := parsePaginationWithDefaults(r, 20, 200)

	var batches []models.WhatsAppFilterBatch
	var total int64

	query := requestDB.Model(&models.WhatsAppFilterBatch{}).Where("organization_id = ?", orgID)
	query.Count(&total)

	if err := query.Order("created_at desc").
		Offset(pg.Offset).
		Limit(pg.Limit).
		Find(&batches).Error; err != nil {
		a.Log.Error("Failed to list filter batches", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list validation batches", nil, "")
	}

	return r.SendEnvelope(map[string]any{
		"data":  batches,
		"total": total,
		"page":  pg.Page,
		"limit": pg.Limit,
	})
}

// GetWhatsAppFilterBatch returns batch details and progress
func (a *App) GetWhatsAppFilterBatch(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if err := a.requirePermission(r, userID, models.ResourceWhatsAppFilter, models.ActionRead); err != nil {
		return err
	}

	batchIDStr := r.RequestCtx.UserValue("id").(string)
	batchID, err := uuid.Parse(batchIDStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid batch ID", nil, "")
	}

	var batch models.WhatsAppFilterBatch
	if err := requestDB.Where("id = ? AND organization_id = ?", batchID, orgID).First(&batch).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Validation batch not found", nil, "")
	}

	return r.SendEnvelope(batch)
}

// GetWhatsAppFilterBatchResults returns paginated verified numbers
func (a *App) GetWhatsAppFilterBatchResults(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if err := a.requirePermission(r, userID, models.ResourceWhatsAppFilter, models.ActionRead); err != nil {
		return err
	}

	batchIDStr := r.RequestCtx.UserValue("id").(string)
	batchID, err := uuid.Parse(batchIDStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid batch ID", nil, "")
	}

	// Ensure batch belongs to organization
	var batch models.WhatsAppFilterBatch
	if err := requestDB.Where("id = ? AND organization_id = ?", batchID, orgID).First(&batch).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Validation batch not found", nil, "")
	}

	pg := parsePaginationWithDefaults(r, 50, 200)

	// Filter query options
	status := string(r.RequestCtx.QueryArgs().Peek("status")) // all, valid, invalid
	search := string(r.RequestCtx.QueryArgs().Peek("q"))

	resultsDB := requestDB.Session(&gorm.Session{NewDB: true, Logger: a.DB.Logger, DryRun: false})
	var results []models.WhatsAppFilterResult
	var total int64

	query := resultsDB.Debug().Model(&models.WhatsAppFilterResult{}).Where("batch_id = ?", batchID)

	if status == "valid" {
		query = query.Where("is_valid = ?", true)
	} else if status == "invalid" {
		query = query.Where("is_valid = ?", false)
	}

	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where("phone_number LIKE ? OR contact_name ILIKE ?", searchPattern, searchPattern)
	}

	query.Count(&total)

	if err := query.Order("created_at asc").
		Offset(pg.Offset).
		Limit(pg.Limit).
		Find(&results).Error; err != nil {
		a.Log.Error("Failed to fetch filter results", "error", err, "batch_id", batchID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to retrieve verification details", nil, "")
	}

	return r.SendEnvelope(map[string]any{
		"data":  results,
		"total": total,
		"page":  pg.Page,
		"limit": pg.Limit,
	})
}

// ExportWhatsAppFilterResults streams a CSV export of batch results
func (a *App) ExportWhatsAppFilterResults(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if err := a.requirePermission(r, userID, models.ResourceWhatsAppFilter, models.ActionRead); err != nil {
		return err
	}

	batchIDStr := r.RequestCtx.UserValue("id").(string)
	batchID, err := uuid.Parse(batchIDStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid batch ID", nil, "")
	}

	// Ensure batch belongs to organization
	var batch models.WhatsAppFilterBatch
	if err := requestDB.Where("id = ? AND organization_id = ?", batchID, orgID).First(&batch).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Validation batch not found", nil, "")
	}

	status := string(r.RequestCtx.QueryArgs().Peek("status")) // all, valid, invalid
	search := string(r.RequestCtx.QueryArgs().Peek("q"))

	query := requestDB.Model(&models.WhatsAppFilterResult{}).Where("batch_id = ?", batchID)

	if status == "valid" {
		query = query.Where("is_valid = ?", true)
	} else if status == "invalid" {
		query = query.Where("is_valid = ?", false)
	}

	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where("phone_number LIKE ? OR contact_name ILIKE ?", searchPattern, searchPattern)
	}

	rows, err := query.Order("created_at asc").Rows()
	if err != nil {
		a.Log.Error("Failed to query export filter results", "error", err, "batch_id", batchID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to query results for export", nil, "")
	}
	defer rows.Close()

	// Build CSV stream
	var buf strings.Builder
	writer := csv.NewWriter(&buf)

	// CSV Headers
	_ = writer.Write([]string{"Phone Number", "Contact Name", "Registered on WhatsApp", "Checked At", "Error Message"})

	shouldMask := a.ShouldMaskPhoneNumbers(orgID)

	for rows.Next() {
		var item models.WhatsAppFilterResult
		if err := requestDB.ScanRows(rows, &item); err != nil {
			continue
		}

		phone := item.PhoneNumber
		if shouldMask {
			phone = MaskPhoneNumber(phone)
		}

		name := item.ContactName
		if shouldMask {
			name = MaskIfPhoneNumber(name)
		}

		checkedStr := ""
		if item.CheckedAt != nil {
			checkedStr = item.CheckedAt.Format(time.RFC3339)
		}

		isValidStr := "false"
		if item.IsValid {
			isValidStr = "true"
		}

		csvRow := []string{
			phone,
			name,
			isValidStr,
			checkedStr,
			item.ErrorMessage,
		}

		// Escape potential injection characters (= and @)
		for j, cell := range csvRow {
			if len(cell) > 0 && (cell[0] == '=' || cell[0] == '@') {
				csvRow[j] = "'" + cell
			}
		}

		_ = writer.Write(csvRow)
	}

	writer.Flush()

	filename := fmt.Sprintf("whatsapp_filter_results_%s_%s.csv", batchIDStr[:8], time.Now().Format("20060102_150405"))
	r.RequestCtx.Response.Header.Set("Content-Type", "text/csv; charset=utf-8")
	r.RequestCtx.Response.Header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"; filename*=utf-8''%s", filename, url.QueryEscape(filename)))
	r.RequestCtx.SetBody([]byte("\xEF\xBB\xBF" + buf.String()))

	return nil
}

// DeleteWhatsAppFilterBatch deletes a batch and all of its results
func (a *App) DeleteWhatsAppFilterBatch(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	if err := a.requirePermission(r, userID, models.ResourceWhatsAppFilter, models.ActionDelete); err != nil {
		return err
	}

	batchIDStr := r.RequestCtx.UserValue("id").(string)
	batchID, err := uuid.Parse(batchIDStr)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid batch ID", nil, "")
	}

	// Verify ownership before deletion
	var batch models.WhatsAppFilterBatch
	if err := requestDB.Where("id = ? AND organization_id = ?", batchID, orgID).First(&batch).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Validation batch not found", nil, "")
	}

	// Delete batch and results in transaction
	tx := requestDB.Session(&gorm.Session{NewDB: true}).Begin()
	if err := tx.Where("batch_id = ?", batch.ID).Delete(&models.WhatsAppFilterResult{}).Error; err != nil {
		tx.Rollback()
		a.Log.Error("Failed to delete filter results from DB", "error", err, "batch_id", batch.ID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete validation details", nil, "")
	}

	if err := tx.Where("id = ?", batch.ID).Delete(&models.WhatsAppFilterBatch{}).Error; err != nil {
		tx.Rollback()
		a.Log.Error("Failed to delete filter batch from DB", "error", err, "batch_id", batch.ID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to delete validation batch", nil, "")
	}

	if err := tx.Commit().Error; err != nil {
		a.Log.Error("Failed to commit deletion transaction", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to finalize deletion", nil, "")
	}

	return r.SendEnvelope(map[string]bool{"success": true})
}

// Helpers

func cleanPhoneNumberDigits(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	// Strip all non-digit characters except "+" (Meta likes + prefix)
	var sb strings.Builder
	for i, r := range trimmed {
		if r >= '0' && r <= '9' {
			sb.WriteRune(r)
		} else if r == '+' && i == 0 {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}
