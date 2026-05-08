package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
)

func (a *App) matchFlowTrigger(orgID uuid.UUID, accountName, messageText string) *models.ChatbotFlow {
	flows, err := a.getChatbotFlowsCached(orgID)
	if err != nil {
		a.Log.Error("Failed to fetch chatbot flows", "error", err)
		return nil
	}

	messageLower := strings.ToLower(messageText)

	for _, flow := range flows {
		for _, keyword := range flow.TriggerKeywords {
			if strings.Contains(messageLower, strings.ToLower(keyword)) {
				return &flow
			}
		}
	}
	return nil
}

func (a *App) startFlow(account *models.WhatsAppAccount, session *models.ChatbotSession, contact *models.Contact, flow *models.ChatbotFlow) {
	a.Log.Info("Starting flow", "flow_id", flow.ID, "flow_name", flow.Name, "contact", contact.PhoneNumber, "num_steps", len(flow.Steps))

	for i, step := range flow.Steps {
		a.Log.Info("Flow step", "index", i, "step_name", step.StepName, "step_order", step.StepOrder, "message_type", step.MessageType)
	}

	session.CurrentFlowID = &flow.ID
	session.CurrentStep = ""
	session.StepRetries = 0
	session.SessionData = models.JSONB{
		"_flow_id":   flow.ID.String(),
		"_flow_name": flow.Name,
	}
	a.DB.Save(session)

	if flow.InitialMessage != "" {
		if err := a.sendAndSaveTextMessage(account, contact, flow.InitialMessage); err != nil {
			a.Log.Error("Failed to send flow initial message", "error", err, "contact", contact.PhoneNumber)
		}
		a.logSessionMessage(session.ID, models.DirectionOutgoing, flow.InitialMessage, "flow_start")
	}

	if len(flow.Steps) > 0 {
		firstStep := &flow.Steps[0]
		a.Log.Info("Sending first step", "step_name", firstStep.StepName, "message_type", firstStep.MessageType, "message", firstStep.Message)
		session.CurrentStep = firstStep.StepName
		a.DB.Model(session).Update("current_step", firstStep.StepName)

		a.sendStepWithSkipCheck(account, session, contact, firstStep, flow, nil)
	} else {
		a.completeFlow(account, session, contact, flow)
	}
}

func (a *App) processFlowResponse(account *models.WhatsAppAccount, session *models.ChatbotSession, contact *models.Contact, userInput string, buttonID string, flowResponseData map[string]any) {
	flow, err := a.getChatbotFlowByIDCached(account.OrganizationID, *session.CurrentFlowID)
	if err != nil {
		a.Log.Error("Failed to load flow", "error", err)
		a.exitFlow(session)
		return
	}

	if a.handleFlowCancellation(account, session, contact, flow, userInput) {
		return
	}

	currentStep, currentStepIndex := findFlowStepByName(flow.Steps, session.CurrentStep)
	if currentStep == nil {
		a.Log.Error("Current step not found", "step_name", session.CurrentStep)
		a.exitFlow(session)
		return
	}

	if a.validateFlowStepInput(account, session, contact, currentStep, userInput, &buttonID) {
		return
	}

	a.storeFlowStepResponse(session, currentStep, userInput, buttonID)
	a.storeFlowResponseData(session, flowResponseData)

	nextStepName := resolveNextStepNameAfterFlowResponse(flow, currentStep, currentStepIndex, userInput, buttonID)
	a.advanceFlowToNextStep(account, session, contact, flow, nextStepName)
}

func (a *App) handleFlowCancellation(account *models.WhatsAppAccount, session *models.ChatbotSession, contact *models.Contact, flow *models.ChatbotFlow, userInput string) bool {
	userInputLower := strings.ToLower(userInput)
	for _, cancelKeyword := range flow.CancelKeywords {
		if strings.Contains(userInputLower, strings.ToLower(cancelKeyword)) {
			if err := a.sendAndSaveTextMessage(account, contact, "Flow cancelled."); err != nil {
				a.Log.Error("Failed to send flow cancel message", "error", err, "contact", contact.PhoneNumber)
			}
			a.logSessionMessage(session.ID, models.DirectionOutgoing, "Flow cancelled.", "flow_cancel")
			a.exitFlow(session)
			return true
		}
	}
	return false
}

