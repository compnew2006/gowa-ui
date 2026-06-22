package graph

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	httpClient *http.Client
}

func New(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{httpClient: httpClient}
}

func (c *Client) Get(endpoint string) (map[string]any, error) {
	var payload map[string]any
	err := c.JSONRequest(http.MethodGet, endpoint, nil, &payload)
	return payload, err
}

func (c *Client) FormPost(endpoint string, form url.Values) (map[string]any, error) {
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return c.request(req)
}

func (c *Client) JSONPost(endpoint string, body map[string]any) (map[string]any, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.request(req)
}

func (c *Client) JSONRequest(method, endpoint string, body io.Reader, out any) error {
	req, err := http.NewRequest(method, endpoint, body)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Facebook Graph API returned status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) request(req *http.Request) (map[string]any, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return payload, ResponseError(resp.StatusCode, payload)
	}
	return payload, nil
}

func ResponseError(statusCode int, payload map[string]any) error {
	if rawErr, ok := payload["error"].(map[string]any); ok {
		parts := []string{fmt.Sprintf("Facebook Graph API returned status %d", statusCode)}
		if message := nonEmpty(rawErr["message"]); message != "" {
			parts = append(parts, message)
		}
		if code := nonEmpty(rawErr["code"]); code != "" {
			parts = append(parts, "code="+code)
		}
		if subcode := nonEmpty(rawErr["error_subcode"]); subcode != "" {
			parts = append(parts, "subcode="+subcode)
		}
		if typ := nonEmpty(rawErr["type"]); typ != "" {
			parts = append(parts, "type="+typ)
		}
		return errors.New(strings.Join(parts, ": "))
	}
	return fmt.Errorf("Facebook Graph API returned status %d", statusCode)
}

func nonEmpty(value any) string {
	text := strings.TrimSpace(fmt.Sprint(value))
	if text == "<nil>" {
		return ""
	}
	return text
}
