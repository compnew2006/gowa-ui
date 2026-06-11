package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	appcrypto "github.com/compnew2006/whatomate/internal/crypto"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/websocket"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	defaultFBCommentPostLimit   = 25
	defaultFBCommentsPerPost    = 50
	maxFBCommentPostLimit       = 100
	maxFBCommentsPerPost        = 100
	defaultFBCommentReplyText   = "تم الرد خاص"
	defaultFBPrivateMessageText = "اهلا كيف اقدر اساعدك"
	fbCommentReplyStatusSent    = "sent"
	fbCommentReplyStatusPartial = "partial"
	fbCommentReplyStatusFailed  = "failed"
	fbCommentReplyStatusSkipped = "skipped"
)

type facebookCommentSettingsRequest struct {
	Enabled                    *bool   `json:"enabled"`
	SyncEnabled                *bool   `json:"sync_enabled"`
	AutoReplyEnabled           *bool   `json:"auto_reply_enabled"`
	AutoCommentReplyEnabled    *bool   `json:"auto_comment_reply_enabled"`
	AutoPrivateReplyEnabled    *bool   `json:"auto_private_reply_enabled"`
	AutoCommentReplyText       *string `json:"auto_comment_reply_text"`
	AutoPrivateMessageText     *string `json:"auto_private_message_text"`
	OnlyAutoReplyUnanswered    *bool   `json:"only_auto_reply_unanswered"`
	IgnorePageAdminComments    *bool   `json:"ignore_page_admin_comments"`
	DefaultSyncPostLimit       *int    `json:"default_sync_post_limit"`
	DefaultSyncCommentsPerPost *int    `json:"default_sync_comments_per_post"`
}

type facebookCommentSyncRequest struct {
	AccountID       string   `json:"account_id"`
	PageID          string   `json:"page_id"`
	PostLimit       int      `json:"post_limit"`
	CommentsPerPost int      `json:"comments_per_post"`
	PostIDs         []string `json:"post_ids"`
	RunAutoReply    *bool    `json:"run_auto_reply"`
}

type facebookCommentReplyRequest struct {
	ReplyText          string `json:"reply_text"`
	PrivateMessageText string `json:"private_message_text"`
	SendCommentReply   *bool  `json:"send_comment_reply"`
	SendPrivateMessage *bool  `json:"send_private_message"`
}

type facebookCommentStatusRequest struct {
	Status models.FacebookCommentStatus `json:"status"`
}

type facebookCommentsWebhookPayload struct {
	Object string                         `json:"object"`
	Entry  []facebookCommentsWebhookEntry `json:"entry"`
}

type facebookCommentsWebhookEntry struct {
	ID      string                          `json:"id"`
	Time    int64                           `json:"time"`
	Changes []facebookCommentsWebhookChange `json:"changes"`
}

type facebookCommentsWebhookChange struct {
	Field string                       `json:"field"`
	Value facebookCommentsWebhookValue `json:"value"`
}

type facebookCommentsWebhookActor struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type facebookCommentsWebhookValue struct {
	Item        string                        `json:"item"`
	Verb        string                        `json:"verb"`
	CommentID   string                        `json:"comment_id"`
	PostID      string                        `json:"post_id"`
	ParentID    string                        `json:"parent_id"`
	SenderID    string                        `json:"sender_id"`
	SenderName  string                        `json:"sender_name"`
	From        facebookCommentsWebhookActor  `json:"from"`
	Message     string                        `json:"message"`
	CreatedTime int64                         `json:"created_time"`
	Permalink   string                        `json:"permalink_url"`
}

func (v facebookCommentsWebhookValue) commenterID() string {
	if v.From.ID != "" {
		return v.From.ID
	}
	return v.SenderID
}

func (v facebookCommentsWebhookValue) commenterName() string {
	if v.From.Name != "" {
		return v.From.Name
	}
	return v.SenderName
}

func isFacebookPageAdminCommenter(pageID, commenterID string) bool {
	pageID = strings.TrimSpace(pageID)
	commenterID = strings.TrimSpace(commenterID)
	if pageID == "" || commenterID == "" {
		return false
	}
	return pageID == commenterID
}

type facebookCommentListResponse struct {
	Comments []facebookCommentResponse `json:"comments"`
	Total    int64                     `json:"total"`
	Page     int                       `json:"page"`
	Limit    int                       `json:"limit"`
}

type facebookCommentResponse struct {
	ID            uuid.UUID                       `json:"id"`
	AccountID     uuid.UUID                       `json:"account_id"`
	PageID        string                          `json:"page_id"`
	PageName      string                          `json:"page_name"`
	PostID        string                          `json:"post_id"`
	PostPermalink string                          `json:"post_permalink"`
	PostMessage   string                          `json:"post_message"`
	ExternalID    string                          `json:"external_id"`
	ParentID      string                          `json:"parent_id"`
	FromID        string                          `json:"from_id"`
	FromName      string                          `json:"from_name"`
	Message       string                          `json:"message"`
	Permalink     string                          `json:"permalink"`
	Status        models.FacebookCommentStatus    `json:"status"`
	Direction     models.FacebookCommentDirection `json:"direction"`
	IsAdminReply  bool                            `json:"is_admin_reply"`
	CommentedAt   string                          `json:"commented_at"`
	LastSyncedAt  string                          `json:"last_synced_at,omitempty"`
	LastRepliedAt string                          `json:"last_replied_at,omitempty"`
	AutoRepliedAt string                          `json:"auto_replied_at,omitempty"`
	Metadata      models.JSONB                    `json:"metadata"`
	Replies       []facebookCommentReplyResponse  `json:"replies"`
}

type facebookCommentReplyResponse struct {
	ID                  uuid.UUID `json:"id"`
	UserID              uuid.UUID `json:"user_id"`
	ReplyText           string    `json:"reply_text"`
	PrivateMessageText  string    `json:"private_message_text"`
	GraphCommentReplyID string    `json:"graph_comment_reply_id"`
	GraphPrivateReplyID string    `json:"graph_private_reply_id"`
	Status              string    `json:"status"`
	ErrorMessage        string    `json:"error_message,omitempty"`
	IsAuto              bool      `json:"is_auto"`
	CreatedAt           string    `json:"created_at"`
}

