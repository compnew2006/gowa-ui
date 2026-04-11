package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/pkg/whatsmeow"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	waTypes "go.mau.fi/whatsmeow/types"
	"gorm.io/gorm"
)

type sendStatusRequest struct {
	Type           string  `json:"type"`
	Text           string  `json:"text"`
	Caption        string  `json:"caption"`
	MediaURL       string  `json:"media_url"`
	TextARGB       *uint32 `json:"text_argb"`
	BackgroundARGB *uint32 `json:"background_argb"`
	Font           string  `json:"font"`
}

type replyStatusRequest struct {
	Text string `json:"text"`
}

type statusItemResponse struct {
	ID                string  `json:"id"`
	InstanceID        string  `json:"instance_id"`
	InstanceName      string  `json:"instance_name"`
	SenderJID         string  `json:"sender_jid"`
	SenderName        string  `json:"sender_name"`
	WhatsAppMessageID string  `json:"whatsapp_message_id"`
	StatusType        string  `json:"status_type"`
	Content           string  `json:"content"`
	MediaURL          string  `json:"media_url"`
	MediaMimeType     string  `json:"media_mime_type"`
	MediaFilename     string  `json:"media_filename"`
	TextARGB          *int64  `json:"text_argb,omitempty"`
	BackgroundARGB    *int64  `json:"background_argb,omitempty"`
	Font              string  `json:"font"`
	IsSelf            bool    `json:"is_self"`
	SeenAt            *string `json:"seen_at,omitempty"`
	CreatedAt         string  `json:"created_at"`
	ExpiresAt         string  `json:"expires_at"`
}

type statusGroupResponse struct {
	GroupID      string               `json:"group_id"`
	InstanceID   string               `json:"instance_id"`
	InstanceName string               `json:"instance_name"`
	SenderJID    string               `json:"sender_jid"`
	SenderName   string               `json:"sender_name"`
	IsSelf       bool                 `json:"is_self"`
	Statuses     []statusItemResponse `json:"statuses"`
}

