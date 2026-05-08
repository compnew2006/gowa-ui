package handlers

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/pkg/whatsapp"
	"github.com/google/uuid"
)

func (a *App) sendOutOfHoursMessage(account *models.WhatsAppAccount, contact *models.Contact, settings *models.ChatbotSettings) {
	if settings.BusinessHours.OutOfHoursMessage == "" {
		return
	}
	if err := a.sendAndSaveTextMessage(account, contact, settings.BusinessHours.OutOfHoursMessage); err != nil {
		a.Log.Error("Failed to send out of hours message", "error", err, "contact", contact.PhoneNumber)
	}
}

func (a *App) handleChatbotConversation(
	account *models.WhatsAppAccount,
	contact *models.Contact,
	msg IncomingTextMessage,
	settings *models.ChatbotSettings,
	payload incomingMessagePayload,
) {
	a.Log.Info("Processing message", "text", payload.MessageText, "buttonID", payload.ButtonID, "from", msg.From)

	session, isNewSession := a.getOrCreateSession(account.OrganizationID, contact.ID, account.Name, msg.From, settings.SessionTimeoutMins)

	a.logSessionMessage(session.ID, models.DirectionIncoming, payload.MessageText, "keyword_check")

	keywordResponse, keywordMatched := a.matchKeywordRules(account.OrganizationID, account.Name, payload.MessageText)
	if keywordMatched && keywordResponse.ResponseType == models.ResponseTypeTransfer {
		a.Log.Info("Transfer keyword matched", "response", keywordResponse.Body)
		renderedKeywordBody := a.renderMessageTemplatePlaceholders(context.Background(), account.OrganizationID, contact, keywordResponse.Body)
		if settings.BusinessHours.Enabled && len(settings.BusinessHours.Hours) > 0 {
			if !a.isWithinBusinessHours(settings.BusinessHours.Hours) {
				a.Log.Info("Outside business hours, sending out of hours message instead of transfer")
				a.sendOutOfHoursMessage(account, contact, settings)
				return
			}
		}
		if renderedKeywordBody != "" {
			if err := a.sendAndSaveTextMessage(account, contact, renderedKeywordBody); err != nil {
				a.Log.Error("Failed to send transfer message", "error", err, "contact", contact.PhoneNumber)
			}
		}
		a.createTransferFromKeyword(account, contact)
		return
	}

	if session.CurrentFlowID != nil {
		a.processFlowResponse(account, session, contact, payload.MessageText, payload.ButtonID, payload.FlowResponseData)
		return
	}

	if flow := a.matchFlowTrigger(account.OrganizationID, account.Name, payload.MessageText); flow != nil {
		a.startFlow(account, session, contact, flow)
		return
	}

	if isNewSession && settings.DefaultResponse != "" {
		a.Log.Info("New session - sending greeting message", "contact", contact.PhoneNumber)
		if len(settings.GreetingButtons) > 0 {
			greetingButtons := make([]map[string]any, 0)
			for _, btn := range settings.GreetingButtons {
				if btnMap, ok := btn.(map[string]any); ok {
					greetingButtons = append(greetingButtons, btnMap)
				}
			}
			if len(greetingButtons) > 0 {
				if err := a.sendAndSaveInteractiveButtons(account, contact, settings.DefaultResponse, greetingButtons); err != nil {
					a.Log.Error("Failed to send greeting buttons", "error", err, "contact", contact.PhoneNumber)
				}
			} else {
				if err := a.sendAndSaveTextMessage(account, contact, settings.DefaultResponse); err != nil {
					a.Log.Error("Failed to send greeting message", "error", err, "contact", contact.PhoneNumber)
				}
			}
		} else {
			if err := a.sendAndSaveTextMessage(account, contact, settings.DefaultResponse); err != nil {
				a.Log.Error("Failed to send greeting message", "error", err, "contact", contact.PhoneNumber)
			}
		}
		a.logSessionMessage(session.ID, models.DirectionOutgoing, settings.DefaultResponse, "greeting")
		return
	}

	if keywordMatched && keywordResponse.ResponseType != models.ResponseTypeTransfer {
		a.Log.Info("Keyword rule matched", "response_type", keywordResponse.ResponseType, "response", keywordResponse.Body)
		renderedKeywordBody := a.renderMessageTemplatePlaceholders(context.Background(), account.OrganizationID, contact, keywordResponse.Body)

		if len(keywordResponse.Buttons) > 0 {
			if err := a.sendAndSaveInteractiveButtons(account, contact, renderedKeywordBody, keywordResponse.Buttons); err != nil {
				a.Log.Error("Failed to send interactive buttons", "error", err, "contact", contact.PhoneNumber)
			}
		} else {
			if err := a.sendAndSaveTextMessage(account, contact, renderedKeywordBody); err != nil {
				a.Log.Error("Failed to send text message", "error", err, "contact", contact.PhoneNumber)
			}
		}
		a.logSessionMessage(session.ID, models.DirectionOutgoing, renderedKeywordBody, "keyword_response")
		return
	}

	if settings.AI.Enabled && settings.AI.Provider != "" && settings.AI.APIKey != "" {
		a.Log.Info("Attempting AI response", "provider", settings.AI.Provider, "model", settings.AI.Model)
		aiResponse, err := a.generateAIResponse(settings, session, payload.MessageText)
		if err != nil {
			a.Log.Error("AI response failed", "error", err, "provider", settings.AI.Provider, "model", settings.AI.Model)
		} else if aiResponse != "" {
			a.Log.Info("AI response generated successfully", "response_length", len(aiResponse))
			if err := a.sendAndSaveTextMessage(account, contact, aiResponse); err != nil {
				a.Log.Error("Failed to send AI response", "error", err, "contact", contact.PhoneNumber)
			}
			a.logSessionMessage(session.ID, models.DirectionOutgoing, aiResponse, "ai_response")
			return
		} else {
			a.Log.Warn("AI returned empty response")
		}
	} else {
		a.Log.Info("AI not configured", "ai_enabled", settings.AI.Enabled, "has_provider", settings.AI.Provider != "", "has_api_key", settings.AI.APIKey != "")
	}

	if settings.FallbackMessage != "" && !isNewSession {
		a.Log.Info("Sending fallback message", "response", settings.FallbackMessage)
		if len(settings.FallbackButtons) > 0 {
			fallbackButtons := make([]map[string]any, 0)
			for _, btn := range settings.FallbackButtons {
				if btnMap, ok := btn.(map[string]any); ok {
					fallbackButtons = append(fallbackButtons, btnMap)
				}
			}
			if len(fallbackButtons) > 0 {
				if err := a.sendAndSaveInteractiveButtons(account, contact, settings.FallbackMessage, fallbackButtons); err != nil {
					a.Log.Error("Failed to send fallback buttons", "error", err, "contact", contact.PhoneNumber)
				}
			} else {
				if err := a.sendAndSaveTextMessage(account, contact, settings.FallbackMessage); err != nil {
					a.Log.Error("Failed to send fallback message", "error", err, "contact", contact.PhoneNumber)
				}
			}
		} else {
			if err := a.sendAndSaveTextMessage(account, contact, settings.FallbackMessage); err != nil {
				a.Log.Error("Failed to send fallback message", "error", err, "contact", contact.PhoneNumber)
			}
		}
		a.logSessionMessage(session.ID, models.DirectionOutgoing, settings.FallbackMessage, "fallback_response")
	} else if !isNewSession {
		a.Log.Info("No fallback message configured for existing session")
	}
}

