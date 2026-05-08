package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
)

func (a *App) generateAIResponse(settings *models.ChatbotSettings, session *models.ChatbotSession, userMessage string) (string, error) {
	contextData := a.buildAIContext(settings.OrganizationID, session, userMessage)

	switch settings.AI.Provider {
	case models.AIProviderOpenAI:
		return a.generateOpenAIResponse(settings, session, userMessage, contextData)
	case models.AIProviderAnthropic:
		return a.generateAnthropicResponse(settings, session, userMessage, contextData)
	case models.AIProviderGoogle:
		return a.generateGoogleResponse(settings, session, userMessage, contextData)
	default:
		return "", fmt.Errorf("unsupported AI provider: %s", settings.AI.Provider)
	}
}

func (a *App) buildAIContext(orgID uuid.UUID, session *models.ChatbotSession, userMessage string) string {
	whatsAppAccount := ""
	if session != nil {
		whatsAppAccount = session.WhatsAppAccount
	}

	contexts, err := a.getAIContextsCached(orgID, whatsAppAccount)
	if err != nil || len(contexts) == 0 {
		return ""
	}

	var contextParts []string

	for _, ctx := range contexts {
		var content string

		switch ctx.ContextType {
		case models.ContextTypeStatic:
			content = ctx.StaticContent

		case models.ContextTypeAPI:
			content = ctx.StaticContent

			apiContent, err := a.fetchAPIContext(ctx.ApiConfig, session, userMessage)
			if err != nil {
				a.Log.Error("Failed to fetch API context", "context_name", ctx.Name, "error", err)
			} else if apiContent != "" {
				if content != "" {
					content = content + "\n\nData:\n" + apiContent
				} else {
					content = apiContent
				}
			}
		}

		if content != "" {
			contextParts = append(contextParts, fmt.Sprintf("### %s\n%s", ctx.Name, content))
		}
	}

	if len(contextParts) == 0 {
		return ""
	}

	return "## Context Information\n\n" + strings.Join(contextParts, "\n\n")
}

func (a *App) fetchAPIContext(apiConfig models.JSONB, session *models.ChatbotSession, userMessage string) (string, error) {
	if apiConfig == nil {
		return "", fmt.Errorf("API config is empty")
	}

	apiURL, ok := apiConfig["url"].(string)
	if !ok || apiURL == "" {
		return "", fmt.Errorf("API URL is required")
	}

	sessionData := models.JSONB{}
	if session != nil {
		sessionData = session.SessionData
		if sessionData == nil {
			sessionData = models.JSONB{}
		}
		sessionData["phone_number"] = session.PhoneNumber
		sessionData["user_message"] = userMessage
	}

	apiURL = a.replaceVariables(apiURL, sessionData)

	method := "GET"
	if m, ok := apiConfig["method"].(string); ok && m != "" {
		method = strings.ToUpper(m)
	}

	var bodyReader io.Reader
	if bodyTemplate, ok := apiConfig["body"].(string); ok && bodyTemplate != "" {
		bodyWithVars := a.replaceVariables(bodyTemplate, sessionData)
		bodyReader = strings.NewReader(bodyWithVars)
	}

	req, err := http.NewRequest(method, apiURL, bodyReader)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if headers, ok := apiConfig["headers"].(map[string]any); ok {
		for key, value := range headers {
			if strVal, ok := value.(string); ok {
				req.Header.Set(key, a.replaceVariables(strVal, sessionData))
			}
		}
	}

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	if responsePath, ok := apiConfig["response_path"].(string); ok && responsePath != "" {
		var jsonResp map[string]any
		if err := json.Unmarshal(respBody, &jsonResp); err == nil {
			if value := getNestedValue(jsonResp, responsePath); value != nil {
				return formatValue(value), nil
			}
		}
	}

	return string(respBody), nil
}

func (a *App) generateOpenAIResponse(settings *models.ChatbotSettings, session *models.ChatbotSession, userMessage string, contextData string) (string, error) {
	url := "https://api.openai.com/v1/chat/completions"

	messages := []map[string]string{}

	systemPrompt := settings.AI.SystemPrompt
	if contextData != "" {
		if systemPrompt != "" {
			systemPrompt = systemPrompt + "\n\n" + contextData
		} else {
			systemPrompt = contextData
		}
	}

	if systemPrompt != "" {
		messages = append(messages, map[string]string{
			"role":    "system",
			"content": systemPrompt,
		})
	}

	if settings.AI.IncludeHistory && session != nil {
		history := a.getSessionHistory(session.ID, settings.AI.HistoryLimit)
		for _, msg := range history {
			role := "user"
			if msg.Direction == models.DirectionOutgoing {
				role = "assistant"
			}
			messages = append(messages, map[string]string{
				"role":    role,
				"content": msg.Message,
			})
		}
	}

	messages = append(messages, map[string]string{
		"role":    "user",
		"content": userMessage,
	})

	payload := map[string]any{
		"model":      settings.AI.Model,
		"messages":   messages,
		"max_tokens": settings.AI.MaxTokens,
	}

	if settings.AI.Temperature > 0 {
		payload["temperature"] = settings.AI.Temperature
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+settings.AI.APIKey)

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.Unmarshal(body, &errResp)
		return "", fmt.Errorf("OpenAI API error: %s", errResp.Error.Message)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Choices) > 0 {
		return strings.TrimSpace(result.Choices[0].Message.Content), nil
	}

	return "", fmt.Errorf("no response from OpenAI")
}