// ListStatuses returns active WhatsApp statuses grouped by sender per instance.
func (a *App) ListStatuses(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceChat, models.ActionRead); err != nil {
		if err == errEnvelopeSent {
			return nil
		}
		return err
	}

	instanceFilter := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("instance_id")))
	now := time.Now().UTC()

	query := requestDB.
		Where("organization_id = ? AND expires_at > ? AND status_type IN ?", orgID, now, []string{
			string(models.WhatsAppStatusTypeText),
			string(models.WhatsAppStatusTypeImage),
			string(models.WhatsAppStatusTypeVideo),
		}).
		Order("created_at ASC")
	if instanceFilter != "" {
		query = query.Where("instance_id = ?", instanceFilter)
	}

	var statuses []models.WhatsAppStatus
	if err := query.Find(&statuses).Error; err != nil {
		a.Log.Error("Failed to list statuses", "error", err, "organization_id", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list statuses", nil, "")
	}

	instanceMap, err := a.loadStatusInstanceMap(orgID)
	if err != nil {
		a.Log.Error("Failed to load instances for statuses", "error", err, "organization_id", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list statuses", nil, "")
	}

	groups := buildStatusGroups(statuses, instanceMap)
	return r.SendEnvelope(map[string]any{
		"groups": groups,
		"total":  len(statuses),
	})
}

// SendStatus publishes a status update (text, image, or video) for an instance.
func (a *App) SendStatus(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceChat, models.ActionWrite); err != nil {
		if err == errEnvelopeSent {
			return nil
		}
		return err
	}
	if !a.isWhatsmeowProvider() || a.WhatsmeowManager == nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "WhatsApp Status is available only with Whatsmeow provider", nil, "")
	}

	instanceID, err := parsePathUUID(r, "id", "instance")
	if err != nil {
		if err == errEnvelopeSent {
			return nil
		}
		return err
	}

	var instance models.WhatsAppInstance
	if err := requestDB.Where("id = ? AND organization_id = ?", instanceID, orgID).First(&instance).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Instance not found", nil, "")
	}

	var req sendStatusRequest
	contentType := strings.ToLower(strings.TrimSpace(string(r.RequestCtx.Request.Header.ContentType())))
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := a.parseMultipartStatusRequest(r, &req); err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
		}
	} else {
		if err := json.Unmarshal(r.RequestCtx.PostBody(), &req); err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
		}
	}

	statusType := strings.ToLower(strings.TrimSpace(req.Type))
	if statusType == "" {
		statusType = "text"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	var (
		waMessageID string
		mediaURL    string
		mediaMime   string
		mediaName   string
	)

	switch statusType {
	case "text":
		style, styleErr := parseStatusTextStyle(req)
		if styleErr != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, styleErr.Error(), nil, "font")
		}
		waMessageID, err = a.WhatsmeowManager.SendTextStatus(ctx, instanceID, req.Text, style)
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
		}
	case "image", "video":
		if a.MessageProvider == nil {
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Message provider is not configured", nil, "")
		}
		mediaURL = strings.TrimSpace(req.MediaURL)
		mediaMime = detectStatusMediaMimeType(mediaURL)
		mediaName = fileNameFromPath(mediaURL)
		if mediaURL == "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "media_url is required for media status", nil, "media_url")
		}
		if statusType == "image" {
			waMessageID, err = a.MessageProvider.SendImage(ctx, instanceID.String(), "status@broadcast", mediaURL, strings.TrimSpace(req.Caption))
		} else {
			waMessageID, err = a.MessageProvider.SendVideo(ctx, instanceID.String(), "status@broadcast", mediaURL, strings.TrimSpace(req.Caption))
		}
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
		}
	default:
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "type must be one of: text, image, video", nil, "type")
	}

	createdAt := time.Now().UTC()
	status := models.WhatsAppStatus{
		BaseModel: models.BaseModel{
			ID:        uuid.New(),
			CreatedAt: createdAt,
			UpdatedAt: createdAt,
		},
		OrganizationID:    orgID,
		InstanceID:        instance.ID,
		WhatsAppAccount:   resolveInstanceStatusAccount(instance),
		SenderJID:         resolveInstanceSenderJID(instance),
		SenderName:        strings.TrimSpace(instance.Name),
		WhatsAppMessageID: strings.TrimSpace(waMessageID),
		StatusType:        models.WhatsAppStatusType(statusType),
		Content:           strings.TrimSpace(firstNonEmptyStatusText(req.Text, req.Caption)),
		MediaURL:          mediaURL,
		MediaMimeType:     mediaMime,
		MediaFilename:     mediaName,
		ExpiresAt:         createdAt.Add(24 * time.Hour),
		Metadata: models.JSONB{
			"from_me":     true,
			"source":      "api",
			"instance_id": instance.ID.String(),
		},
	}
	if req.TextARGB != nil {
		value := int64(*req.TextARGB)
		status.TextARGB = &value
	}
	if req.BackgroundARGB != nil {
		value := int64(*req.BackgroundARGB)
		status.BackgroundARGB = &value
	}
	font, _ := parseStatusFont(req.Font)
	if font != nil {
		status.Font = font.String()
	}

	if err := requestDB.Create(&status).Error; err != nil {
		a.Log.Error("Failed to persist outgoing status", "error", err, "instance_id", instance.ID, "wa_message_id", waMessageID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Status sent but failed to persist record", nil, "")
	}

	response := buildStatusItemResponse(status, instance.Name)
	return r.SendEnvelope(response)
}