type KeywordResponse struct {
	Body         string
	Buttons      []map[string]any
	ResponseType models.ResponseType
}

func (a *App) matchKeywordRules(orgID uuid.UUID, accountName, messageText string) (*KeywordResponse, bool) {
	rules, err := a.getKeywordRulesCached(orgID, accountName)
	if err != nil {
		a.Log.Error("Failed to fetch keyword rules", "error", err)
		return nil, false
	}

	messageLower := strings.ToLower(messageText)

	for _, rule := range rules {
		for _, keyword := range rule.Keywords {
			keywordLower := strings.ToLower(keyword)
			matched := false

			switch rule.MatchType {
			case models.MatchTypeExact:
				if rule.CaseSensitive {
					matched = messageText == keyword
				} else {
					matched = messageLower == keywordLower
				}
			case models.MatchTypeContains:
				if rule.CaseSensitive {
					matched = strings.Contains(messageText, keyword)
				} else {
					matched = strings.Contains(messageLower, keywordLower)
				}
			case models.MatchTypeStartsWith:
				if rule.CaseSensitive {
					matched = strings.HasPrefix(messageText, keyword)
				} else {
					matched = strings.HasPrefix(messageLower, keywordLower)
				}
			case models.MatchTypeRegex:
				re, err := regexp.Compile(keyword)
				if err == nil {
					matched = re.MatchString(messageText)
				}
			default:
				matched = strings.Contains(messageLower, keywordLower)
			}

			if matched {
				response := &KeywordResponse{
					ResponseType: rule.ResponseType,
				}

				if rule.ResponseType == models.ResponseTypeTransfer {
					if body, ok := rule.ResponseContent["body"].(string); ok {
						response.Body = body
					}
					return response, true
				}

				if body, ok := rule.ResponseContent["body"].(string); ok {
					response.Body = body
				}

				if buttons, ok := rule.ResponseContent["buttons"].([]interface{}); ok && len(buttons) > 0 {
					response.Buttons = make([]map[string]any, 0, len(buttons))
					for _, btn := range buttons {
						if btnMap, ok := btn.(map[string]any); ok {
							response.Buttons = append(response.Buttons, btnMap)
						}
					}
				}

				if response.Body != "" {
					return response, true
				}
			}
		}
	}

	return nil, false
}