type facebookPostEdgeResponse struct {
	Data []facebookPostEdge `json:"data"`
}

type facebookPostEdge struct {
	ID           string                      `json:"id"`
	Message      string                      `json:"message"`
	PermalinkURL string                      `json:"permalink_url"`
	CreatedTime  string                      `json:"created_time"`
	Comments     facebookCommentEdgeResponse `json:"comments"`
}

type facebookCommentEdgeResponse struct {
	Data []facebookCommentEdge `json:"data"`
}

type facebookCommentEdge struct {
	ID           string                    `json:"id"`
	Message      string                    `json:"message"`
	CreatedTime  string                    `json:"created_time"`
	PermalinkURL string                    `json:"permalink_url"`
	From         facebookCommentActor      `json:"from"`
	Parent       *facebookCommentParentRef `json:"parent"`
	CommentCount int                       `json:"comment_count"`
}

type facebookCommentDetailResponse struct {
	ID   string               `json:"id"`
	From facebookCommentActor `json:"from"`
}

type facebookGraphBatchResponse struct {
	Code int    `json:"code"`
	Body string `json:"body"`
}

type facebookCommentActor struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type facebookCommentParentRef struct {
	ID string `json:"id"`
}

func (a *App) GetFacebookCommentSettings(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceAccounts, models.ActionRead); err != nil {
		return nil
	}

	settings, err := a.getOrCreateFacebookCommentSettings(requestDB, orgID)
	if err != nil {
		a.Log.Error("Failed to load Facebook comment settings", "error", err, "organization_id", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load Facebook comment settings", nil, "")
	}
	return r.SendEnvelope(map[string]any{"settings": settings})
}

func (a *App) UpdateFacebookCommentSettings(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceAccounts, models.ActionWrite); err != nil {
		return nil
	}

	var req facebookCommentSettingsRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}
	settings, err := a.getOrCreateFacebookCommentSettings(requestDB, orgID)
	if err != nil {
		a.Log.Error("Failed to load Facebook comment settings for update", "error", err, "organization_id", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update Facebook comment settings", nil, "")
	}

	applyFacebookCommentSettingsRequest(settings, req)
	if err := requestDB.Save(settings).Error; err != nil {
		a.Log.Error("Failed to save Facebook comment settings", "error", err, "organization_id", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update Facebook comment settings", nil, "")
	}
	return r.SendEnvelope(map[string]any{"settings": settings})
}

func (a *App) ListFacebookComments(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceAccounts, models.ActionRead); err != nil {
		return nil
	}

	page := queryInt(r, "page", 1, 1, 100000)
	limit := queryInt(r, "limit", 30, 1, 100)
	status := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("status")))
	accountID := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("account_id")))
	pageID := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("page_id")))
	search := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("search")))

	query := requestDB.Model(&models.FacebookComment{}).Where("organization_id = ?", orgID)
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}
	if accountID != "" {
		if parsed, err := uuid.Parse(accountID); err == nil {
			query = query.Where("account_id = ?", parsed)
		}
	}
	if pageID != "" && pageID != "all" {
		query = query.Where("page_id = ?", pageID)
	}
	if search != "" {
		like := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(message) LIKE ? OR LOWER(from_name) LIKE ? OR LOWER(post_message) LIKE ?", like, like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		a.Log.Error("Failed to count Facebook comments", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list Facebook comments", nil, "")
	}

	var comments []models.FacebookComment
	if err := query.Preload("Replies", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at ASC")
	}).Order("commented_at DESC").Limit(limit).Offset((page - 1) * limit).Find(&comments).Error; err != nil {
		a.Log.Error("Failed to list Facebook comments", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to list Facebook comments", nil, "")
	}

	response := make([]facebookCommentResponse, len(comments))
	for i, comment := range comments {
		response[i] = facebookCommentToResponse(comment)
	}
	return r.SendEnvelope(facebookCommentListResponse{
		Comments: response,
		Total:    total,
		Page:     page,
		Limit:    limit,
	})
}

func (a *App) SyncFacebookComments(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceAccounts, models.ActionWrite); err != nil {
		return nil
	}

	var req facebookCommentSyncRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}
	settings, err := a.getOrCreateFacebookCommentSettings(requestDB, orgID)
	if err != nil {
		a.Log.Error("Failed to load Facebook comment settings for sync", "error", err, "organization_id", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to sync Facebook comments", nil, "")
	}
	if !settings.Enabled || !settings.SyncEnabled {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Facebook comment sync is disabled", nil, "")
	}

	accounts, err := a.facebookAccountsForCommentSync(requestDB, orgID, req.AccountID)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, "")
	}
	postLimit := clampPositive(req.PostLimit, settings.DefaultSyncPostLimit, maxFBCommentPostLimit)
	commentsPerPost := clampPositive(req.CommentsPerPost, settings.DefaultSyncCommentsPerPost, maxFBCommentsPerPost)
	runAutoReply := settings.AutoReplyEnabled
	if req.RunAutoReply != nil {
		runAutoReply = *req.RunAutoReply
	}

	var synced, created, autoReplies int
	var failures []string
	for _, account := range accounts {
		pageTokens, err := a.facebookPageTokens(&account)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: failed to decrypt page tokens", account.Name))
			continue
		}
		pageNames := facebookAccountPageNames(account)
		for pageID, pageToken := range pageTokens {
			if req.PageID != "" && req.PageID != "all" && req.PageID != pageID {
				continue
			}
			pageName := pageNames[pageID]
			result := a.syncFacebookPageComments(requestDB, orgID, account, pageID, pageName, pageToken, req.PostIDs, postLimit, commentsPerPost, settings, runAutoReply, userID)
			synced += result.Synced
			created += result.Created
			autoReplies += result.AutoReplies
			failures = append(failures, result.Failures...)
		}
	}

	return r.SendEnvelope(map[string]any{
		"synced":       synced,
		"created":      created,
		"auto_replies": autoReplies,
		"failures":     failures,
	})
}

