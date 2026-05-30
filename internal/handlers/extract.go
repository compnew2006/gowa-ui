package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

type ExtractContactResponse struct {
	ID            uuid.UUID  `json:"id"`
	PhoneNumber   string     `json:"phone_number"`
	ProfileName   string     `json:"profile_name"`
	LastMessageAt *time.Time `json:"last_message_at"`
	MessageCount  int64      `json:"message_count"`
	UnreadCount   int        `json:"unread_count"`
	InstanceID    *uuid.UUID `json:"instance_id,omitempty"`
	WhatsAppAccount string  `json:"whatsapp_account,omitempty"`
}

type ExtractionStatsResponse struct {
	InstanceID    uuid.UUID `json:"instance_id"`
	InstanceName  string    `json:"instance_name"`
	PhoneNumber   string    `json:"phone_number"`
	TotalContacts int64     `json:"total_contacts"`
	TotalMessages int64     `json:"total_messages"`
	LastSyncAt    *time.Time `json:"last_sync_at"`
	Status        string    `json:"status"`
}

type ExtractSyncRequest struct {
	InstanceID string `json:"instance_id"`
}

// ListExtractableContacts returns paginated contacts with message counts
func (a *App) ListExtractableContacts(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, "campaigns", "read"); err != nil {
		return nil
	}

	pg := parsePaginationWithDefaults(r, 50, 500)
	search := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("search")))
	instanceIDStr := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("instance_id")))

	query := requestDB.Model(&models.Contact{}).
		Where("contacts.organization_id = ?", orgID)

	if instanceIDStr != "" {
		instanceID, parseErr := uuid.Parse(instanceIDStr)
		if parseErr != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid instance_id", nil, "")
		}
		query = query.Where("contacts.instance_id = ?", instanceID)
	}

	if search != "" {
		pattern := "%" + search + "%"
		query = query.Where("contacts.phone_number LIKE ? OR contacts.profile_name ILIKE ?", pattern, pattern)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		a.Log.Error("Failed to count extractable contacts", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to count contacts", nil, "")
	}

	type contactWithCount struct {
		models.Contact
		MessageCount int64
	}

	var enriched []contactWithCount
	if err := query.
		Select("contacts.*, COALESCE(meta.message_count, 0) AS message_count").
		Joins("LEFT JOIN (SELECT contact_id, COUNT(*) AS message_count FROM messages WHERE organization_id = ? GROUP BY contact_id) meta ON meta.contact_id = contacts.id", orgID).
		Order("contacts.last_message_at DESC NULLS LAST").
		Offset(pg.Offset).Limit(pg.Limit).
		Scan(&enriched).Error; err != nil {
		a.Log.Error("Failed to list extractable contacts", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list contacts", nil, "")
	}

	response := make([]ExtractContactResponse, 0, len(enriched))
	for _, c := range enriched {
		resp := ExtractContactResponse{
			ID:            c.ID,
			PhoneNumber:   c.PhoneNumber,
			ProfileName:   c.ProfileName,
			LastMessageAt: c.LastMessageAt,
			MessageCount:  c.MessageCount,
			WhatsAppAccount: c.WhatsAppAccount,
		}
		if c.InstanceID != nil {
			resp.InstanceID = c.InstanceID
		}
		response = append(response, resp)
	}

	return r.SendEnvelope(map[string]interface{}{
		"data":       response,
		"total":      total,
		"page":       pg.Page,
		"limit":      pg.Limit,
		"total_pages": (int(total) + pg.Limit - 1) / pg.Limit,
	})
}

// ExportExtractedContacts exports contacts with message counts as CSV
func (a *App) ExportExtractedContacts(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, "campaigns", "export"); err != nil {
		return nil
	}

	var req struct {
		InstanceID string `json:"instance_id"`
		Search     string `json:"search"`
	}
	if err := json.Unmarshal(r.RequestCtx.PostBody(), &req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	query := requestDB.Model(&models.Contact{}).
		Where("contacts.organization_id = ?", orgID)

	if req.InstanceID != "" {
		instanceID, parseErr := uuid.Parse(req.InstanceID)
		if parseErr != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid instance_id", nil, "")
		}
		query = query.Where("contacts.instance_id = ?", instanceID)
	}

	if req.Search != "" {
		pattern := "%" + req.Search + "%"
		query = query.Where("contacts.phone_number LIKE ? OR contacts.profile_name ILIKE ?", pattern, pattern)
	}

	type exportContact struct {
		models.Contact
		MessageCount int64
	}

	var enriched []exportContact
	if err := query.
		Select("contacts.*, COALESCE(meta.message_count, 0) AS message_count").
		Joins("LEFT JOIN (SELECT contact_id, COUNT(*) AS message_count FROM messages WHERE organization_id = ? GROUP BY contact_id) meta ON meta.contact_id = contacts.id", orgID).
		Order("contacts.last_message_at DESC NULLS LAST").
		Scan(&enriched).Error; err != nil {
		a.Log.Error("Failed to export contacts", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to export contacts", nil, "")
	}

	shouldMask := a.ShouldMaskPhoneNumbers(orgID)

	var buf strings.Builder
	buf.WriteString("\xEF\xBB\xBF")
	writer := csv.NewWriter(&buf)

	header := []string{"Phone Number", "Name", "Last Message At", "Message Count"}
	_ = writer.Write(header)

	for _, c := range enriched {
		phone := c.PhoneNumber
		name := c.ProfileName
		if shouldMask {
			phone = MaskPhoneNumber(phone)
			name = MaskIfPhoneNumber(name)
		}

		lastMsgAt := ""
		if c.LastMessageAt != nil {
			lastMsgAt = c.LastMessageAt.Format(time.RFC3339)
		}

		row := []string{phone, name, lastMsgAt, strconv.FormatInt(c.MessageCount, 10)}
		for j, cell := range row {
			if len(cell) > 0 && (cell[0] == '=' || cell[0] == '@') {
				row[j] = "'" + cell
			} else if looksLikeNumericString(cell) {
				row[j] = "\t" + cell
			}
		}
		_ = writer.Write(row)
	}

	writer.Flush()

	filename := fmt.Sprintf("extracted_contacts_%s.csv", time.Now().Format("20060102_150405"))
	r.RequestCtx.Response.Header.Set("Content-Type", "text/csv; charset=utf-8")
	r.RequestCtx.Response.Header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	r.RequestCtx.SetBody([]byte(buf.String()))

	return nil
}

// GetExtractionStats returns per-instance extraction statistics
func (a *App) GetExtractionStats(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, "campaigns", "read"); err != nil {
		return nil
	}

	instanceIDStr := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("instance_id")))

	var instances []models.WhatsAppInstance
	q := requestDB.Where("organization_id = ?", orgID)
	if instanceIDStr != "" {
		instID, parseErr := uuid.Parse(instanceIDStr)
		if parseErr != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid instance_id", nil, "")
		}
		q = q.Where("id = ?", instID)
	}
	if err := q.Find(&instances).Error; err != nil {
		a.Log.Error("Failed to list instances for stats", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to get stats", nil, "")
	}

	stats := make([]ExtractionStatsResponse, 0, len(instances))
	for _, inst := range instances {
		var totalContacts int64
		a.DB.Model(&models.Contact{}).
			Where("organization_id = ? AND instance_id = ?", orgID, inst.ID).
			Count(&totalContacts)

		var totalMessages int64
		a.DB.Model(&models.Message{}).
			Where("organization_id = ? AND instance_id = ?", orgID, inst.ID).
			Count(&totalMessages)

		stats = append(stats, ExtractionStatsResponse{
			InstanceID:    inst.ID,
			InstanceName:  inst.Name,
			PhoneNumber:   inst.PhoneNumber,
			TotalContacts: totalContacts,
			TotalMessages: totalMessages,
			LastSyncAt:    inst.LastConnectedAt,
			Status:        string(inst.Status),
		})
	}

	return r.SendEnvelope(map[string]interface{}{
		"stats": stats,
	})
}

