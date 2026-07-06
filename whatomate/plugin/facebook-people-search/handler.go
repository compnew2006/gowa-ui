package facebookpeoplesearch

import (
	"errors"
	"strings"

	"github.com/compnew2006/whatomate/internal/middleware"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/tenant"
	facebookcore "github.com/compnew2006/whatomate/plugin/facebook-core"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

type PeopleSearchResponse struct {
	Name           string `json:"name"`
	PageID         string `json:"page_id"`
	FollowersCount string `json:"followers_count"`
}

type AddContactsRequest struct {
	ListName string `json:"name"`
	Data     []struct {
		Identifier string `json:"identifier"`
		Name       string `json:"name"`
	} `json:"data"`
}

func (p *Plugin) requestContext(r *fastglue.Request) (*gorm.DB, uuid.UUID, uuid.UUID, bool) {
	organizationID, organizationOK := middleware.GetOrganizationID(r)
	userID, userOK := middleware.GetUserID(r)
	if !organizationOK || !userOK || organizationID == uuid.Nil || userID == uuid.Nil || p.app == nil {
		_ = r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
		return nil, uuid.Nil, uuid.Nil, false
	}
	return tenant.ScopedDB(p.app.DB, organizationID), organizationID, userID, true
}

func (p *Plugin) SearchFBPeople(r *fastglue.Request) error {
	requestDB, organizationID, userID, ok := p.requestContext(r)
	if !ok {
		return nil
	}
	if !p.app.HasPermission(userID, models.ResourceAccounts, models.ActionRead, organizationID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Insufficient permissions", nil, "")
	}

	campaignID := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("campaign_id")))
	if campaignID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "campaign_id is required", nil, "campaign_id")
	}
	page, err := facebookcore.ParseIntQuery(r, "page", 1, 1, 1<<31-1)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "invalid page", nil, "page")
	}
	perPage, err := facebookcore.ParseIntQuery(r, "per_page", 25, 1, 200)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "invalid per_page (1-200)", nil, "per_page")
	}

	queryText := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("q")))
	var total int64
	countQuery := requestDB.Model(&PeopleSearch{}).
		Where("organization_id = ? AND campaign_id = ?", organizationID, campaignID)
	if queryText != "" {
		like := "%" + queryText + "%"
		countQuery = countQuery.Where("name ILIKE ? OR page_id ILIKE ?", like, like)
	}
	if err := countQuery.Count(&total).Error; err != nil {
		p.app.Log.Error("Failed to count FB people searches", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to search people", nil, "")
	}

	var rows []PeopleSearch
	listQuery := requestDB.Model(&PeopleSearch{}).
		Where("organization_id = ? AND campaign_id = ?", organizationID, campaignID).
		Order("created_at DESC, id DESC").
		Limit(perPage).
		Offset((page - 1) * perPage)
	if queryText != "" {
		like := "%" + queryText + "%"
		listQuery = listQuery.Where("name ILIKE ? OR page_id ILIKE ?", like, like)
	}
	if err := listQuery.Find(&rows).Error; err != nil {
		p.app.Log.Error("Failed to list FB people searches", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to search people", nil, "")
	}

	data := make([]PeopleSearchResponse, 0, len(rows))
	for _, row := range rows {
		data = append(data, PeopleSearchResponse{
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

func (p *Plugin) AddFBPeopleContacts(r *fastglue.Request) error {
	requestDB, organizationID, userID, ok := p.requestContext(r)
	if !ok {
		return nil
	}
	if !p.app.HasPermission(userID, models.ResourceContacts, models.ActionWrite, organizationID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You do not have permission to create contacts", nil, "")
	}

	var request AddContactsRequest
	if err := r.Decode(&request, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}
	request.ListName = strings.TrimSpace(request.ListName)
	if request.ListName == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "name is required", nil, "name")
	}
	if len(request.Data) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "data cannot be empty", nil, "data")
	}
	if len(request.Data) > 5000 {
		return r.SendErrorEnvelope(fasthttp.StatusRequestEntityTooLarge, "Maximum 5000 contacts allowed per request", nil, "data")
	}

	createdCount := 0
	updatedCount := 0
	err := requestDB.Transaction(func(tx *gorm.DB) error {
		for _, item := range request.Data {
			normalizedPhone := normalizePhoneIdentifier(item.Identifier)
			if normalizedPhone == "" || len(normalizedPhone) < 5 || len(normalizedPhone) > 50 {
				continue
			}
			displayName := strings.TrimSpace(item.Name)
			if displayName == "" {
				displayName = normalizedPhone
			}

			var existing models.Contact
			err := tx.Unscoped().
				Where("organization_id = ? AND phone_number = ?", organizationID, normalizedPhone).
				First(&existing).Error
			if err == nil {
				updates := map[string]any{"deleted_at": nil}
				if existing.ProfileName == "" {
					updates["profile_name"] = displayName
				}
				tags, hasTag := contactTags(existing.Tags, request.ListName)
				if !hasTag {
					updates["tags"] = append(tags, request.ListName)
				}
				if err := tx.Model(&existing).Updates(updates).Error; err != nil {
					return err
				}
				updatedCount++
				continue
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			contact := models.Contact{
				BaseModel:      models.BaseModel{ID: uuid.New()},
				OrganizationID: organizationID,
				PhoneNumber:    normalizedPhone,
				ProfileName:    displayName,
				Status:         models.ChatStatusPending,
				Tags:           models.JSONBArray{request.ListName},
				IsRead:         true,
			}
			if err := tx.Create(&contact).Error; err != nil {
				return err
			}
			createdCount++
		}
		return nil
	})
	if err != nil {
		p.app.Log.Error("Failed to import Facebook people contacts", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to import contacts", nil, "")
	}

	return r.SendEnvelope(map[string]interface{}{
		"success": true,
		"created": createdCount,
		"updated": updatedCount,
		"total":   createdCount + updatedCount,
	})
}

func (p *Plugin) ListFBPeopleCampaigns(r *fastglue.Request) error {
	requestDB, organizationID, userID, ok := p.requestContext(r)
	if !ok {
		return nil
	}
	if !p.app.HasPermission(userID, models.ResourceAccounts, models.ActionRead, organizationID) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Insufficient permissions", nil, "")
	}

	var campaigns []string
	if err := requestDB.Model(&PeopleSearch{}).
		Where("organization_id = ?", organizationID).
		Distinct("campaign_id").
		Pluck("campaign_id", &campaigns).Error; err != nil {
		p.app.Log.Error("Failed to list unique people search campaigns", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list campaigns", nil, "")
	}
	return r.SendEnvelope(map[string]interface{}{
		"success":   true,
		"campaigns": campaigns,
	})
}

func normalizePhoneIdentifier(identifier string) string {
	var cleaned []rune
	for _, character := range strings.TrimSpace(identifier) {
		if (character >= '0' && character <= '9') || character == '+' {
			cleaned = append(cleaned, character)
		}
	}
	normalized := string(cleaned)
	if strings.HasPrefix(normalized, "+") {
		normalized = normalized[1:]
	}
	return normalized
}

func contactTags(existing models.JSONBArray, listName string) (models.JSONBArray, bool) {
	tags := make(models.JSONBArray, 0, len(existing))
	hasTag := false
	for _, tag := range existing {
		value, ok := tag.(string)
		if !ok {
			continue
		}
		tags = append(tags, value)
		if strings.EqualFold(value, listName) {
			hasTag = true
		}
	}
	return tags, hasTag
}