func (a *App) ReplyFacebookComment(r *fastglue.Request) error {
	requestDB := a.requestDB(r)
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceAccounts, models.ActionWrite); err != nil {
		return nil
	}
	commentRef := strings.TrimSpace(fmt.Sprint(r.RequestCtx.UserValue("id")))
	if commentRef == "" || commentRef == "<nil>" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid Facebook comment ID", nil, "")
	}

	var req facebookCommentReplyRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}
	sendCommentReply := true
	sendPrivateMessage := false
	if req.SendCommentReply != nil {
		sendCommentReply = *req.SendCommentReply
	}
	if req.SendPrivateMessage != nil {
		sendPrivateMessage = *req.SendPrivateMessage
	}
	req.ReplyText = strings.TrimSpace(req.ReplyText)
	req.PrivateMessageText = strings.TrimSpace(req.PrivateMessageText)
	if sendCommentReply && req.ReplyText == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "reply_text is required", nil, "reply_text")
	}
	if sendPrivateMessage && req.PrivateMessageText == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "private_message_text is required", nil, "private_message_text")
	}
	if !sendCommentReply && !sendPrivateMessage {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Select at least one reply channel", nil, "")
	}

	comment, account, pageToken, err := a.facebookCommentOperationContext(a.DB, orgID, commentRef)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Facebook comment not found", nil, "")
		}
		a.Log.Error("Failed to load Facebook comment for reply", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load Facebook comment", nil, "")
	}

	reply, err := a.sendAndStoreFacebookCommentReply(requestDB, account, comment, userID, req.ReplyText, req.PrivateMessageText, sendCommentReply, sendPrivateMessage, false, pageToken)
	if err != nil {
		a.Log.Error("Failed to store Facebook comment reply", "error", err, "comment_id", comment.ID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to save Facebook comment reply", nil, "")
	}
	return r.SendEnvelope(map[string]any{"reply": facebookCommentReplyToResponse(reply)})
}

func (a *App) UpdateFacebookCommentStatus(r *fastglue.Request) error {
	orgID, userID, err := a.getOrgAndUserID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Unauthorized", nil, "")
	}
	if err := a.requirePermission(r, userID, models.ResourceAccounts, models.ActionWrite); err != nil {
		return nil
	}
	commentRef := strings.TrimSpace(fmt.Sprint(r.RequestCtx.UserValue("id")))
	if commentRef == "" || commentRef == "<nil>" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid Facebook comment ID", nil, "")
	}
	var req facebookCommentStatusRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}
	if !validFacebookCommentStatus(req.Status) {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid comment status", nil, "status")
	}
	commentQuery := a.DB.Model(&models.FacebookComment{}).Where("organization_id = ?", orgID)
	if commentID, err := uuid.Parse(commentRef); err == nil {
		commentQuery = commentQuery.Where("id = ? OR external_id = ?", commentID, commentRef)
	} else {
		commentQuery = commentQuery.Where("external_id = ?", commentRef)
	}
	if err := commentQuery.Update("status", req.Status).Error; err != nil {
		a.Log.Error("Failed to update Facebook comment status", "error", err, "comment_id", commentRef)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update Facebook comment", nil, "")
	}
	if a.WSHub != nil {
		updated := models.FacebookComment{}
		_ = commentQuery.First(&updated).Error
		a.WSHub.BroadcastToOrg(orgID, websocket.WSMessage{
			Type:    websocket.TypeFacebookCommentUpdated,
			Payload: facebookCommentToResponse(updated),
		})
	}
	return r.SendEnvelope(map[string]any{"status": req.Status})
}

func (a *App) VerifyFacebookCommentsWebhook(r *fastglue.Request) error {
	verifyToken := ""
	if a.Config != nil {
		verifyToken = strings.TrimSpace(a.Config.FacebookOAuth.WebhookVerifyToken)
	}
	mode := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("hub.mode")))
	token := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("hub.verify_token")))
	challenge := string(r.RequestCtx.QueryArgs().Peek("hub.challenge"))
	if verifyToken == "" || mode != "subscribe" || token != verifyToken {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Invalid Facebook webhook verification token", nil, "")
	}
	r.RequestCtx.SetStatusCode(fasthttp.StatusOK)
	r.RequestCtx.SetContentType("text/plain; charset=utf-8")
	r.RequestCtx.SetBodyString(challenge)
	return nil
}

func (a *App) ReceiveFacebookCommentsWebhook(r *fastglue.Request) error {
	body := r.RequestCtx.PostBody()
	if err := a.verifyFacebookWebhookSignature(r, body); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Invalid Facebook webhook signature", nil, "")
	}
	var payload facebookCommentsWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid Facebook webhook payload", nil, "")
	}
	if payload.Object != "page" {
		return r.SendEnvelope(map[string]string{"status": "ignored"})
	}

	var processed, autoReplies int
	var failures []string
	for _, entry := range payload.Entry {
		pageID := strings.TrimSpace(entry.ID)
		if pageID == "" {
			continue
		}
		account, pageToken, err := a.findFacebookAccountByPageID(pageID)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: account not found", pageID))
			continue
		}
		settings, err := a.getOrCreateFacebookCommentSettings(a.DB, account.OrganizationID)
		if err != nil || !settings.Enabled {
			continue
		}
		pageNames := facebookAccountPageNames(*account)
		for _, change := range entry.Changes {
			if change.Field != "feed" || change.Value.Item != "comment" || strings.TrimSpace(change.Value.CommentID) == "" {
				continue
			}
			comment, created, err := a.upsertFacebookWebhookComment(a.DB, account, pageID, pageNames[pageID], change.Value)
			if err != nil {
				a.Log.Error("Failed to save Facebook webhook comment", "error", err, "page_id", pageID, "comment_id", change.Value.CommentID)
				failures = append(failures, fmt.Sprintf("%s: failed to save webhook comment", pageID))
				continue
			}
			if a.WSHub != nil {
				wsType := websocket.TypeFacebookCommentCreated
				if !created {
					wsType = websocket.TypeFacebookCommentUpdated
				}
				a.WSHub.BroadcastToOrg(account.OrganizationID, websocket.WSMessage{
					Type:    wsType,
					Payload: facebookCommentToResponse(*comment),
				})
			}
			processed++
			if created && !comment.IsAdminReply && settings.AutoReplyEnabled && shouldAutoReplyFacebookComment(a.DB, settings, *comment) {
				if _, err := a.sendAndStoreFacebookCommentReply(a.DB, account, comment, account.UserID, settings.AutoCommentReplyText, settings.AutoPrivateMessageText, settings.AutoCommentReplyEnabled, settings.AutoPrivateReplyEnabled, true, pageToken); err == nil {
					autoReplies++
				} else {
					failures = append(failures, fmt.Sprintf("%s: auto reply failed", change.Value.CommentID))
				}
			}
		}
	}
	return r.SendEnvelope(map[string]any{
		"status":       "ok",
		"processed":    processed,
		"auto_replies": autoReplies,
		"failures":     failures,
	})
}