func (a *App) sendAndSaveTextMessage(account *models.WhatsAppAccount, contact *models.Contact, message string) error {
	ctx := context.Background()
	_, err := a.SendOutgoingMessage(ctx, OutgoingMessageRequest{
		Account: account,
		Contact: contact,
		Type:    models.MessageTypeText,
		Content: message,
	}, ChatbotSendOptions())
	return err
}

func (a *App) sendAndSaveInteractiveButtons(account *models.WhatsAppAccount, contact *models.Contact, bodyText string, buttons []map[string]any) error {
	replyButtons := make([]map[string]any, 0, len(buttons))
	ctaButtons := make([]map[string]any, 0)
	for _, btn := range buttons {
		btnType, _ := btn["type"].(string)
		switch btnType {
		case "url":
			ctaButtons = append(ctaButtons, btn)
		case "phone":
			phoneNumber, _ := btn["phone_number"].(string)
			if phoneNumber != "" {
				ctaButtons = append(ctaButtons, map[string]any{
					"title": btn["title"],
					"url":   "tel:" + phoneNumber,
				})
			}
		default:
			replyButtons = append(replyButtons, btn)
		}
	}

	if len(replyButtons) > 0 && len(ctaButtons) > 0 {
		ctaButtons = nil
	}

	if len(replyButtons) > 0 {
		waButtons := make([]whatsapp.Button, 0, len(replyButtons))
		for i, btn := range replyButtons {
			if i >= 10 {
				break
			}
			buttonID, _ := btn["id"].(string)
			buttonTitle, _ := btn["title"].(string)
			if buttonID == "" {
				buttonID = fmt.Sprintf("btn_%d", i+1)
			}
			if buttonTitle == "" {
				continue
			}
			waButtons = append(waButtons, whatsapp.Button{
				ID:    buttonID,
				Title: buttonTitle,
			})
		}

		if len(waButtons) > 0 {
			interactiveType := "button"
			if len(waButtons) > 3 {
				interactiveType = "list"
			}
			ctx := context.Background()
			if _, err := a.SendOutgoingMessage(ctx, OutgoingMessageRequest{
				Account:         account,
				Contact:         contact,
				Type:            models.MessageTypeInteractive,
				InteractiveType: interactiveType,
				BodyText:        bodyText,
				Buttons:         waButtons,
			}, ChatbotSendOptions()); err != nil {
				return err
			}
		}
	}

	if len(ctaButtons) > 2 {
		ctaButtons = ctaButtons[:2]
	}
	for i, ctaBtn := range ctaButtons {
		btnTitle, _ := ctaBtn["title"].(string)
		btnURL, _ := ctaBtn["url"].(string)
		if btnTitle != "" && btnURL != "" {
			ctaBody := bodyText
			if i > 0 {
				ctaBody = btnTitle
			}
			if err := a.sendAndSaveCTAURLButton(account, contact, ctaBody, btnTitle, btnURL); err != nil {
				return err
			}
		}
	}

	if len(replyButtons) == 0 && len(ctaButtons) == 0 {
		return a.sendAndSaveTextMessage(account, contact, bodyText)
	}

	return nil
}