func (a *App) generateAnthropicResponse(settings *models.ChatbotSettings, session *models.ChatbotSession, userMessage string, contextData string) (string, error) {
	url := "https://api.anthropic.com/v1/messages"

	messages := []map[string]string{}

	if settings.AI.IncludeHistory && session != nil {
		history := a.getSessionHistory(session.ID, settings.AI.HistoryLimit)
		for _, msg := range history {
			role := "user"
			if msg.Direction == models.DirectionOutgoing {
				role = "assistant"
			}
			messages = append(messages, map[string]string{
				"role":    role,
				"content": msg.Message,
			})
		}
	}

	messages = append(messages, map[string]string{
		"role":    "user",
		"content": userMessage,
	})

	payload := map[string]any{
		"model":      settings.AI.Model,
		"messages":   messages,
		"max_tokens": settings.AI.MaxTokens,
	}

	systemPrompt := settings.AI.SystemPrompt
	if contextData != "" {
		if systemPrompt != "" {
			systemPrompt = systemPrompt + "\n\n" + contextData
		} else {
			systemPrompt = contextData
		}
	}

	if systemPrompt != "" {
		payload["system"] = systemPrompt
	}

	if settings.AI.Temperature > 0 {
		payload["temperature"] = settings.AI.Temperature
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", settings.AI.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.Unmarshal(body, &errResp)
		return "", fmt.Errorf("anthropic API error: %s", errResp.Error.Message)
	}

	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	for _, content := range result.Content {
		if content.Type == "text" {
			return strings.TrimSpace(content.Text), nil
		}
	}

	return "", fmt.Errorf("no text response from Anthropic")
}

func (a *App) generateGoogleResponse(settings *models.ChatbotSettings, session *models.ChatbotSession, userMessage string, contextData string) (string, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent",
		settings.AI.Model)

	contents := []map[string]any{}

	if settings.AI.IncludeHistory && session != nil {
		history := a.getSessionHistory(session.ID, settings.AI.HistoryLimit)
		for _, msg := range history {
			role := "user"
			if msg.Direction == models.DirectionOutgoing {
				role = "model"
			}
			contents = append(contents, map[string]any{
				"role": role,
				"parts": []map[string]string{
					{"text": msg.Message},
				},
			})
		}
	}

	contents = append(contents, map[string]any{
		"role": "user",
		"parts": []map[string]string{
			{"text": userMessage},
		},
	})

	payload := map[string]any{
		"contents": contents,
		"generationConfig": map[string]any{
			"maxOutputTokens": settings.AI.MaxTokens,
		},
	}

	systemPrompt := settings.AI.SystemPrompt
	if contextData != "" {
		if systemPrompt != "" {
			systemPrompt = systemPrompt + "\n\n" + contextData
		} else {
			systemPrompt = contextData
		}
	}

	if systemPrompt != "" {
		payload["systemInstruction"] = map[string]any{
			"parts": []map[string]string{
				{"text": systemPrompt},
			},
		}
	}

	if settings.AI.Temperature > 0 {
		payload["generationConfig"].(map[string]any)["temperature"] = settings.AI.Temperature
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", settings.AI.APIKey)

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.Unmarshal(body, &errResp)
		return "", fmt.Errorf("google AI API error: %s", errResp.Error.Message)
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Candidates) > 0 && len(result.Candidates[0].Content.Parts) > 0 {
		return strings.TrimSpace(result.Candidates[0].Content.Parts[0].Text), nil
	}

	return "", fmt.Errorf("no response from Google AI")
}

func (a *App) getSessionHistory(sessionID uuid.UUID, limit int) []models.ChatbotSessionMessage {
	var messages []models.ChatbotSessionMessage
	a.DB.Where("session_id = ?", sessionID).
		Order("created_at DESC").
		Limit(limit).
		Find(&messages)

	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages
}