func (a *App) verifyFacebookWebhookSignature(r *fastglue.Request, body []byte) error {
	appSecret := ""
	if a.Config != nil {
		appSecret = strings.TrimSpace(a.Config.FacebookOAuth.AppSecret)
	}
	if appSecret == "" {
		return nil
	}
	header := strings.TrimSpace(string(r.RequestCtx.Request.Header.Peek("X-Hub-Signature-256")))
	if header == "" {
		return errors.New("missing signature")
	}
	if !strings.HasPrefix(header, "sha256=") {
		return errors.New("unsupported signature")
	}
	provided, err := hex.DecodeString(strings.TrimPrefix(header, "sha256="))
	if err != nil {
		return err
	}
	mac := hmac.New(sha256.New, []byte(appSecret))
	_, _ = mac.Write(body)
	expected := mac.Sum(nil)
	if !hmac.Equal(provided, expected) {
		return errors.New("signature mismatch")
	}
	return nil
}

func (a *App) getOrCreateFacebookCommentSettings(db *gorm.DB, orgID uuid.UUID) (*models.FacebookCommentSettings, error) {
	var settings models.FacebookCommentSettings
	err := db.Where("organization_id = ?", orgID).First(&settings).Error
	if err == nil {
		normalizeFacebookCommentSettings(&settings)
		return &settings, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	settings = models.FacebookCommentSettings{
		OrganizationID:             orgID,
		Enabled:                    true,
		SyncEnabled:                true,
		AutoReplyEnabled:           false,
		AutoCommentReplyEnabled:    true,
		AutoPrivateReplyEnabled:    true,
		AutoCommentReplyText:       defaultFBCommentReplyText,
		AutoPrivateMessageText:     defaultFBPrivateMessageText,
		OnlyAutoReplyUnanswered:    true,
		IgnorePageAdminComments:    true,
		DefaultSyncPostLimit:       defaultFBCommentPostLimit,
		DefaultSyncCommentsPerPost: defaultFBCommentsPerPost,
		Metadata:                   models.JSONB{},
	}
	if err := a.DB.Create(&settings).Error; err != nil {
		return nil, err
	}
	return &settings, nil
}

func (a *App) findFacebookAccountByPageID(pageID string) (*models.FacebookAccount, string, error) {
	var accounts []models.FacebookAccount
	if err := a.DB.Where("method = ? AND status = ?", models.FBAccountMethodOAuth, models.FBAccountStatusActive).Find(&accounts).Error; err != nil {
		return nil, "", err
	}
	for _, account := range accounts {
		pageTokens, err := a.facebookPageTokens(&account)
		if err != nil {
			continue
		}
		if token := strings.TrimSpace(pageTokens[pageID]); token != "" {
			return &account, token, nil
		}
	}
	return nil, "", gorm.ErrRecordNotFound
}

func (a *App) upsertFacebookWebhookComment(db *gorm.DB, account *models.FacebookAccount, pageID, pageName string, value facebookCommentsWebhookValue) (*models.FacebookComment, bool, error) {
	now := time.Now()
	commentedAt := now
	if value.CreatedTime > 0 {
		commentedAt = time.Unix(value.CreatedTime, 0)
	}
	isAdminReply := isFacebookPageAdminCommenter(pageID, value.commenterID())
	comment := models.FacebookComment{
		OrganizationID: account.OrganizationID,
		AccountID:      account.ID,
		PageID:         pageID,
		PageName:       pageName,
		PostID:         value.PostID,
		PostPermalink:  "",
		PostMessage:    "",
		ExternalID:     value.CommentID,
		ParentID:       value.ParentID,
		FromID:         value.commenterID(),
		FromName:       value.commenterName(),
		Message:        value.Message,
		Permalink:      value.Permalink,
		Status:         models.FBCommentStatusOpen,
		Direction:      models.FBCommentDirectionIncoming,
		IsAdminReply:   isAdminReply,
		CommentedAt:    commentedAt,
		LastSyncedAt:   &now,
		Metadata: models.JSONB{
			"source": "facebook_webhook",
			"verb":   value.Verb,
		},
	}
	normalizeFacebookCommentForSave(&comment)
	commentDB := db.Session(&gorm.Session{NewDB: true}).Table((&models.FacebookComment{}).TableName())
	tx := commentDB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "organization_id"}, {Name: "external_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"page_name", "post_id", "parent_id", "from_id", "from_name", "message",
			"permalink", "commented_at", "last_synced_at", "metadata",
			"is_admin_reply", "updated_at",
		}),
	}).Create(&comment)
	if tx.Error != nil {
		return nil, false, tx.Error
	}
	var saved models.FacebookComment
	if err := commentDB.Where("organization_id = ? AND external_id = ?", account.OrganizationID, value.CommentID).First(&saved).Error; err != nil {
		return nil, false, err
	}
	created := saved.CreatedAt.Equal(saved.UpdatedAt) || saved.LastRepliedAt == nil
	return &saved, created, nil
}