// MarkStatusSeen marks a status as seen and forwards read receipt to WhatsApp.
func (a *App) MarkStatusSeen(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceChat, models.ActionWrite); err != nil {
		if err == errEnvelopeSent {
			return nil
		}
		return err
	}
	if !a.isWhatsmeowProvider() || a.WhatsmeowManager == nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "WhatsApp Status is available only with Whatsmeow provider", nil, "")
	}

	statusID, err := parsePathUUID(r, "id", "status")
	if err != nil {
		if err == errEnvelopeSent {
			return nil
		}
		return err
	}

	var status models.WhatsAppStatus
	if err := requestDB.Where("id = ? AND organization_id = ?", statusID, orgID).First(&status).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Status not found", nil, "")
	}
	if status.ExpiresAt.Before(time.Now().UTC()) {
		return r.SendErrorEnvelope(fasthttp.StatusGone, "Status has expired", nil, "")
	}

	if fromMe, _ := status.Metadata["from_me"].(bool); fromMe {
		return r.SendEnvelope(map[string]any{
			"status": "ignored",
		})
	}

	if status.WhatsAppMessageID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Status has no WhatsApp message ID", nil, "")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := a.WhatsmeowManager.SendStatusReadReceipt(ctx, status.InstanceID, status.SenderJID, status.WhatsAppMessageID); err != nil {
		a.Log.Error("Failed to send status read receipt", "error", err, "status_id", status.ID, "instance_id", status.InstanceID)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Failed to mark status as seen", nil, "")
	}

	seenAt := time.Now().UTC()
	if err := requestDB.Model(&models.WhatsAppStatus{}).Where("id = ?", status.ID).Update("seen_at", seenAt).Error; err != nil {
		a.Log.Warn("Failed to update status seen timestamp", "error", err, "status_id", status.ID)
	}

	return r.SendEnvelope(map[string]any{
		"status":  "ok",
		"seen_at": seenAt.Format(time.RFC3339),
	})
}

// ReplyToStatus sends a direct reply message to the owner of a status.
func (a *App) ReplyToStatus(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceChat, models.ActionWrite); err != nil {
		if err == errEnvelopeSent {
			return nil
		}
		return err
	}
	if !a.isWhatsmeowProvider() || a.MessageProvider == nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Status replies are available only with Whatsmeow provider", nil, "")
	}

	statusID, err := parsePathUUID(r, "id", "status")
	if err != nil {
		if err == errEnvelopeSent {
			return nil
		}
		return err
	}

	var status models.WhatsAppStatus
	if err := requestDB.Where("id = ? AND organization_id = ?", statusID, orgID).First(&status).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Status not found", nil, "")
	}
	if status.ExpiresAt.Before(time.Now().UTC()) {
		return r.SendErrorEnvelope(fasthttp.StatusGone, "Status has expired", nil, "")
	}
	if fromMe, _ := status.Metadata["from_me"].(bool); fromMe {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Cannot reply to your own status", nil, "")
	}

	var req replyStatusRequest
	if err := json.Unmarshal(r.RequestCtx.PostBody(), &req); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid request body", nil, "")
	}
	replyText := strings.TrimSpace(req.Text)
	if replyText == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "text is required", nil, "text")
	}

	contact, err := a.findOrCreateStatusReplyContact(orgID, status)
	if err != nil {
		a.Log.Error("Failed to resolve contact for status reply", "error", err, "status_id", status.ID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to prepare status reply", nil, "")
	}

	accountName := strings.TrimSpace(status.WhatsAppAccount)
	if accountName == "" {
		accountName = strings.TrimSpace(contact.WhatsAppAccount)
	}
	if accountName == "" {
		accountName = status.InstanceID.String()
	}
	msgReq := OutgoingMessageRequest{
		Account: &models.WhatsAppAccount{
			OrganizationID: orgID,
			Name:           accountName,
		},
		Contact:    contact,
		InstanceID: &status.InstanceID,
		Type:       models.MessageTypeText,
		Content:    replyText,
	}

	opts := DefaultSendOptions()
	opts.SentByUserID = &userID
	opts.DispatchWebhook = false
	opts.Async = false

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	message, sendErr := a.SendOutgoingMessage(ctx, msgReq, opts)
	if sendErr != nil {
		if restrictedMessage, reasonCode, ok := asRestrictedSendViolationWithReason(sendErr); ok {
			return r.SendErrorEnvelope(fasthttp.StatusForbidden, restrictedMessage, reasonCodeDetails(reasonCode), "")
		}
		a.Log.Error("Failed to send status reply", "error", sendErr, "status_id", status.ID)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Failed to send status reply", nil, "")
	}

	return r.SendEnvelope(map[string]any{
		"status":     "ok",
		"message_id": message.ID.String(),
	})
}

