package whatsmeow

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/pkg/chat_close_ratings"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	defaultChatCloseRatingFollowupWindowMinutes = 15
	maxChatCloseRatingFollowupWindowMinutes     = 1440
	chatCloseRatingFollowupMessageLimit         = 3

	chatCloseRatingFollowupContextKey  = "followup"
	chatCloseRatingFollowupEntriesKey  = "entries"
	chatCloseRatingFollowupCommentsKey = "comments"
)

type chatCloseRatingSettings struct {
	Enabled               bool
	FollowupWindowMinutes int
}

func readChatCloseRatingSettings(settings models.JSONB) chatCloseRatingSettings {
	result := chatCloseRatingSettings{
		Enabled:               true,
		FollowupWindowMinutes: chat_close_ratings.DefaultFollowupWindowMinutes,
	}

	if settings == nil {
		return result
	}

	if rawEnabled, ok := settings[InstanceSettingChatCloseRatingEnabled]; ok {
		if enabled, ok := rawEnabled.(bool); ok {
			result.Enabled = enabled
		}
	}
	if rawFollowupWindow, ok := settings[InstanceSettingChatCloseRatingFollowupWindowMinutes]; ok {
		if parsed := chat_close_ratings.ParseFollowupWindowMinutes(rawFollowupWindow); parsed > 0 {
			result.FollowupWindowMinutes = parsed
		}
	}

	return result
}

func (cm *ConnectionManager) loadChatCloseRatingSettings(ctx context.Context, instanceID *uuid.UUID) (chatCloseRatingSettings, error) {
	result := chatCloseRatingSettings{
		Enabled:               true,
		FollowupWindowMinutes: defaultChatCloseRatingFollowupWindowMinutes,
	}
	if instanceID == nil || *instanceID == uuid.Nil {
		return result, nil
	}

	var instance models.WhatsAppInstance
	if err := cm.db.WithContext(ctx).
		Select("settings").
		Where("id = ?", *instanceID).
		First(&instance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return result, nil
		}
		return chatCloseRatingSettings{}, err
	}

	return readChatCloseRatingSettings(instance.Settings), nil
}

func (cm *ConnectionManager) findActiveChatCloseRatingCycle(
	ctx context.Context,
	orgID uuid.UUID,
	instanceID *uuid.UUID,
	contactID uuid.UUID,
	now time.Time,
) (*models.ChatClosureRating, chat_close_ratings.FollowupState, error) {
	settings, err := cm.loadChatCloseRatingSettings(ctx, instanceID)
	if err != nil {
		return nil, chat_close_ratings.FollowupState{}, err
	}
	if !settings.Enabled {
		return nil, chat_close_ratings.FollowupState{}, nil
	}

	windowStart := now.UTC().Add(-2 * 24 * time.Hour)

	var cycle models.ChatClosureRating
	if err := cm.db.WithContext(ctx).Where(
		"organization_id = ? AND contact_id = ? AND state IN ? AND closed_at >= ?",
		orgID,
		contactID,
		[]models.ChatClosureRatingState{
			models.ChatClosureRatingStatePending,
			models.ChatClosureRatingStateRated,
		},
		windowStart,
	).Order("closed_at DESC").First(&cycle).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, chat_close_ratings.FollowupState{}, nil
		}
		return nil, chat_close_ratings.FollowupState{}, err
	}

	var closedAt time.Time
	if !cycle.ClosedAt.IsZero() {
		closedAt = cycle.ClosedAt
	} else {
		closedAt = now.UTC()
	}

	followup := chat_close_ratings.ReadFollowupState(closedAt, cycle.ContextMessages, settings.FollowupWindowMinutes)
	if followup.IsActive(now) {
		return &cycle, followup, nil
	}

	if cycle.State == models.ChatClosureRatingStatePending {
		if err := cm.db.WithContext(ctx).
			Model(&models.ChatClosureRating{}).
			Where("id = ? AND state = ?", cycle.ID, models.ChatClosureRatingStatePending).
			Update("state", models.ChatClosureRatingStateExpired).Error; err != nil {
			return nil, followup, err
		}
	}

	return nil, followup, nil
}