func applyFacebookCommentSettingsRequest(settings *models.FacebookCommentSettings, req facebookCommentSettingsRequest) {
	if req.Enabled != nil {
		settings.Enabled = *req.Enabled
	}
	if req.SyncEnabled != nil {
		settings.SyncEnabled = *req.SyncEnabled
	}
	if req.AutoReplyEnabled != nil {
		settings.AutoReplyEnabled = *req.AutoReplyEnabled
	}
	if req.AutoCommentReplyEnabled != nil {
		settings.AutoCommentReplyEnabled = *req.AutoCommentReplyEnabled
	}
	if req.AutoPrivateReplyEnabled != nil {
		settings.AutoPrivateReplyEnabled = *req.AutoPrivateReplyEnabled
	}
	if req.AutoCommentReplyText != nil {
		settings.AutoCommentReplyText = strings.TrimSpace(*req.AutoCommentReplyText)
	}
	if req.AutoPrivateMessageText != nil {
		settings.AutoPrivateMessageText = strings.TrimSpace(*req.AutoPrivateMessageText)
	}
	if req.OnlyAutoReplyUnanswered != nil {
		settings.OnlyAutoReplyUnanswered = *req.OnlyAutoReplyUnanswered
	}
	if req.IgnorePageAdminComments != nil {
		settings.IgnorePageAdminComments = *req.IgnorePageAdminComments
	}
	if req.DefaultSyncPostLimit != nil {
		settings.DefaultSyncPostLimit = clampPositive(*req.DefaultSyncPostLimit, defaultFBCommentPostLimit, maxFBCommentPostLimit)
	}
	if req.DefaultSyncCommentsPerPost != nil {
		settings.DefaultSyncCommentsPerPost = clampPositive(*req.DefaultSyncCommentsPerPost, defaultFBCommentsPerPost, maxFBCommentsPerPost)
	}
	normalizeFacebookCommentSettings(settings)
}

func normalizeFacebookCommentSettings(settings *models.FacebookCommentSettings) {
	if strings.TrimSpace(settings.AutoCommentReplyText) == "" {
		settings.AutoCommentReplyText = defaultFBCommentReplyText
	}
	if strings.TrimSpace(settings.AutoPrivateMessageText) == "" {
		settings.AutoPrivateMessageText = defaultFBPrivateMessageText
	}
	settings.DefaultSyncPostLimit = clampPositive(settings.DefaultSyncPostLimit, defaultFBCommentPostLimit, maxFBCommentPostLimit)
	settings.DefaultSyncCommentsPerPost = clampPositive(settings.DefaultSyncCommentsPerPost, defaultFBCommentsPerPost, maxFBCommentsPerPost)
}

func normalizeFacebookCommentForSave(comment *models.FacebookComment) {
	comment.PageID = truncateFacebookCommentField(comment.PageID, 255)
	comment.PageName = truncateFacebookCommentField(comment.PageName, 255)
	comment.PostID = truncateFacebookCommentField(comment.PostID, 255)
	comment.ExternalID = truncateFacebookCommentField(comment.ExternalID, 255)
	comment.ParentID = truncateFacebookCommentField(comment.ParentID, 255)
	comment.FromID = truncateFacebookCommentField(comment.FromID, 255)
	comment.FromName = truncateFacebookCommentField(comment.FromName, 255)
}

func truncateFacebookCommentField(value string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes])
}

func (a *App) facebookAccountsForCommentSync(db *gorm.DB, orgID uuid.UUID, accountID string) ([]models.FacebookAccount, error) {
	query := a.DB.Where("organization_id = ? AND method = ? AND status = ?", orgID, models.FBAccountMethodOAuth, models.FBAccountStatusActive)
	if strings.TrimSpace(accountID) != "" {
		parsed, err := uuid.Parse(strings.TrimSpace(accountID))
		if err != nil {
			return nil, errors.New("Invalid Facebook account id")
		}
		query = query.Where("id = ?", parsed)
	}
	var accounts []models.FacebookAccount
	if err := query.Find(&accounts).Error; err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return nil, errors.New("No active OAuth Facebook accounts found")
	}
	return accounts, nil
}

func (a *App) facebookPageTokens(account *models.FacebookAccount) (map[string]string, error) {
	tokenJSON, err := appcrypto.DecryptWithPolicy(account.PageTokens, a.Config.App.EncryptionKey, true)
	if err != nil {
		return nil, err
	}
	var pageTokens map[string]string
	if err := json.Unmarshal([]byte(tokenJSON), &pageTokens); err != nil {
		return nil, err
	}
	return pageTokens, nil
}

func facebookAccountPageNames(account models.FacebookAccount) map[string]string {
	names := map[string]string{}
	if account.Data == nil {
		return names
	}
	pages, ok := account.Data["pages"].([]any)
	if !ok {
		return names
	}
	for _, rawPage := range pages {
		page, ok := rawPage.(map[string]any)
		if !ok {
			continue
		}
		id := strings.TrimSpace(fmt.Sprint(page["id"]))
		name := strings.TrimSpace(fmt.Sprint(page["name"]))
		if id != "" {
			names[id] = name
		}
	}
	return names
}

type facebookCommentSyncResult struct {
	Synced      int
	Created     int
	AutoReplies int
	Failures    []string
}