// TriggerHistorySync triggers a history sync for a given instance
func (a *App) TriggerHistorySync(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, "campaigns", "update"); err != nil {
		return nil
	}

	var req ExtractSyncRequest
	if err := json.Unmarshal(r.RequestCtx.PostBody(), &req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}

	instanceID, err := uuid.Parse(req.InstanceID)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid instance_id", nil, "")
	}

	var instance models.WhatsAppInstance
	if err := requestDB.Where("id = ? AND organization_id = ?", instanceID, orgID).First(&instance).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Instance not found", nil, "")
		}
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to find instance", nil, "")
	}

	if a.WhatsmeowManager == nil {
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "WhatsApp manager not initialized", nil, "")
	}

	client := a.WhatsmeowManager.GetClient(instanceID)
	if client == nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Instance is not connected", nil, "")
	}

	msg := client.BuildHistorySyncRequest(nil, 100)
	if _, err := client.SendPeerMessage(r.RequestCtx, msg); err != nil {
		a.Log.Error("Failed to trigger history sync", "error", err, "instance_id", instanceID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to trigger history sync", nil, "")
	}

	a.Log.Info("History sync triggered successfully", "instance_id", instanceID)
	return r.SendEnvelope(map[string]interface{}{
		"message":     "History sync triggered successfully",
		"instance_id": instanceID.String(),
	})
}