func (cm *ConnectionManager) shouldSkipClosedChatAutoReopenForIncomingMessage(
	ctx context.Context,
	orgID uuid.UUID,
	contact *models.Contact,
	msgType models.MessageType,
	content string,
) bool {
	if contact == nil {
		return false
	}
	if contact.EffectiveStatus() != models.ChatStatusClosed {
		return false
	}
	if msgType != models.MessageTypeText {
		return false
	}
	_ = content

	cycle, _, err := cm.findActiveChatCloseRatingCycle(ctx, orgID, contact.InstanceID, contact.ID, time.Now().UTC())
	if err != nil {
		cm.logger.Error("Failed to resolve pending close rating cycle before auto-reopen", "error", err, "organization_id", orgID, "contact_id", contact.ID)
		return false
	}
	return cycle != nil
}

func (cm *ConnectionManager) maybeCaptureChatCloseRating(
	ctx context.Context,
	orgID uuid.UUID,
	contact *models.Contact,
	incomingMessage *models.Message,
) bool {
	if contact == nil || incomingMessage == nil {
		return false
	}
	if incomingMessage.Direction != models.DirectionIncoming || incomingMessage.MessageType != models.MessageTypeText {
		return false
	}

	now := time.Now().UTC()
	cycle, followup, err := cm.findActiveChatCloseRatingCycle(ctx, orgID, contact.InstanceID, contact.ID, now)
	if err != nil {
		cm.logger.Error("Failed to resolve close rating cycle", "error", err, "organization_id", orgID, "contact_id", contact.ID)
		return false
	}
	if cycle == nil {
		return false
	}

	trimmedContent := strings.TrimSpace(incomingMessage.Content)
	ratingValue, hasRating := chat_close_ratings.ParseInboundRatingValue(incomingMessage.Content)
	contextMessages := cycle.ContextMessages
	if hasRating {
		contextMessages = chat_close_ratings.BuildChatCloseRatingContext(cm.db.WithContext(ctx), contact.ID, incomingMessage)
	}
	if contextMessages == nil {
		contextMessages = models.JSONB{}
	}

	var ratingPointer *int
	entryKind := "comment"
	if hasRating {
		ratingPointer = &ratingValue
		entryKind = "rating"
	}
	followup.Entries = chat_close_ratings.AppendChatCloseRatingFollowupEntry(followup.Entries, incomingMessage, incomingMessage.Content, entryKind, ratingPointer)
	if !hasRating && trimmedContent != "" {
		followup.Comments = append(followup.Comments, trimmedContent)
	}
	if followup.RemainingMessages > 0 {
		followup.RemainingMessages--
	}
	contextMessages = chat_close_ratings.WriteFollowupState(contextMessages, followup)

	updates := map[string]any{
		"context_messages": contextMessages,
	}
	if hasRating {
		ratedAt := now
		updates["state"] = models.ChatClosureRatingStateRated
		updates["rating"] = ratingValue
		updates["rated_at"] = &ratedAt
		updates["rating_message"] = trimmedContent
		updates["rating_message_id"] = incomingMessage.ID
	} else if cycle.State == models.ChatClosureRatingStateRated && trimmedContent != "" {
		updates["rating_message"] = trimmedContent
		updates["rating_message_id"] = incomingMessage.ID
	} else if cycle.State == models.ChatClosureRatingStatePending && !followup.IsActive(now) {
		updates["state"] = models.ChatClosureRatingStateExpired
	}

	result := cm.db.WithContext(ctx).
		Model(&models.ChatClosureRating{}).
		Where("id = ? AND state IN ?", cycle.ID, []models.ChatClosureRatingState{
			models.ChatClosureRatingStatePending,
			models.ChatClosureRatingStateRated,
		}).
		Updates(updates)
	if result.Error != nil {
		cm.logger.Error("Failed to persist chat close rating follow-up", "error", result.Error, "cycle_id", cycle.ID)
		return false
	}
	if result.RowsAffected == 0 {
		return false
	}

	if hasRating {
		cm.logger.Info("Captured chat close rating", "cycle_id", cycle.ID, "contact_id", contact.ID, "rating", ratingValue)
	} else {
		cm.logger.Info("Captured chat close rating follow-up comment", "cycle_id", cycle.ID, "contact_id", contact.ID)
	}
	return true
}