// ServeStatusMedia serves media files attached to status records.
func (a *App) ServeStatusMedia(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceChat, models.ActionRead); err != nil {
		if err == errEnvelopeSent {
			return nil
		}
		return err
	}

	statusID, err := parsePathUUID(r, "id", "status")
	if err != nil {
		if err == errEnvelopeSent {
			return nil
		}
		return err
	}

	var status models.WhatsAppStatus
	if err := requestDB.Where("id = ? AND organization_id = ?", statusID, orgID).First(&status).Error; err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Status not found", nil, "")
	}

	mediaPath := strings.TrimSpace(status.MediaURL)
	if mediaPath == "" {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "No media found", nil, "")
	}

	return a.serveLocalMediaFile(r, mediaPath, status.MediaMimeType)
}

func (a *App) parseMultipartStatusRequest(r *fastglue.Request, req *sendStatusRequest) error {
	if req == nil {
		return fmt.Errorf("request is required")
	}

	form, err := r.RequestCtx.MultipartForm()
	if err != nil {
		return fmt.Errorf("invalid multipart form")
	}

	req.Type = formValue(form.Value["type"])
	req.Caption = formValue(form.Value["caption"])
	req.Text = formValue(form.Value["text"])
	req.Font = formValue(form.Value["font"])

	if value := strings.TrimSpace(formValue(form.Value["text_argb"])); value != "" {
		parsed, parseErr := strconv.ParseUint(value, 10, 32)
		if parseErr != nil {
			return fmt.Errorf("text_argb must be a uint32 number")
		}
		typed := uint32(parsed)
		req.TextARGB = &typed
	}
	if value := strings.TrimSpace(formValue(form.Value["background_argb"])); value != "" {
		parsed, parseErr := strconv.ParseUint(value, 10, 32)
		if parseErr != nil {
			return fmt.Errorf("background_argb must be a uint32 number")
		}
		typed := uint32(parsed)
		req.BackgroundARGB = &typed
	}

	files := form.File["file"]
	if len(files) == 0 {
		req.MediaURL = strings.TrimSpace(formValue(form.Value["media_url"]))
		return nil
	}

	fileHeader := files[0]
	file, err := fileHeader.Open()
	if err != nil {
		return fmt.Errorf("failed to read uploaded file")
	}
	defer func() { _ = file.Close() }()

	fileData, err := io.ReadAll(io.LimitReader(file, whatsappDocumentMaxBytes+1))
	if err != nil {
		return fmt.Errorf("failed to read file data")
	}

	mimeType := resolveWhatsAppMediaMIME(fileHeader.Header.Get("Content-Type"), fileHeader.Filename, fileData)
	mediaType := deriveWhatsAppMediaMessageType(mimeType)
	maxAllowedBytes := whatsappMediaMaxSizeBytes(mediaType)
	if int64(len(fileData)) > maxAllowedBytes {
		return fmt.Errorf("%s file is too large (max %dMB)", mediaType, whatsappMediaMaxSizeMB(mediaType))
	}
	if mediaType != models.MessageTypeImage && mediaType != models.MessageTypeVideo {
		return fmt.Errorf("status media must be image or video")
	}
	req.Type = string(mediaType)

	localPath, err := a.saveMediaLocally(fileData, mimeType, fileHeader.Filename)
	if err != nil {
		return fmt.Errorf("failed to save uploaded file")
	}
	req.MediaURL = localPath
	return nil
}

func parseStatusTextStyle(req sendStatusRequest) (whatsmeow.StatusTextStyle, error) {
	style := whatsmeow.StatusTextStyle{
		TextARGB:       req.TextARGB,
		BackgroundARGB: req.BackgroundARGB,
	}

	font, err := parseStatusFont(req.Font)
	if err != nil {
		return style, err
	}
	style.Font = font
	return style, nil
}

func parseStatusFont(raw string) (*waE2E.ExtendedTextMessage_FontType, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	if numeric, err := strconv.ParseInt(trimmed, 10, 32); err == nil {
		font := waE2E.ExtendedTextMessage_FontType(numeric)
		return &font, nil
	}

	normalized := strings.ToUpper(trimmed)
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, " ", "_")
	if value, ok := waE2E.ExtendedTextMessage_FontType_value[normalized]; ok {
		font := waE2E.ExtendedTextMessage_FontType(value)
		return &font, nil
	}

	return nil, fmt.Errorf("unsupported font value")
}