func findFlowStepByName(steps []models.ChatbotFlowStep, stepName string) (*models.ChatbotFlowStep, int) {
	for i := range steps {
		if steps[i].StepName == stepName {
			return &steps[i], i
		}
	}
	return nil, -1
}

func (a *App) validateFlowStepInput(
	account *models.WhatsAppAccount,
	session *models.ChatbotSession,
	contact *models.Contact,
	currentStep *models.ChatbotFlowStep,
	userInput string,
	buttonID *string,
) bool {
	if a.handleInvalidRegexInput(account, session, contact, currentStep, userInput, *buttonID) {
		return true
	}
	if a.handleInvalidButtonSelection(account, session, contact, currentStep, userInput, buttonID) {
		return true
	}
	return false
}

func (a *App) handleInvalidRegexInput(
	account *models.WhatsAppAccount,
	session *models.ChatbotSession,
	contact *models.Contact,
	currentStep *models.ChatbotFlowStep,
	userInput string,
	buttonID string,
) bool {
	if currentStep.ValidationRegex == "" || buttonID != "" {
		return false
	}

	re, err := regexp.Compile(currentStep.ValidationRegex)
	if err == nil && !re.MatchString(userInput) {
		session.StepRetries++
		if currentStep.RetryOnInvalid && session.StepRetries < currentStep.MaxRetries {
			a.DB.Model(session).Update("step_retries", session.StepRetries)
			errorMsg := currentStep.ValidationError
			if errorMsg == "" {
				errorMsg = "Invalid input. Please try again."
			}
			if err := a.sendAndSaveTextMessage(account, contact, errorMsg); err != nil {
				a.Log.Error("Failed to send validation error", "error", err, "contact", contact.PhoneNumber)
			}
			a.logSessionMessage(session.ID, models.DirectionOutgoing, errorMsg, currentStep.StepName+"_retry")
			return true
		}
		a.Log.Warn("Max retries exceeded", "step", currentStep.StepName)
	}
	return false
}

func (a *App) handleInvalidButtonSelection(
	account *models.WhatsAppAccount,
	session *models.ChatbotSession,
	contact *models.Contact,
	currentStep *models.ChatbotFlowStep,
	userInput string,
	buttonID *string,
) bool {
	shouldValidateButtons := len(currentStep.Buttons) > 0 &&
		(currentStep.InputType == models.InputTypeButton || currentStep.InputType == models.InputTypeSelect || *buttonID != "")
	if !shouldValidateButtons {
		return false
	}

	isValidButton := false
	userInputLower := strings.ToLower(userInput)

	for i, btn := range currentStep.Buttons {
		if btnMap, ok := btn.(map[string]any); ok {
			btnID, _ := btnMap["id"].(string)
			btnTitle, _ := btnMap["title"].(string)

			if btnID == "" {
				btnID = fmt.Sprintf("btn_%d", i+1)
			}

			if *buttonID != "" && *buttonID == btnID {
				isValidButton = true
				break
			}
			if strings.ToLower(btnTitle) == userInputLower || btnID == userInput {
				isValidButton = true
				if *buttonID == "" {
					*buttonID = btnID
				}
				break
			}
		}
	}

	if !isValidButton {
		session.StepRetries++
		a.Log.Debug("Invalid button selection", "buttonID", *buttonID, "userInput", userInput, "step", currentStep.StepName, "retries", session.StepRetries)
		a.DB.Model(session).Update("step_retries", session.StepRetries)

		maxRetries := currentStep.MaxRetries
		if maxRetries == 0 {
			maxRetries = 3
		}

		if session.StepRetries >= maxRetries {
			a.Log.Warn("Max button retries exceeded, closing conversation", "step", currentStep.StepName)
			if err := a.sendAndSaveTextMessage(account, contact, "Sorry, we couldn't continue. Please try again later."); err != nil {
				a.Log.Error("Failed to send max retries message", "error", err, "contact", contact.PhoneNumber)
			}
			a.exitFlow(session)
			a.closeSession(session)
			return true
		}

		a.sendStepMessage(account, session, contact, currentStep)
		return true
	}
	return false
}

