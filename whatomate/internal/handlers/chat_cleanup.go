package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

func parseDeleteChatsQueryFlag(r *fastglue.Request) (bool, error) {
	raw := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek("delete_chats")))
	if raw == "" {
		return false, nil
	}

	deleteChats, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("invalid delete_chats flag")
	}

	return deleteChats, nil
}

func (a *App) deleteWhatsAppAccountWithOptionalChatPurge(account *models.WhatsAppAccount, orgID uuid.UUID, deleteChats bool) error {
	return a.DB.Transaction(func(tx *gorm.DB) error {
		if deleteChats {
			if err := a.purgeChatsByAccountName(tx, orgID, account.Name); err != nil {
				return err
			}
		}

		return tx.Delete(account).Error
	})
}

func (a *App) deleteWhatsAppInstanceWithOptionalChatPurge(instance *models.WhatsAppInstance, orgID uuid.UUID, deleteChats bool) error {
	return a.DB.Transaction(func(tx *gorm.DB) error {
		if deleteChats {
			if err := a.purgeChatsByInstanceID(tx, orgID, instance.ID); err != nil {
				return err
			}
		}

		return tx.Delete(instance).Error
	})
}

func (a *App) purgeChatsByAccountName(tx *gorm.DB, orgID uuid.UUID, accountName string) error {
	contactIDs := func() *gorm.DB {
		return tx.
			Unscoped().
			Model(&models.Contact{}).
			Select("id").
			Where("organization_id = ? AND whatsapp_account = ?", orgID, accountName)
	}

	return a.purgeChatsByContactScope(tx, orgID, contactIDs)
}

func (a *App) purgeChatsByInstanceID(tx *gorm.DB, orgID, instanceID uuid.UUID) error {
	contactIDs := func() *gorm.DB {
		return tx.
			Unscoped().
			Model(&models.Contact{}).
			Select("id").
			Where("organization_id = ? AND instance_id = ?", orgID, instanceID)
	}

	return a.purgeChatsByContactScope(tx, orgID, contactIDs)
}

func (a *App) purgeChatsByContactScope(tx *gorm.DB, orgID uuid.UUID, contactIDs func() *gorm.DB) error {
	sessionIDs := func() *gorm.DB {
		return tx.
			Unscoped().
			Model(&models.ChatbotSession{}).
			Select("id").
			Where("organization_id = ? AND contact_id IN (?)", orgID, contactIDs())
	}

	if err := tx.
		Unscoped().
		Where("session_id IN (?)", sessionIDs()).
		Delete(&models.ChatbotSessionMessage{}).Error; err != nil {
		return fmt.Errorf("failed to delete chatbot session messages: %w", err)
	}

	if err := tx.
		Unscoped().
		Where("organization_id = ? AND contact_id IN (?)", orgID, contactIDs()).
		Delete(&models.ChatbotSession{}).Error; err != nil {
		return fmt.Errorf("failed to delete chatbot sessions: %w", err)
	}

	if err := tx.
		Unscoped().
		Where("organization_id = ? AND contact_id IN (?)", orgID, contactIDs()).
		Delete(&models.ConversationNote{}).Error; err != nil {
		return fmt.Errorf("failed to delete conversation notes: %w", err)
	}

	if err := tx.
		Unscoped().
		Where("organization_id = ? AND contact_id IN (?)", orgID, contactIDs()).
		Delete(&models.AgentTransfer{}).Error; err != nil {
		return fmt.Errorf("failed to delete agent transfers: %w", err)
	}

	if err := tx.
		Unscoped().
		Where("organization_id = ? AND contact_id IN (?)", orgID, contactIDs()).
		Delete(&models.ChatClosureRating{}).Error; err != nil {
		return fmt.Errorf("failed to delete chat closure ratings: %w", err)
	}

	if err := tx.
		Unscoped().
		Where("organization_id = ? AND contact_id IN (?)", orgID, contactIDs()).
		Delete(&models.Message{}).Error; err != nil {
		return fmt.Errorf("failed to delete messages: %w", err)
	}

	if err := tx.
		Unscoped().
		Where("organization_id = ? AND id IN (?)", orgID, contactIDs()).
		Delete(&models.Contact{}).Error; err != nil {
		return fmt.Errorf("failed to delete contacts: %w", err)
	}

	return nil
}