func (a *App) syncFacebookPageComments(db *gorm.DB, orgID uuid.UUID, account models.FacebookAccount, pageID, pageName, pageToken string, postIDs []string, postLimit, commentsPerPost int, settings *models.FacebookCommentSettings, runAutoReply bool, userID uuid.UUID) facebookCommentSyncResult {
	var result facebookCommentSyncResult
	oauthCfg, err := a.facebookOAuthRuntimeConfig(nil)
	if err != nil {
		result.Failures = append(result.Failures, err.Error())
		return result
	}
	posts, err := a.fetchFacebookPostsWithComments(oauthCfg, pageID, pageToken, postIDs, postLimit, commentsPerPost)
	if err != nil {
		result.Failures = append(result.Failures, fmt.Sprintf("%s: %v", pageNameOrID(pageName, pageID), err))
		return result
	}
	actorFallbacks := a.fetchMissingFacebookCommentActors(oauthCfg, posts, pageToken, pageID, pageName)
	now := time.Now()
	for _, post := range posts {
		for _, edge := range post.Comments.Data {
			if strings.TrimSpace(edge.ID) == "" {
				continue
			}
			actor := edge.From
			if strings.TrimSpace(actor.ID) == "" && strings.TrimSpace(actor.Name) == "" {
				if fetchedActor, ok := actorFallbacks[edge.ID]; ok {
					actor = fetchedActor
				}
			}
			isAdminReply := isFacebookPageAdminCommenter(pageID, actor.ID) || isFacebookPageAdminCommenter(pageID, edge.From.ID)
			commentedAt := parseFacebookTime(edge.CreatedTime, now)
			comment := models.FacebookComment{
				OrganizationID: orgID,
				AccountID:      account.ID,
				PageID:         pageID,
				PageName:       pageName,
				PostID:         post.ID,
				PostPermalink:  post.PermalinkURL,
				PostMessage:    post.Message,
				ExternalID:     edge.ID,
				ParentID:       parentID(edge.Parent),
				FromID:         actor.ID,
				FromName:       actor.Name,
				Message:        edge.Message,
				Permalink:      edge.PermalinkURL,
				Status:         models.FBCommentStatusOpen,
				Direction:      models.FBCommentDirectionIncoming,
				IsAdminReply:   isAdminReply,
				CommentedAt:    commentedAt,
				LastSyncedAt:   &now,
				Metadata: models.JSONB{
					"comment_count": edge.CommentCount,
					"source":        "graph_sync",
				},
			}
			normalizeFacebookCommentForSave(&comment)
			commentDB := db.Session(&gorm.Session{NewDB: true}).Table((&models.FacebookComment{}).TableName())
			tx := commentDB.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "organization_id"}, {Name: "external_id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"page_name", "post_permalink", "post_message", "parent_id", "from_id", "from_name",
					"message", "permalink", "commented_at", "last_synced_at", "metadata",
					"is_admin_reply", "updated_at",
				}),
			}).Create(&comment)
			if tx.Error != nil {
				a.Log.Error("Failed to save Facebook synced comment", "error", tx.Error, "page_id", pageID, "page_name", pageName, "comment_id", edge.ID)
				result.Failures = append(result.Failures, fmt.Sprintf("%s: failed to save comment %s: %v", pageNameOrID(pageName, pageID), edge.ID, tx.Error))
				continue
			}
			result.Synced++
			if tx.RowsAffected > 0 {
				var saved models.FacebookComment
				_ = commentDB.Where("organization_id = ? AND external_id = ?", orgID, edge.ID).First(&saved).Error
				if saved.CreatedAt.Equal(saved.UpdatedAt) || saved.LastRepliedAt == nil {
					result.Created++
				}
				if runAutoReply && shouldAutoReplyFacebookComment(db, settings, saved) {
					if _, err := a.sendAndStoreFacebookCommentReply(db, &account, &saved, userID, settings.AutoCommentReplyText, settings.AutoPrivateMessageText, settings.AutoCommentReplyEnabled, settings.AutoPrivateReplyEnabled, true, pageToken); err == nil {
						result.AutoReplies++
					} else {
						result.Failures = append(result.Failures, fmt.Sprintf("%s: auto reply failed for %s", pageNameOrID(pageName, pageID), edge.ID))
					}
				}
			}
		}
	}
	return result
}

func (a *App) fetchFacebookPostsWithComments(cfg facebookOAuthRuntimeConfig, pageID, pageToken string, postIDs []string, postLimit, commentsPerPost int) ([]facebookPostEdge, error) {
	if len(postIDs) > 0 {
		posts := make([]facebookPostEdge, 0, len(postIDs))
		for _, postID := range postIDs {
			postID = strings.TrimSpace(postID)
			if postID == "" {
				continue
			}
			post, err := a.fetchFacebookPostWithComments(cfg, postID, pageToken, commentsPerPost)
			if err != nil {
				return posts, err
			}
			posts = append(posts, post)
		}
		return posts, nil
	}

	query := url.Values{
		"fields":       {fmt.Sprintf("id,message,permalink_url,created_time,comments.limit(%d){id,message,from{id,name},created_time,permalink_url,comment_count,parent}", commentsPerPost)},
		"limit":        {strconv.Itoa(postLimit)},
		"access_token": {pageToken},
	}
	endpoint := fmt.Sprintf("%s/%s/%s/posts?%s", cfg.BaseURL, cfg.APIVersion, url.PathEscape(pageID), query.Encode())
	var payload facebookPostEdgeResponse
	if err := a.facebookJSONRequest(http.MethodGet, endpoint, nil, &payload); err != nil {
		return nil, err
	}
	return payload.Data, nil
}

func (a *App) fetchFacebookPostWithComments(cfg facebookOAuthRuntimeConfig, postID, pageToken string, commentsPerPost int) (facebookPostEdge, error) {
	query := url.Values{
		"fields":       {fmt.Sprintf("id,message,permalink_url,created_time,comments.limit(%d){id,message,from{id,name},created_time,permalink_url,comment_count,parent}", commentsPerPost)},
		"access_token": {pageToken},
	}
	endpoint := fmt.Sprintf("%s/%s/%s?%s", cfg.BaseURL, cfg.APIVersion, url.PathEscape(postID), query.Encode())
	var payload facebookPostEdge
	if err := a.facebookJSONRequest(http.MethodGet, endpoint, nil, &payload); err != nil {
		return payload, err
	}
	return payload, nil
}

func (a *App) fetchMissingFacebookCommentActors(cfg facebookOAuthRuntimeConfig, posts []facebookPostEdge, pageToken, pageID, pageName string) map[string]facebookCommentActor {
	commentIDs := make([]string, 0)
	seen := make(map[string]struct{})
	for _, post := range posts {
		for _, edge := range post.Comments.Data {
			if strings.TrimSpace(edge.ID) == "" {
				continue
			}
			if strings.TrimSpace(edge.From.ID) != "" || strings.TrimSpace(edge.From.Name) != "" {
				continue
			}
			if _, ok := seen[edge.ID]; ok {
				continue
			}
			seen[edge.ID] = struct{}{}
			commentIDs = append(commentIDs, edge.ID)
		}
	}
	actors := make(map[string]facebookCommentActor, len(commentIDs))
	if len(commentIDs) == 0 {
		return actors
	}
	endpoint := fmt.Sprintf("%s/%s/", cfg.BaseURL, cfg.APIVersion)
	for start := 0; start < len(commentIDs); start += 50 {
		end := start + 50
		if end > len(commentIDs) {
			end = len(commentIDs)
		}
		requests := make([]map[string]string, 0, end-start)
		for _, commentID := range commentIDs[start:end] {
			requests = append(requests, map[string]string{
				"method":       http.MethodGet,
				"relative_url": fmt.Sprintf("%s?fields=from{id,name}", url.PathEscape(commentID)),
			})
		}
		batchBody, err := json.Marshal(requests)
		if err != nil {
			a.Log.Warn("Failed to build Facebook comment actor batch", "error", err, "page_id", pageID, "page_name", pageName)
			continue
		}
		responses, err := a.facebookGraphBatchFormPost(endpoint, url.Values{
			"access_token": {pageToken},
			"batch":        {string(batchBody)},
		})
		if err != nil {
			a.Log.Warn("Failed to fetch Facebook comment actors batch", "error", err, "page_id", pageID, "page_name", pageName, "comment_count", end-start)
			continue
		}
		for idx, response := range responses {
			if idx >= end-start || response.Code < 200 || response.Code >= 300 || strings.TrimSpace(response.Body) == "" {
				continue
			}
			var detail facebookCommentDetailResponse
			if err := json.Unmarshal([]byte(response.Body), &detail); err != nil {
				continue
			}
			if strings.TrimSpace(detail.From.ID) == "" && strings.TrimSpace(detail.From.Name) == "" {
				continue
			}
			actors[commentIDs[start+idx]] = detail.From
		}
	}
	return actors
}