func (a *App) storeFlowStepResponse(session *models.ChatbotSession, currentStep *models.ChatbotFlowStep, userInput string, buttonID string) {
	if currentStep.StoreAs == "" {
		return
	}
	sessionData := session.SessionData
	if sessionData == nil {
		sessionData = models.JSONB{}
	}
	if buttonID != "" {
		sessionData[currentStep.StoreAs] = buttonID
		sessionData[currentStep.StoreAs+"_title"] = userInput
	} else {
		sessionData[currentStep.StoreAs] = userInput
	}
	a.DB.Model(session).Update("session_data", sessionData)
	session.SessionData = sessionData
}

func (a *App) storeFlowResponseData(session *models.ChatbotSession, flowResponseData map[string]any) {
	if len(flowResponseData) == 0 {
		return
	}

	sessionData := session.SessionData
	if sessionData == nil {
		sessionData = models.JSONB{}
	}
	for key, value := range flowResponseData {
		sessionData[key] = value
		a.Log.Debug("Stored flow response field", "key", key, "value", value)
	}
	sessionData["_flow_response"] = flowResponseData
	a.DB.Model(session).Update("session_data", sessionData)
	session.SessionData = sessionData
	a.Log.Info("Stored WhatsApp Flow response in session", "fields", len(flowResponseData))
}

func resolveNextStepNameAfterFlowResponse(
	flow *models.ChatbotFlow,
	currentStep *models.ChatbotFlowStep,
	currentStepIndex int,
	userInput string,
	buttonID string,
) string {
	nextStepName := currentStep.NextStep
	if nextStepName == "" && currentStepIndex+1 < len(flow.Steps) {
		nextStepName = flow.Steps[currentStepIndex+1].StepName
	}

	if len(currentStep.ConditionalNext) == 0 {
		return nextStepName
	}
	if buttonID != "" {
		if next, ok := currentStep.ConditionalNext[buttonID].(string); ok {
			return next
		}
		if next, ok := currentStep.ConditionalNext[userInput].(string); ok {
			return next
		}
		if defaultNext, ok := currentStep.ConditionalNext["default"].(string); ok {
			return defaultNext
		}
		return nextStepName
	}
	if next, ok := currentStep.ConditionalNext[userInput].(string); ok {
		return next
	}
	if defaultNext, ok := currentStep.ConditionalNext["default"].(string); ok {
		return defaultNext
	}
	return nextStepName
}

func (a *App) advanceFlowToNextStep(
	account *models.WhatsAppAccount,
	session *models.ChatbotSession,
	contact *models.Contact,
	flow *models.ChatbotFlow,
	nextStepName string,
) {
	if nextStepName == "" {
		a.completeFlow(account, session, contact, flow)
		return
	}

	nextStep, _ := findFlowStepByName(flow.Steps, nextStepName)
	if nextStep == nil {
		a.Log.Warn("Next step not found, completing flow", "next_step", nextStepName)
		a.completeFlow(account, session, contact, flow)
		return
	}

	a.DB.Model(session).Updates(map[string]any{
		"current_step": nextStep.StepName,
		"step_retries": 0,
	})

	a.Log.Info("Moving to next step", "nextStep", nextStep.StepName, "skipCondition", nextStep.SkipCondition, "sessionData", session.SessionData)
	a.sendStepWithSkipCheck(account, session, contact, nextStep, flow, nil)
}

func (a *App) completeFlow(account *models.WhatsAppAccount, session *models.ChatbotSession, contact *models.Contact, flow *models.ChatbotFlow) {
	a.Log.Info("Completing flow", "flow_id", flow.ID, "session_id", session.ID)

	if flow.CompletionMessage != "" {
		message := a.replaceVariables(flow.CompletionMessage, session.SessionData)
		if err := a.sendAndSaveTextMessage(account, contact, message); err != nil {
			a.Log.Error("Failed to send flow completion message", "error", err, "contact", contact.PhoneNumber)
		}
		a.logSessionMessage(session.ID, models.DirectionOutgoing, message, "flow_complete")
	}

	if flow.OnCompleteAction == "webhook" && len(flow.CompletionConfig) > 0 {
		go a.sendFlowCompletionWebhook(flow, session, contact)
	}

	now := time.Now()
	a.DB.Model(session).Updates(map[string]any{
		"current_step": "",
		"status":       models.SessionStatusCompleted,
		"completed_at": now,
	})

	a.ClearContactChatbotTracking(contact.ID)
}