func buildStatusGroups(statuses []models.WhatsAppStatus, instanceMap map[uuid.UUID]models.WhatsAppInstance) []statusGroupResponse {
	groupOrder := make([]string, 0)
	groups := make(map[string]*statusGroupResponse)

	for _, status := range statuses {
		instance := instanceMap[status.InstanceID]
		isSelf := strings.TrimSpace(instance.JID) != "" && strings.EqualFold(strings.TrimSpace(instance.JID), strings.TrimSpace(status.SenderJID))
		if !isSelf {
			instancePhone := resolveInstanceSenderJID(instance)
			isSelf = strings.EqualFold(strings.TrimSpace(instancePhone), strings.TrimSpace(status.SenderJID))
		}

		groupKey := status.InstanceID.String() + "|" + strings.ToLower(strings.TrimSpace(status.SenderJID))
		group := groups[groupKey]
		if group == nil {
			group = &statusGroupResponse{
				GroupID:      groupKey,
				InstanceID:   status.InstanceID.String(),
				InstanceName: strings.TrimSpace(instance.Name),
				SenderJID:    status.SenderJID,
				SenderName:   displayStatusSenderName(status),
				IsSelf:       isSelf,
				Statuses:     make([]statusItemResponse, 0),
			}
			groups[groupKey] = group
			groupOrder = append(groupOrder, groupKey)
		}

		group.Statuses = append(group.Statuses, buildStatusItemResponse(status, instance.Name))
		if group.SenderName == "" {
			group.SenderName = displayStatusSenderName(status)
		}
	}

	response := make([]statusGroupResponse, 0, len(groupOrder))
	for _, key := range groupOrder {
		group := groups[key]
		sort.Slice(group.Statuses, func(i, j int) bool {
			return group.Statuses[i].CreatedAt < group.Statuses[j].CreatedAt
		})
		response = append(response, *group)
	}

	sort.SliceStable(response, func(i, j int) bool {
		leftStatuses := response[i].Statuses
		rightStatuses := response[j].Statuses
		if len(leftStatuses) == 0 || len(rightStatuses) == 0 {
			return response[i].SenderName < response[j].SenderName
		}
		return leftStatuses[len(leftStatuses)-1].CreatedAt > rightStatuses[len(rightStatuses)-1].CreatedAt
	})

	return response
}

func buildStatusItemResponse(status models.WhatsAppStatus, instanceName string) statusItemResponse {
	resp := statusItemResponse{
		ID:                status.ID.String(),
		InstanceID:        status.InstanceID.String(),
		InstanceName:      strings.TrimSpace(instanceName),
		SenderJID:         status.SenderJID,
		SenderName:        displayStatusSenderName(status),
		WhatsAppMessageID: status.WhatsAppMessageID,
		StatusType:        string(status.StatusType),
		Content:           status.Content,
		MediaURL:          resolveStatusMediaURL(status),
		MediaMimeType:     status.MediaMimeType,
		MediaFilename:     status.MediaFilename,
		TextARGB:          status.TextARGB,
		BackgroundARGB:    status.BackgroundARGB,
		Font:              status.Font,
		CreatedAt:         status.CreatedAt.UTC().Format(time.RFC3339),
		ExpiresAt:         status.ExpiresAt.UTC().Format(time.RFC3339),
	}
	if fromMe, ok := status.Metadata["from_me"].(bool); ok {
		resp.IsSelf = fromMe
	}
	if status.SeenAt != nil {
		seen := status.SeenAt.UTC().Format(time.RFC3339)
		resp.SeenAt = &seen
	}
	return resp
}

func resolveStatusMediaURL(status models.WhatsAppStatus) string {
	mediaURL := strings.TrimSpace(status.MediaURL)
	if mediaURL == "" {
		return ""
	}

	lowerMediaURL := strings.ToLower(mediaURL)
	if strings.HasPrefix(lowerMediaURL, "http://") ||
		strings.HasPrefix(lowerMediaURL, "https://") ||
		strings.HasPrefix(lowerMediaURL, "data:") {
		return mediaURL
	}

	if strings.HasPrefix(mediaURL, "/api/statuses/") {
		return mediaURL
	}

	return "/api/statuses/" + status.ID.String() + "/media"
}