func (a *App) facebookGraphBatchFormPost(endpoint string, form url.Values) ([]facebookGraphBatchResponse, error) {
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := a.oauthHTTPClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Facebook Graph API returned status %d", resp.StatusCode)
	}
	var responses []facebookGraphBatchResponse
	if err := json.Unmarshal(body, &responses); err != nil {
		return nil, err
	}
	return responses, nil
}

func shouldAutoReplyFacebookComment(db *gorm.DB, settings *models.FacebookCommentSettings, comment models.FacebookComment) bool {
	if settings == nil || !settings.AutoReplyEnabled || comment.ID == uuid.Nil {
		return false
	}
	if comment.IsAdminReply {
		return false
	}
	if comment.AutoRepliedAt != nil {
		return false
	}
	if settings.OnlyAutoReplyUnanswered {
		var count int64
		if err := db.Model(&models.FacebookCommentReply{}).Where("comment_id = ? AND status IN ?", comment.ID, []string{fbCommentReplyStatusSent, fbCommentReplyStatusPartial}).Count(&count).Error; err != nil {
			return false
		}
		return count == 0
	}
	return true
}

func (a *App) facebookCommentOperationContext(db *gorm.DB, orgID uuid.UUID, commentRef string) (*models.FacebookComment, *models.FacebookAccount, string, error) {
	var comment models.FacebookComment
	query := db.Where("organization_id = ?", orgID)
	if commentID, err := uuid.Parse(strings.TrimSpace(commentRef)); err == nil {
		query = query.Where("id = ? OR external_id = ?", commentID, commentRef)
	} else {
		query = query.Where("external_id = ?", commentRef)
	}
	if err := query.First(&comment).Error; err != nil {
		return nil, nil, "", err
	}
	var account models.FacebookAccount
	if err := db.Where("id = ? AND organization_id = ?", comment.AccountID, orgID).First(&account).Error; err != nil {
		return nil, nil, "", err
	}
	pageTokens, err := a.facebookPageTokens(&account)
	if err != nil {
		return nil, nil, "", err
	}
	pageToken := pageTokens[comment.PageID]
	if strings.TrimSpace(pageToken) == "" {
		return nil, nil, "", errors.New("page token not found")
	}
	return &comment, &account, pageToken, nil
}

func (a *App) sendAndStoreFacebookCommentReply(db *gorm.DB, account *models.FacebookAccount, comment *models.FacebookComment, userID uuid.UUID, replyText, privateMessageText string, sendCommentReply, sendPrivateMessage, isAuto bool, pageToken string) (models.FacebookCommentReply, error) {
	now := time.Now()
	reply := models.FacebookCommentReply{
		OrganizationID:     comment.OrganizationID,
		CommentID:          comment.ID,
		AccountID:          account.ID,
		PageID:             comment.PageID,
		UserID:             userID,
		ReplyText:          strings.TrimSpace(replyText),
		PrivateMessageText: strings.TrimSpace(privateMessageText),
		Status:             fbCommentReplyStatusSent,
		IsAuto:             isAuto,
		Metadata:           models.JSONB{},
	}

	var errorsOut []string
	var dmSkipped bool
	if sendCommentReply {
		payload, err := a.sendFacebookCommentReply(comment.ExternalID, reply.ReplyText, pageToken)
		if err != nil {
			errorsOut = append(errorsOut, "comment reply: "+err.Error())
		} else {
			reply.GraphCommentReplyID = strings.TrimSpace(fmt.Sprint(payload["id"]))
		}
	}
	if sendPrivateMessage {
		payload, err := a.sendFacebookPrivateReply(comment.PageID, comment.ExternalID, comment.FromID, reply.PrivateMessageText, pageToken)
		if err != nil {
			if isFacebookUserCantDMError(err) {
				dmSkipped = true
				reply.Metadata["dm_skipped"] = true
				reply.Metadata["dm_skip_reason"] = "user_cant_be_dmed"
				a.Log.Warn("Facebook DM skipped: user cannot be messaged", "error", err, "comment_id", comment.ExternalID, "from_id", comment.FromID)
			} else {
				errorsOut = append(errorsOut, "private reply: "+err.Error())
			}
		} else {
			reply.GraphPrivateReplyID = facebookGraphMessageID(payload)
		}
	}
	commentLegSucceeded := sendCommentReply && reply.GraphCommentReplyID != ""
	dmLegSucceeded := sendPrivateMessage && reply.GraphPrivateReplyID != ""
	switch {
	case len(errorsOut) > 0 && (commentLegSucceeded || dmLegSucceeded):
		reply.ErrorMessage = strings.Join(errorsOut, "; ")
		reply.Status = fbCommentReplyStatusPartial
	case len(errorsOut) > 0:
		reply.ErrorMessage = strings.Join(errorsOut, "; ")
		reply.Status = fbCommentReplyStatusFailed
	case dmSkipped && !commentLegSucceeded:
		reply.Status = fbCommentReplyStatusSkipped
	default:
		reply.Status = fbCommentReplyStatusSent
	}
	if err := db.Create(&reply).Error; err != nil {
		return reply, err
	}
	updates := map[string]any{"last_replied_at": &now}
	if reply.Status == fbCommentReplyStatusSent || reply.Status == fbCommentReplyStatusPartial {
		updates["status"] = models.FBCommentStatusReplied
	}
	if isAuto {
		updates["auto_replied_at"] = &now
	}
	updateDB := db.Session(&gorm.Session{NewDB: true})
	_ = updateDB.Model(&models.FacebookComment{}).Where("id = ?", comment.ID).Updates(updates).Error
	if a.WSHub != nil {
		updated := models.FacebookComment{}
		_ = db.Where("id = ?", comment.ID).First(&updated).Error
		updated.Replies = []models.FacebookCommentReply{reply}
		a.WSHub.BroadcastToOrg(updated.OrganizationID, websocket.WSMessage{
			Type:    websocket.TypeFacebookCommentUpdated,
			Payload: facebookCommentToResponse(updated),
		})
	}
	return reply, nil
}