func (a *App) sendFlowCompletionWebhook(flow *models.ChatbotFlow, session *models.ChatbotSession, contact *models.Contact) {
	config := flow.CompletionConfig

	webhookURL, ok := config["url"].(string)
	if !ok || webhookURL == "" {
		a.Log.Error("Webhook URL not configured", "flow_id", flow.ID)
		return
	}

	webhookURL = a.replaceVariables(webhookURL, session.SessionData)

	method := "POST"
	if m, ok := config["method"].(string); ok && m != "" {
		method = strings.ToUpper(m)
	}

	payload := map[string]any{
		"flow_id":      flow.ID.String(),
		"flow_name":    flow.Name,
		"session_id":   session.ID.String(),
		"phone_number": session.PhoneNumber,
		"contact_id":   contact.ID.String(),
		"contact_name": contact.ProfileName,
		"session_data": session.SessionData,
		"completed_at": time.Now().UTC().Format(time.RFC3339),
	}

	var bodyReader io.Reader
	if bodyTemplate, ok := config["body"].(string); ok && bodyTemplate != "" {
		bodyWithVars := a.replaceVariables(bodyTemplate, session.SessionData)
		bodyReader = strings.NewReader(bodyWithVars)
	} else {
		jsonPayload, err := json.Marshal(payload)
		if err != nil {
			a.Log.Error("Failed to marshal webhook payload", "error", err)
			return
		}
		bodyReader = bytes.NewReader(jsonPayload)
	}

	req, err := http.NewRequest(method, webhookURL, bodyReader)
	if err != nil {
		a.Log.Error("Failed to create webhook request", "error", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Whatomate-Webhook/1.0")

	if headers, ok := config["headers"].(map[string]any); ok {
		for key, value := range headers {
			if strVal, ok := value.(string); ok {
				req.Header.Set(key, a.replaceVariables(strVal, session.SessionData))
			}
		}
	}

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		a.Log.Error("Webhook request failed", "error", err, "url", webhookURL)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		a.Log.Info("Webhook sent successfully",
			"flow_id", flow.ID,
			"session_id", session.ID,
			"status", resp.StatusCode,
		)
	} else {
		a.Log.Error("Webhook returned error",
			"flow_id", flow.ID,
			"session_id", session.ID,
			"status", resp.StatusCode,
			"response", string(body),
		)
	}
}

func (a *App) sendStepWithSkipCheck(account *models.WhatsAppAccount, session *models.ChatbotSession, contact *models.Contact, step *models.ChatbotFlowStep, flow *models.ChatbotFlow, skippedSteps map[string]bool) {
	if skippedSteps == nil {
		skippedSteps = make(map[string]bool)
	}
	if skippedSteps[step.StepName] {
		a.Log.Warn("Skip loop detected, completing flow", "step", step.StepName)
		a.completeFlow(account, session, contact, flow)
		return
	}

	sessionData := session.SessionData
	if sessionData == nil {
		sessionData = models.JSONB{}
	}

	if a.shouldSkipStep(step, sessionData) {
		a.Log.Info("Skipping step", "step", step.StepName, "condition", step.SkipCondition)
		skippedSteps[step.StepName] = true

		nextStepName := step.NextStep
		if nextStepName == "" {
			for i, s := range flow.Steps {
				if s.StepName == step.StepName && i+1 < len(flow.Steps) {
					nextStepName = flow.Steps[i+1].StepName
					break
				}
			}
		}

		if nextStepName == "" {
			a.completeFlow(account, session, contact, flow)
			return
		}

		var nextStep *models.ChatbotFlowStep
		for i := range flow.Steps {
			if flow.Steps[i].StepName == nextStepName {
				nextStep = &flow.Steps[i]
				break
			}
		}

		if nextStep == nil {
			a.Log.Warn("Next step not found after skip, completing flow", "next_step", nextStepName)
			a.completeFlow(account, session, contact, flow)
			return
		}

		session.CurrentStep = nextStep.StepName
		a.DB.Model(session).Update("current_step", nextStep.StepName)

		a.sendStepWithSkipCheck(account, session, contact, nextStep, flow, skippedSteps)
		return
	}

	a.sendStepMessage(account, session, contact, step)

	if step.InputType == models.InputTypeNone {

		nextStepName := step.NextStep
		if nextStepName == "" {
			for i, s := range flow.Steps {
				if s.StepName == step.StepName && i+1 < len(flow.Steps) {
					nextStepName = flow.Steps[i+1].StepName
					break
				}
			}
		}

		if nextStepName == "" {
			a.completeFlow(account, session, contact, flow)
			return
		}

		var nextStep *models.ChatbotFlowStep
		for i := range flow.Steps {
			if flow.Steps[i].StepName == nextStepName {
				nextStep = &flow.Steps[i]
				break
			}
		}

		if nextStep == nil {
			a.Log.Warn("Next step not found after no-input step, completing flow", "next_step", nextStepName)
			a.completeFlow(account, session, contact, flow)
			return
		}

		session.CurrentStep = nextStep.StepName
		a.DB.Model(session).Update("current_step", nextStep.StepName)

		a.sendStepWithSkipCheck(account, session, contact, nextStep, flow, skippedSteps)
	}
}

func (a *App) sendStepMessage(account *models.WhatsAppAccount, session *models.ChatbotSession, contact *models.Contact, step *models.ChatbotFlowStep) {
	var message string

	a.Log.Debug("sendStepMessage called", "step", step.StepName, "message_type", step.MessageType, "input_config", step.InputConfig)

	switch step.MessageType {
	case models.FlowStepTypeAPIFetch:
		apiResp, err := a.fetchApiResponse(step.ApiConfig, session.SessionData, step.Message)
		if err != nil {
			a.Log.Error("Failed to fetch API response", "error", err, "step", step.StepName)
			if fallback, ok := step.ApiConfig["fallback_message"].(string); ok && fallback != "" {
				message = processTemplate(fallback, session.SessionData)
			} else if step.Message != "" {
				message = processTemplate(step.Message, session.SessionData)
			} else {
				message = "Sorry, there was an error processing your request."
			}
			if err := a.sendAndSaveTextMessage(account, contact, message); err != nil {
				a.Log.Error("Failed to send API error message", "error", err, "contact", contact.PhoneNumber)
			}
		} else {
			message = apiResp.Message

			if apiResp.MappedData != nil {
				for k, v := range apiResp.MappedData {
					session.SessionData[k] = v
				}
				a.DB.Model(session).Update("session_data", session.SessionData)
			}

			if len(apiResp.Buttons) > 0 {
				if err := a.sendAndSaveInteractiveButtons(account, contact, message, apiResp.Buttons); err != nil {
					a.Log.Error("Failed to send API response buttons", "error", err, "contact", contact.PhoneNumber)
				}
			} else {
				if err := a.sendAndSaveTextMessage(account, contact, message); err != nil {
					a.Log.Error("Failed to send API response message", "error", err, "contact", contact.PhoneNumber)
				}
			}
		}
		a.logSessionMessage(session.ID, models.DirectionOutgoing, message, step.StepName)

	case models.FlowStepTypeButtons:
		message = processTemplate(step.Message, session.SessionData)
		if len(step.Buttons) > 0 {
			buttons := make([]map[string]any, 0, len(step.Buttons))
			for _, btn := range step.Buttons {
				if btnMap, ok := btn.(map[string]any); ok {
					buttons = append(buttons, btnMap)
				}
			}
			if err := a.sendAndSaveInteractiveButtons(account, contact, message, buttons); err != nil {
				a.Log.Error("Failed to send buttons", "error", err, "contact", contact.PhoneNumber)
			}
		} else {
			if err := a.sendAndSaveTextMessage(account, contact, message); err != nil {
				a.Log.Error("Failed to send step message", "error", err, "contact", contact.PhoneNumber)
			}
		}
		a.logSessionMessage(session.ID, models.DirectionOutgoing, message, step.StepName)

	case models.FlowStepTypeTransfer:
		message = processTemplate(step.Message, session.SessionData)
		if message != "" {
			if err := a.sendAndSaveTextMessage(account, contact, message); err != nil {
				a.Log.Error("Failed to send transfer message", "error", err, "contact", contact.PhoneNumber)
			}
			a.logSessionMessage(session.ID, models.DirectionOutgoing, message, step.StepName)
		}

		var teamID *uuid.UUID
		var notes string
		if step.TransferConfig != nil {
			if teamIDStr, ok := step.TransferConfig["team_id"].(string); ok && teamIDStr != "" && teamIDStr != "_general" {
				if parsedID, err := uuid.Parse(teamIDStr); err == nil {
					teamID = &parsedID
				}
			}
			if n, ok := step.TransferConfig["notes"].(string); ok {
				notes = processTemplate(n, session.SessionData)
			}
		}

		if teamID != nil {
			a.createTransferToTeam(account, contact, *teamID, notes, models.TransferSourceFlow)
		} else {
			a.createTransferToQueue(account, contact, models.TransferSourceFlow)
		}

		a.exitFlow(session)
		return

	case models.FlowStepTypeWhatsAppFlow:
		a.Log.Debug("Processing WhatsApp Flow step", "step", step.StepName, "input_config", step.InputConfig)
		message = processTemplate(step.Message, session.SessionData)

		var flowID, headerText, ctaText string
		if step.InputConfig != nil {
			if fid, ok := step.InputConfig["whatsapp_flow_id"].(string); ok {
				flowID = fid
				a.Log.Debug("Found WhatsApp Flow ID", "flow_id", flowID)
			}
			if header, ok := step.InputConfig["flow_header"].(string); ok {
				headerText = processTemplate(header, session.SessionData)
			}
			if cta, ok := step.InputConfig["flow_cta"].(string); ok {
				ctaText = cta
			}
		}

		if flowID == "" {
			a.Log.Error("WhatsApp Flow step missing flow ID", "step", step.StepName)
			if err := a.sendAndSaveTextMessage(account, contact, message); err != nil {
				a.Log.Error("Failed to send fallback message", "error", err, "contact", contact.PhoneNumber)
			}
		} else {
			var waFlow models.WhatsAppFlow
			firstScreen := ""
			if err := a.DB.Where("meta_flow_id = ?", flowID).First(&waFlow).Error; err != nil {
				a.Log.Debug("Could not find WhatsApp Flow in database, using default screen", "meta_flow_id", flowID)
			} else {
				if len(waFlow.Screens) > 0 {
					if screenMap, ok := waFlow.Screens[0].(map[string]any); ok {
						if screenID, ok := screenMap["id"].(string); ok {
							firstScreen = screenID
							a.Log.Debug("Found first screen from flow", "first_screen", firstScreen)
						}
					}
				}
				if firstScreen == "" && waFlow.FlowJSON != nil {
					if screens, ok := waFlow.FlowJSON["screens"].([]interface{}); ok && len(screens) > 0 {
						if screenMap, ok := screens[0].(map[string]any); ok {
							if screenID, ok := screenMap["id"].(string); ok {
								firstScreen = screenID
								a.Log.Debug("Found first screen from flow_json", "first_screen", firstScreen)
							}
						}
					}
				}
			}

			flowToken := fmt.Sprintf("chatbot_%s_%s_%d", session.ID.String(), step.StepName, time.Now().UnixNano())
			a.Log.Debug("Sending WhatsApp Flow message", "flow_id", flowID, "first_screen", firstScreen, "cta", ctaText)

			if err := a.sendAndSaveFlowMessage(account, contact, flowID, headerText, message, ctaText, flowToken, firstScreen); err != nil {
				a.Log.Error("Failed to send WhatsApp Flow message", "error", err, "contact", contact.PhoneNumber, "flow_id", flowID)
			}
		}
		a.logSessionMessage(session.ID, models.DirectionOutgoing, message, step.StepName)

	default:
		a.Log.Debug("Unhandled message type, falling back to text", "message_type", step.MessageType, "step", step.StepName)
		message = processTemplate(step.Message, session.SessionData)
		if err := a.sendAndSaveTextMessage(account, contact, message); err != nil {
			a.Log.Error("Failed to send step message", "error", err, "contact", contact.PhoneNumber)
		}
		a.logSessionMessage(session.ID, models.DirectionOutgoing, message, step.StepName)
	}
}

type ApiResponse struct {
	Message      string
	Buttons      []map[string]any
	MappedData   map[string]any
	ResponseData map[string]any
}

func (a *App) fetchApiResponse(apiConfig models.JSONB, sessionData models.JSONB, messageTemplate string) (*ApiResponse, error) {
	if apiConfig == nil {
		return nil, fmt.Errorf("API config is empty")
	}

	apiURL, ok := apiConfig["url"].(string)
	if !ok || apiURL == "" {
		return nil, fmt.Errorf("API URL is required")
	}

	apiURL = processTemplate(apiURL, sessionData)

	method := "GET"
	if m, ok := apiConfig["method"].(string); ok && m != "" {
		method = strings.ToUpper(m)
	}

	var bodyReader io.Reader
	if bodyTemplate, ok := apiConfig["body"].(string); ok && bodyTemplate != "" {
		bodyWithVars := processTemplate(bodyTemplate, sessionData)
		bodyReader = strings.NewReader(bodyWithVars)
	}

	req, err := http.NewRequest(method, apiURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if headers, ok := apiConfig["headers"].(map[string]any); ok {
		for key, value := range headers {
			if strVal, ok := value.(string); ok {
				req.Header.Set(key, processTemplate(strVal, sessionData))
			}
		}
	}

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	limitReader := io.LimitReader(resp.Body, 1024*1024)
	respBody, err := io.ReadAll(limitReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var jsonResp map[string]any
	if err := json.Unmarshal(respBody, &jsonResp); err != nil {
		return &ApiResponse{Message: string(respBody)}, nil
	}

	result := &ApiResponse{
		ResponseData: jsonResp,
	}

	if responseMapping, ok := apiConfig["response_mapping"].(map[string]any); ok {
		mappingStrings := make(map[string]string)
		for varName, path := range responseMapping {
			if pathStr, ok := path.(string); ok {
				mappingStrings[varName] = pathStr
			}
		}
		result.MappedData = extractResponseMapping(jsonResp, mappingStrings)

		for k, v := range result.MappedData {
			sessionData[k] = v
		}
	}

	if messageTemplate != "" {
		result.Message = processTemplate(messageTemplate, sessionData)
	} else if msg, ok := jsonResp["message"].(string); ok {
		result.Message = msg
	} else {
		result.Message = string(respBody)
	}

	if buttons, ok := jsonResp["buttons"].([]interface{}); ok && len(buttons) > 0 {
		result.Buttons = make([]map[string]any, 0, len(buttons))
		for _, btn := range buttons {
			if btnMap, ok := btn.(map[string]any); ok {
				normalizedBtn := make(map[string]any)

				if id, ok := btnMap["id"].(string); ok {
					normalizedBtn["id"] = id
				}

				if value, ok := btnMap["value"].(string); ok {
					normalizedBtn["title"] = value
				} else if title, ok := btnMap["title"].(string); ok {
					normalizedBtn["title"] = title
				}

				if normalizedBtn["id"] != nil && normalizedBtn["title"] != nil {
					result.Buttons = append(result.Buttons, normalizedBtn)
				}
			}
		}
	}

	return result, nil
}

func (a *App) shouldSkipStep(step *models.ChatbotFlowStep, sessionData map[string]any) bool {
	if step.SkipCondition == "" {
		a.Log.Debug("No skip condition for step", "step", step.StepName)
		return false
	}
	a.Log.Info("Evaluating skip condition", "step", step.StepName, "condition", step.SkipCondition, "sessionData", sessionData)
	result := evaluateExpression(step.SkipCondition, sessionData)
	a.Log.Info("Skip condition result", "step", step.StepName, "result", result)
	return result
}

func evaluateExpression(expr string, data map[string]any) bool {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return false
	}

	if !needsExpressionParsing(expr) {
		return evaluateSingleCondition(expr, data)
	}

	rpn, ok := parseExpressionToRPN(expr)
	if !ok {
		return false
	}

	return evaluateExpressionRPN(rpn, data)
}
