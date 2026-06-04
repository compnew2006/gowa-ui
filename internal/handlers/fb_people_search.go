package handlers

import (
	"errors"
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

type FBPeopleSearchResponse struct {
	Name           string `json:"name"`
	PageID         string `json:"page_id"`
	FollowersCount string `json:"followers_count"`
}

func (a *App) SearchFBPeople(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceAccounts, models.ActionRead); err != nil {
		return nil
	}

	campaignID := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("campaign_id")))
	if campaignID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "campaign_id is required", nil, "campaign_id")
	}

	page, err := a.parseIntQuery(r, "page", 1, 1, 1<<31-1)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "invalid page", nil, "page")
	}

	perPage, err := a.parseIntQuery(r, "per_page", 25, 1, 200)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "invalid per_page (1-200)", nil, "per_page")
	}

	q := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("q")))

	var total int64
	countQuery := requestDB.Model(&models.FBPeopleSearch{}).
		Where("organization_id = ? AND campaign_id = ?", orgID, campaignID)
	if q != "" {
		like := "%" + q + "%"
		countQuery = countQuery.Where("name ILIKE ? OR page_id ILIKE ?", like, like)
	}
	if err := countQuery.Count(&total).Error; err != nil {
		a.Log.Error("Failed to count FB people searches", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to search people", nil, "")
	}

	var rows []models.FBPeopleSearch
	listQuery := requestDB.Model(&models.FBPeopleSearch{}).
		Where("organization_id = ? AND campaign_id = ?", orgID, campaignID).
		Order("created_at DESC, id DESC").
		Limit(perPage).
		Offset((page - 1) * perPage)
	if q != "" {
		like := "%" + q + "%"
		listQuery = listQuery.Where("name ILIKE ? OR page_id ILIKE ?", like, like)
	}
	if err := listQuery.Find(&rows).Error; err != nil {
		a.Log.Error("Failed to list FB people searches", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to search people", nil, "")
	}

	data := make([]FBPeopleSearchResponse, 0, len(rows))
	for _, row := range rows {
		data = append(data, FBPeopleSearchResponse{
			Name:           row.Name,
			PageID:         row.PageID,
			FollowersCount: row.FollowersCount,
		})
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(perPage) - 1) / int64(perPage))
	}

	return r.SendEnvelope(map[string]interface{}{
		"success":     true,
		"campaign_id": campaignID,
		"page":        page,
		"per_page":    perPage,
		"total":       total,
		"total_pages": totalPages,
		"data":        data,
	})
}

type AddFBPeopleContactsRequest struct {
	ListName string `json:"name"`
	Data     []struct {
		Identifier string `json:"identifier"`
		Name       string `json:"name"`
	} `json:"data"`
}

func (a *App) AddFBPeopleContacts(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}

	// Check permission
	if !a.HasPermission(userID, models.ResourceContacts, models.ActionWrite, orgID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You do not have permission to create contacts", nil, "")
	}

	var req AddFBPeopleContactsRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	req.ListName = strings.TrimSpace(req.ListName)
	if req.ListName == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "name is required", nil, "name")
	}

	if len(req.Data) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "data cannot be empty", nil, "data")
	}

	if len(req.Data) > 5000 {
		return r.SendErrorEnvelope(fasthttp.StatusRequestEntityTooLarge, "Maximum 5000 contacts allowed per request", nil, "data")
	}

	var createdCount int
	var updatedCount int

	// Begin transaction
	tx := requestDB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for _, item := range req.Data {
		identifier := strings.TrimSpace(item.Identifier)
		// Clean/normalize identifier - keep only digits (and '+' sign if present)
		var cleaned []rune
		for _, char := range identifier {
			if (char >= '0' && char <= '9') || char == '+' {
				cleaned = append(cleaned, char)
			}
		}
		normalizedPhone := string(cleaned)
		if len(normalizedPhone) > 0 && normalizedPhone[0] == '+' {
			normalizedPhone = normalizedPhone[1:]
		}

		if normalizedPhone == "" || len(normalizedPhone) < 5 || len(normalizedPhone) > 50 {
			continue
		}

		displayName := strings.TrimSpace(item.Name)
		if displayName == "" {
			displayName = normalizedPhone
		}

		var existing models.Contact
		err := tx.Unscoped().Where("organization_id = ? AND phone_number = ?", orgID, normalizedPhone).First(&existing).Error
		if err == nil {
			// Contact exists, update it
			updates := map[string]any{
				"deleted_at": nil,
			}
			if existing.ProfileName == "" {
				updates["profile_name"] = displayName
			}
			// Append the list name to tags if not already there
			hasTag := false
			tags := make(models.JSONBArray, 0)
			for _, tag := range existing.Tags {
				if s, ok := tag.(string); ok {
					tags = append(tags, s)
					if strings.EqualFold(s, req.ListName) {
						hasTag = true
					}
				}
			}
			if !hasTag {
				tags = append(tags, req.ListName)
				updates["tags"] = tags
			}

			if err := tx.Model(&existing).Updates(updates).Error; err != nil {
				tx.Rollback()
				a.Log.Error("Failed to update contact during bulk add", "error", err)
				return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to import contacts", nil, "")
			}
			updatedCount++
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new contact
			newContact := models.Contact{
				BaseModel: models.BaseModel{
					ID: uuid.New(),
				},
				OrganizationID: orgID,
				PhoneNumber:    normalizedPhone,
				ProfileName:    displayName,
				Status:         models.ChatStatusPending,
				Tags:           models.JSONBArray{req.ListName},
				IsRead:         true,
			}
			if err := tx.Create(&newContact).Error; err != nil {
				tx.Rollback()
				a.Log.Error("Failed to create contact during bulk add", "error", err)
				return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to import contacts", nil, "")
			}
			createdCount++
		} else {
			tx.Rollback()
			a.Log.Error("Failed to check existing contact during bulk add", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to import contacts", nil, "")
		}
	}

	if err := tx.Commit().Error; err != nil {
		a.Log.Error("Failed to commit bulk add transaction", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to import contacts", nil, "")
	}

	return r.SendEnvelope(map[string]interface{}{
		"success": true,
		"created": createdCount,
		"updated": updatedCount,
		"total":   createdCount + updatedCount,
	})
}

func (a *App) ListFBPeopleCampaigns(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceAccounts, models.ActionRead); err != nil {
		return nil
	}

	var campaigns []string
	if err := requestDB.Model(&models.FBPeopleSearch{}).
		Where("organization_id = ?", orgID).
		Distinct().
		Pluck("campaign_id", &campaigns).Error; err != nil {
		a.Log.Error("Failed to list unique people search campaigns", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list campaigns", nil, "")
	}

	return r.SendEnvelope(map[string]interface{}{
		"success":   true,
		"campaigns": campaigns,
	})
}