func (a *App) sendFacebookCommentReply(commentExternalID, message, pageToken string) (map[string]any, error) {
	oauthCfg, err := a.facebookOAuthRuntimeConfig(nil)
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/%s/%s/comments", oauthCfg.BaseURL, oauthCfg.APIVersion, url.PathEscape(commentExternalID))
	form := url.Values{"message": {message}, "access_token": {pageToken}}
	return a.facebookGraphFormPost(endpoint, form)
}

func (a *App) sendFacebookPrivateReply(pageID, commentExternalID, senderID, message, pageToken string) (map[string]any, error) {
	oauthCfg, err := a.facebookOAuthRuntimeConfig(nil)
	if err != nil {
		return nil, err
	}
	payload, err := a.sendFacebookCommentPrivateMessage(oauthCfg, pageID, commentExternalID, message, pageToken)
	if err == nil {
		return payload, nil
	}
	if strings.TrimSpace(senderID) == "" {
		return payload, err
	}
	return a.sendFacebookDirectMessengerMessage(oauthCfg, senderID, message, pageToken)
}

func (a *App) sendFacebookCommentPrivateMessage(oauthCfg facebookOAuthRuntimeConfig, pageID, commentID, message, pageToken string) (map[string]any, error) {
	endpoint := fmt.Sprintf("%s/%s/%s/messages?access_token=%s", oauthCfg.BaseURL, oauthCfg.APIVersion, url.PathEscape(pageID), url.QueryEscape(pageToken))
	body := map[string]any{
		"recipient": map[string]any{"comment_id": commentID},
		"message":   map[string]any{"text": message},
	}
	return a.facebookGraphJSONPost(endpoint, body)
}

func (a *App) sendFacebookDirectMessengerMessage(oauthCfg facebookOAuthRuntimeConfig, senderID, message, pageToken string) (map[string]any, error) {
	endpoint := fmt.Sprintf("%s/%s/me/messages?access_token=%s", oauthCfg.BaseURL, oauthCfg.APIVersion, url.QueryEscape(pageToken))
	body := map[string]any{
		"messaging_type": "RESPONSE",
		"recipient":      map[string]any{"id": senderID},
		"message":        map[string]any{"text": message},
	}
	return a.facebookGraphJSONPost(endpoint, body)
}



func isFacebookUserCantDMError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "10903") ||
		strings.Contains(msg, "can't reply to this activity") ||
		strings.Contains(msg, "cant reply to this activity")
}

func facebookGraphMessageID(payload map[string]any) string {
	for _, key := range []string{"id", "message_id", "recipient_id"} {
		if value := strings.TrimSpace(fmt.Sprint(payload[key])); value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}

func facebookCommentToResponse(comment models.FacebookComment) facebookCommentResponse {
	replies := make([]facebookCommentReplyResponse, len(comment.Replies))
	for i, reply := range comment.Replies {
		replies[i] = facebookCommentReplyToResponse(reply)
	}
	return facebookCommentResponse{
		ID:            comment.ID,
		AccountID:     comment.AccountID,
		PageID:        comment.PageID,
		PageName:      comment.PageName,
		PostID:        comment.PostID,
		PostPermalink: comment.PostPermalink,
		PostMessage:   comment.PostMessage,
		ExternalID:    comment.ExternalID,
		ParentID:      comment.ParentID,
		FromID:        comment.FromID,
		FromName:      comment.FromName,
		Message:       comment.Message,
		Permalink:     comment.Permalink,
		Status:        comment.Status,
		Direction:     comment.Direction,
		IsAdminReply:  comment.IsAdminReply,
		CommentedAt:   comment.CommentedAt.Format(time.RFC3339),
		LastSyncedAt:  optionalTimeRFC3339(comment.LastSyncedAt),
		LastRepliedAt: optionalTimeRFC3339(comment.LastRepliedAt),
		AutoRepliedAt: optionalTimeRFC3339(comment.AutoRepliedAt),
		Metadata:      comment.Metadata,
		Replies:       replies,
	}
}

func facebookCommentReplyToResponse(reply models.FacebookCommentReply) facebookCommentReplyResponse {
	return facebookCommentReplyResponse{
		ID:                  reply.ID,
		UserID:              reply.UserID,
		ReplyText:           reply.ReplyText,
		PrivateMessageText:  reply.PrivateMessageText,
		GraphCommentReplyID: reply.GraphCommentReplyID,
		GraphPrivateReplyID: reply.GraphPrivateReplyID,
		Status:              reply.Status,
		ErrorMessage:        reply.ErrorMessage,
		IsAuto:              reply.IsAuto,
		CreatedAt:           reply.CreatedAt.Format(time.RFC3339),
	}
}

func optionalTimeRFC3339(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}

func queryInt(r *fastglue.Request, key string, fallback, min, max int) int {
	raw := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek(key)))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	if parsed < min {
		return min
	}
	if parsed > max {
		return max
	}
	return parsed
}

func clampPositive(value, fallback, max int) int {
	if value <= 0 {
		value = fallback
	}
	if value > max {
		return max
	}
	return value
}

func parseFacebookTime(value string, fallback time.Time) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05-0700"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func parentID(parent *facebookCommentParentRef) string {
	if parent == nil {
		return ""
	}
	return parent.ID
}

func pageNameOrID(name, id string) string {
	if strings.TrimSpace(name) != "" {
		return name
	}
	return id
}

func validFacebookCommentStatus(status models.FacebookCommentStatus) bool {
	switch status {
	case models.FBCommentStatusOpen, models.FBCommentStatusReplied, models.FBCommentStatusClosed, models.FBCommentStatusArchived:
		return true
	default:
		return false
	}
}
