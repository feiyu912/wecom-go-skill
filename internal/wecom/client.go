package wecom

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pangp/wecom-go-skill/internal/config"
)

type Client struct {
	baseURL    string
	corpID     string
	corpSecret string
	timeout    time.Duration
	debug      bool
	httpClient *http.Client
	token      string
}

type apiError struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

func New(cfg config.EffectiveConfig, debug bool) *Client {
	return &Client{
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		corpID:     cfg.CorpID,
		corpSecret: cfg.CorpSecret,
		timeout:    cfg.Timeout,
		debug:      debug,
		httpClient: &http.Client{Timeout: cfg.Timeout},
	}
}

func (c *Client) AccessToken(forceRefresh bool) (map[string]any, error) {
	if c.token != "" && !forceRefresh {
		return map[string]any{"access_token": c.token}, nil
	}

	endpoint := fmt.Sprintf("%s/cgi-bin/gettoken?corpid=%s&corpsecret=%s",
		c.baseURL,
		url.QueryEscape(c.corpID),
		url.QueryEscape(c.corpSecret),
	)

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := readAPIResponse(resp, "/cgi-bin/gettoken")
	if err != nil {
		return nil, err
	}

	token, _ := data["access_token"].(string)
	if token == "" {
		return nil, fmt.Errorf("missing access_token in response")
	}
	c.token = token
	return data, nil
}

func (c *Client) Post(path string, payload map[string]any) (map[string]any, error) {
	token, err := c.ensureToken()
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s%s?access_token=%s", c.baseURL, path, url.QueryEscape(token))
	if c.debug {
		debugPayload := map[string]any{
			"method": "POST",
			"url":    endpoint,
			"json":   payload,
		}
		pretty, _ := json.MarshalIndent(debugPayload, "", "  ")
		fmt.Println(string(pretty))
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return readAPIResponse(resp, path)
}

func (c *Client) DownloadWithCookie(downloadURL string, cookieName string, cookieValue string, writer io.Writer) error {
	trimmedURL := strings.TrimSpace(downloadURL)
	if trimmedURL == "" {
		return fmt.Errorf("download_url is required")
	}

	req, err := http.NewRequest(http.MethodGet, trimmedURL, nil)
	if err != nil {
		return err
	}
	if strings.TrimSpace(cookieName) != "" || strings.TrimSpace(cookieValue) != "" {
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", strings.TrimSpace(cookieName), strings.TrimSpace(cookieValue)))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("http error %d from download_url", resp.StatusCode)
	}
	_, err = io.Copy(writer, resp.Body)
	return err
}

func (c *Client) ensureToken() (string, error) {
	if c.token != "" {
		return c.token, nil
	}
	data, err := c.AccessToken(false)
	if err != nil {
		return "", err
	}
	token, _ := data["access_token"].(string)
	return token, nil
}

func decodeJSON(reader io.Reader) (map[string]any, error) {
	var data map[string]any
	if err := json.NewDecoder(reader).Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

func readAPIResponse(resp *http.Response, path string) (map[string]any, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return nil, emptyBodyError(path, resp.StatusCode)
	}

	var data map[string]any
	if err := json.Unmarshal(trimmed, &data); err != nil {
		return nil, fmt.Errorf("unable to decode %s response (status %d): %w; body=%q", path, resp.StatusCode, err, clipBody(trimmed))
	}
	if err := ensureOK(path, resp.StatusCode, data); err != nil {
		return nil, err
	}
	return data, nil
}

func ensureOK(path string, statusCode int, data map[string]any) error {
	if statusCode >= 400 {
		return fmt.Errorf("http error %d from %s: %s", statusCode, path, apiMessage(data))
	}

	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}

	var apiErr apiError
	if err := json.Unmarshal(raw, &apiErr); err != nil {
		return err
	}
	if apiErr.ErrCode != 0 {
		return fmt.Errorf("wecom api error %d: %s", apiErr.ErrCode, apiErr.ErrMsg)
	}
	return nil
}

func emptyBodyError(path string, statusCode int) error {
	if strings.HasPrefix(path, "/cgi-bin/todo/") {
		return fmt.Errorf("http error %d from %s: empty response body; this tenant/app does not expose the WeCom todo API at this endpoint", statusCode, path)
	}
	return fmt.Errorf("http error %d from %s: empty response body", statusCode, path)
}

func apiMessage(data map[string]any) string {
	if msg, ok := data["errmsg"].(string); ok && strings.TrimSpace(msg) != "" {
		return msg
	}
	return "request failed"
}

func clipBody(body []byte) string {
	const limit = 200
	if len(body) <= limit {
		return string(body)
	}
	return string(body[:limit]) + "..."
}
