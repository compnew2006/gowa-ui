package worker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/contactutil"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/queue"
	"github.com/compnew2006/whatomate/pkg/provider"
	"github.com/google/uuid"
)

// handleGroupRecipientJob processes a group recipient message job.
// It is called from HandleRecipientJob as an early-return branch when
// recipient.RecipientType == "group". The existing send pipeline is reused
// — recipient.PhoneNumber is set to the group JID so sendTemplateMessageViaProvider
// works without modification.
func (w *Worker) handleGroupRecipientJob(ctx context.Context, job *queue.RecipientJob, recipient models.BulkMessageRecipient, campaign models.BulkMessageCampaign) error {
	groupJID := recipient.GroupJID
	if !contactutil.IsValidGroupJID(groupJID) {
		reason := fmt.Sprintf("Invalid group JID format: %s", groupJID)
		w.updateRecipientStatus(job.RecipientID, models.MessageStatusFailed, "", reason)
		w.incrementCampaignCount(job.CampaignID, "failed_count")
		return nil
	}

	if !w.isWhatsmeowProvider() {
		reason := "Group messaging is not supported by this provider (Meta Cloud API). Use a whatsmeow instance."
		w.updateRecipientStatus(job.RecipientID, models.MessageStatusFailed, "", reason)
		w.incrementCampaignCount(job.CampaignID, "failed_count")
		return nil
	}

	instanceID := strings.TrimSpace(campaign.WhatsAppAccount)

	// Verify group still exists (optional but recommended).
	if gp, ok := w.MessageProvider.(provider.GroupProvider); ok {
		if _, err := gp.VerifyGroupMembership(ctx, instanceID, groupJID); err != nil {
			reason := fmt.Sprintf("Group not found or inaccessible: %s", groupJID)
			w.updateRecipientStatus(job.RecipientID, models.MessageStatusFailed, "", reason)
			w.incrementCampaignCount(job.CampaignID, "failed_count")
			return nil
		}
	}

	// Use group JID as the phone number so the existing send pipeline works.
	recipient.PhoneNumber = groupJID

	// Build template params without a contact (nil is safe — resolver uses fallbacks).
	templateBody := ""
	if campaign.Template != nil {
		templateBody = campaign.Template.BodyContent
	}
	mergedTemplateParams := w.resolveCampaignTemplateParams(
		ctx,
		job.OrganizationID,
		nil,
		groupJID,
		recipient.GroupName,
		templateBody,
		job.TemplateParams,
	)
	recipient.TemplateParams = mergedTemplateParams

	delayScopeKey := resolveCampaignDelayScopeKey(campaign.WhatsAppAccount, campaign.ID)
	delayDuration, delayErr := w.computeCampaignDelayDuration(ctx, delayScopeKey, campaign.MinDelaySeconds, campaign.MaxDelaySeconds)
	if delayErr != nil {
		return fmt.Errorf("failed to compute campaign send delay: %w", delayErr)
	}

	if delayDuration > 0 && w.Redis != nil {
		sendAt := time.Now().Add(delayDuration)
		if schedErr := w.scheduleRecipientSend(ctx, job, sendAt); schedErr != nil {
			w.Log.Error("Failed to schedule group send, falling back to immediate", "error", schedErr, "recipient_id", job.RecipientID)
		} else {
			if transErr := w.transitionRecipientToSending(ctx, job.RecipientID, uuid.New().String()); transErr != nil {
				w.Log.Warn("Group recipient already being sent by another worker (scheduled send will be skipped at poll time)", "recipient_id", job.RecipientID, "error", transErr)
			}
			return nil
		}
	}

	if delayDuration > 0 && w.Redis == nil {
		if sleepErr := sleepWithContext(ctx, delayDuration); sleepErr != nil {
			return fmt.Errorf("campaign delay interrupted: %w", sleepErr)
		}
	}

	if w.MessageProvider == nil {
		sendErr := fmt.Errorf("message provider is not configured")
		w.updateRecipientStatus(job.RecipientID, models.MessageStatusFailed, "", sendErr.Error())
		w.incrementCampaignCount(job.CampaignID, "failed_count")
		return nil
	}

	waMessageID, sendErr := w.sendTemplateMessageViaProvider(ctx, instanceID, &campaign, campaign.Template, &recipient)

	// Create Message record (no ContactID for group recipients).
	var messageInstanceID *uuid.UUID
	if parsedInstanceID, parseErr := uuid.Parse(instanceID); parseErr == nil {
		messageInstanceID = &parsedInstanceID
	}

	message := models.Message{
		OrganizationID:    job.OrganizationID,
		InstanceID:        messageInstanceID,
		WhatsAppAccount:   campaign.WhatsAppAccount,
		WhatsAppMessageID: waMessageID,
		Direction:         models.DirectionOutgoing,
		MessageType:       models.MessageTypeTemplate,
		TemplateParams:    recipient.TemplateParams,
		Metadata: models.JSONB{
			"campaign_id":    job.CampaignID.String(),
			"recipient_name": job.RecipientName,
			"group_jid":      groupJID,
			"group_name":     recipient.GroupName,
		},
	}
	if campaign.Template != nil {
		message.TemplateName = campaign.Template.Name
		message.Content = renderCampaignTemplateBody(campaign.Template.BodyContent, recipient.TemplateParams)
	}

	if sendErr != nil {
		w.Log.Error("Failed to send group message", "error", sendErr, "group_jid", groupJID)
		message.Status = models.MessageStatusFailed
		message.ErrorMessage = sendErr.Error()
		w.updateRecipientStatus(job.RecipientID, models.MessageStatusFailed, "", sendErr.Error())
		w.incrementCampaignCount(job.CampaignID, "failed_count")
	} else {
		w.Log.Info("Group message sent", "group_jid", groupJID, "message_id", waMessageID)
		message.Status = models.MessageStatusSent
		w.updateRecipientStatus(job.RecipientID, models.MessageStatusSent, waMessageID, "")
		w.incrementCampaignCount(job.CampaignID, "sent_count")
	}

	if err := w.DB.Create(&message).Error; err != nil {
		w.Log.Error("Failed to save group message", "error", err, "group_jid", groupJID)
	}

	w.checkCampaignCompletion(ctx, job.CampaignID, job.OrganizationID)
	return nil
}