func (a *App) findOrCreateStatusReplyContact(orgID uuid.UUID, status models.WhatsAppStatus) (*models.Contact, error) {
	primaryPhone := resolveStatusReplyContactPhone(status.SenderJID)
	if primaryPhone == "" {
		return nil, fmt.Errorf("status sender jid is empty")
	}

	candidates := []string{primaryPhone}
	senderJID := strings.TrimSpace(status.SenderJID)
	if senderJID != "" && senderJID != primaryPhone {
		candidates = append(candidates, senderJID)
	}

	var contact models.Contact
	err := a.DB.
		Where("organization_id = ? AND instance_id = ? AND phone_number IN ?", orgID, status.InstanceID, candidates).
		Order("created_at DESC").
		First(&contact).Error
	if err == nil {
		return &contact, nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	profileName := strings.TrimSpace(status.SenderName)
	if profileName == "" {
		profileName = primaryPhone
	}
	instanceID := status.InstanceID
	contact = models.Contact{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  orgID,
		InstanceID:      &instanceID,
		PhoneNumber:     primaryPhone,
		ProfileName:     profileName,
		WhatsAppAccount: strings.TrimSpace(status.WhatsAppAccount),
		Metadata:        models.JSONB{},
	}
	if err := a.DB.Create(&contact).Error; err != nil {
		return nil, err
	}

	return &contact, nil
}

func resolveStatusReplyContactPhone(senderJID string) string {
	trimmed := strings.TrimSpace(senderJID)
	if trimmed == "" {
		return ""
	}

	parsed, err := waTypes.ParseJID(trimmed)
	if err != nil {
		return trimmed
	}

	nonAD := parsed.ToNonAD()
	if nonAD.Server == waTypes.DefaultUserServer && nonAD.User != "" {
		return nonAD.User
	}
	return nonAD.String()
}

func (a *App) loadStatusInstanceMap(orgID uuid.UUID) (map[uuid.UUID]models.WhatsAppInstance, error) {
	var instances []models.WhatsAppInstance
	if err := a.DB.Where("organization_id = ?", orgID).Find(&instances).Error; err != nil {
		return nil, err
	}
	result := make(map[uuid.UUID]models.WhatsAppInstance, len(instances))
	for _, instance := range instances {
		result[instance.ID] = instance
	}
	return result, nil
}

func resolveInstanceSenderJID(instance models.WhatsAppInstance) string {
	if strings.TrimSpace(instance.JID) != "" {
		return strings.TrimSpace(instance.JID)
	}

	phone := strings.TrimSpace(instance.PhoneNumber)
	if phone == "" {
		return ""
	}
	if strings.Contains(phone, "@") {
		return phone
	}
	return phone + "@s.whatsapp.net"
}

func resolveInstanceStatusAccount(instance models.WhatsAppInstance) string {
	phone := strings.TrimSpace(instance.PhoneNumber)
	if phone != "" {
		return phone
	}
	return instance.ID.String()
}

func displayStatusSenderName(status models.WhatsAppStatus) string {
	if trimmed := strings.TrimSpace(status.SenderName); trimmed != "" {
		return trimmed
	}
	jid := strings.TrimSpace(status.SenderJID)
	if jid == "" {
		return "Unknown"
	}
	if idx := strings.Index(jid, "@"); idx > 0 {
		return jid[:idx]
	}
	return jid
}

func detectStatusMediaMimeType(mediaURL string) string {
	lower := strings.ToLower(strings.TrimSpace(mediaURL))
	switch {
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(lower, ".gif"):
		return "image/gif"
	case strings.HasSuffix(lower, ".webp"):
		return "image/webp"
	case strings.HasSuffix(lower, ".mov"):
		return "video/quicktime"
	case strings.HasSuffix(lower, ".mkv"):
		return "video/x-matroska"
	case strings.HasSuffix(lower, ".webm"):
		return "video/webm"
	default:
		if strings.HasSuffix(lower, ".mp4") {
			return "video/mp4"
		}
	}
	return ""
}

func fileNameFromPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	idx := strings.LastIndex(path, "/")
	if idx >= 0 && idx+1 < len(path) {
		return path[idx+1:]
	}
	return path
}

func formValue(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[0])
}

func firstNonEmptyStatusText(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