func (a *App) sendAndSaveCTAURLButton(account *models.WhatsAppAccount, contact *models.Contact, bodyText, buttonText, url string) error {
	ctx := context.Background()
	_, err := a.SendOutgoingMessage(ctx, OutgoingMessageRequest{
		Account:         account,
		Contact:         contact,
		Type:            models.MessageTypeInteractive,
		InteractiveType: "cta_url",
		BodyText:        bodyText,
		ButtonText:      buttonText,
		URL:             url,
	}, ChatbotSendOptions())
	return err
}

func (a *App) sendAndSaveFlowMessage(account *models.WhatsAppAccount, contact *models.Contact, flowID, headerText, bodyText, ctaText, flowToken, firstScreen string) error {
	ctx := context.Background()
	_, err := a.SendOutgoingMessage(ctx, OutgoingMessageRequest{
		Account:         account,
		Contact:         contact,
		Type:            models.MessageTypeFlow,
		FlowID:          flowID,
		FlowHeader:      headerText,
		BodyText:        bodyText,
		FlowCTA:         ctaText,
		FlowToken:       flowToken,
		FlowFirstScreen: firstScreen,
	}, ChatbotSendOptions())
	return err
}

func (a *App) getOrCreateSession(orgID, contactID uuid.UUID, accountName, phoneNumber string, timeoutMins int) (*models.ChatbotSession, bool) {
	now := time.Now()

	var session models.ChatbotSession
	timeout := now.Add(-time.Duration(timeoutMins) * time.Minute)
	result := a.DB.Where("organization_id = ? AND contact_id = ? AND whats_app_account = ? AND status = ? AND last_activity_at > ?",
		orgID, contactID, accountName, models.SessionStatusActive, timeout).First(&session)

	if result.Error == nil {
		a.DB.Model(&session).Update("last_activity_at", now)
		return &session, false
	}

	session = models.ChatbotSession{
		BaseModel:       models.BaseModel{ID: uuid.New()},
		OrganizationID:  orgID,
		ContactID:       contactID,
		WhatsAppAccount: accountName,
		PhoneNumber:     phoneNumber,
		Status:          models.SessionStatusActive,
		SessionData:     models.JSONB{},
		StartedAt:       now,
		LastActivityAt:  now,
	}
	if err := a.DB.Create(&session).Error; err != nil {
		a.Log.Error("Failed to create session", "error", err)
	}
	return &session, true
}

func (a *App) logSessionMessage(sessionID uuid.UUID, direction models.Direction, message, stepName string) {
	msg := models.ChatbotSessionMessage{
		BaseModel: models.BaseModel{ID: uuid.New()},
		SessionID: sessionID,
		Direction: direction,
		Message:   message,
		StepName:  stepName,
	}
	if err := a.DB.Create(&msg).Error; err != nil {
		a.Log.Error("Failed to log session message", "error", err)
	}
}

func (a *App) exitFlow(session *models.ChatbotSession) {
	now := time.Now()
	a.DB.Model(session).Updates(map[string]any{
		"current_step": "",
		"step_retries": 0,
		"status":       models.SessionStatusCompleted,
		"completed_at": now,
	})

	a.ClearContactChatbotTracking(session.ContactID)
}

func (a *App) closeSession(session *models.ChatbotSession) {
	a.DB.Model(session).Updates(map[string]any{
		"status":       models.SessionStatusCompleted,
		"completed_at": time.Now(),
	})

	a.ClearContactChatbotTracking(session.ContactID)
}

func (a *App) replaceVariables(message string, data models.JSONB) string {
	if data == nil {
		return message
	}
	result := message
	for key, value := range data {
		placeholder := "{{" + key + "}}"
		if strVal, ok := value.(string); ok {
			result = strings.ReplaceAll(result, placeholder, strVal)
		}
	}
	return result
}

func (a *App) isWithinBusinessHours(businessHours models.JSONBArray) bool {
	now := time.Now()
	currentDay := int(now.Weekday())
	currentTime := now.Format("15:04")

	for _, bh := range businessHours {
		bhMap, ok := bh.(map[string]any)
		if !ok {
			continue
		}

		day, ok := bhMap["day"].(float64)
		if !ok {
			continue
		}

		if int(day) != currentDay {
			continue
		}

		enabled, ok := bhMap["enabled"].(bool)
		if !ok || !enabled {
			return false
		}

		startTime, ok := bhMap["start_time"].(string)
		if !ok {
			continue
		}
		endTime, ok := bhMap["end_time"].(string)
		if !ok {
			continue
		}

		if currentTime >= startTime && currentTime <= endTime {
			return true
		}
		return false
	}

	return false
}
