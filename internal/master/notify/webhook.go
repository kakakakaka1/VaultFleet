package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type WebhookConfig struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

type WebhookNotifier struct {
	url     string
	headers map[string]string
	client  *http.Client
}

func NewWebhookNotifier(config WebhookConfig) *WebhookNotifier {
	headers := make(map[string]string, len(config.Headers))
	for key, value := range config.Headers {
		headers[key] = value
	}

	return &WebhookNotifier{
		url:     config.URL,
		headers: headers,
		client:  &http.Client{Timeout: defaultHTTPTimeout},
	}
}

func (n *WebhookNotifier) Send(ctx context.Context, msg NotifyMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal webhook message: %w", err)
	}

	resp, err := n.sendToURL(ctx, n.url, data)
	if err != nil {
		if fallbackURL, ok := dockerHostFallbackURL(n.url); ok {
			resp, err = n.sendToURL(ctx, fallbackURL, data)
		}
		if err != nil {
			return sanitizedSendError{op: "send webhook message", err: err}
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (n *WebhookNotifier) sendToURL(ctx context.Context, targetURL string, data []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range n.headers {
		req.Header.Set(key, value)
	}
	return n.client.Do(req)
}

func (n *WebhookNotifier) Type() string {
	return "webhook"
}

func validateWebhookURL(rawURL string) error {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return fmt.Errorf("invalid webhook url")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("webhook url must use http or https")
	}
	if parsed.Host == "" {
		return fmt.Errorf("webhook url host is required")
	}
	return nil
}

func dockerHostFallbackURL(rawURL string) (string, bool) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", false
	}
	switch strings.ToLower(parsed.Hostname()) {
	case "localhost", "127.0.0.1", "::1":
	default:
		return "", false
	}
	port := parsed.Port()
	parsed.Host = "host.docker.internal"
	if port != "" {
		parsed.Host += ":" + port
	}
	return parsed.String(), true
}

func validateWebhookHeaders(headers map[string]string) error {
	for key, value := range headers {
		if !isValidHTTPHeaderName(key) {
			return fmt.Errorf("invalid webhook header name %q", key)
		}
		if strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("invalid webhook header value for %q", key)
		}
	}
	return nil
}

func isValidHTTPHeaderName(name string) bool {
	if name == "" {
		return false
	}
	for _, ch := range name {
		switch {
		case ch >= 'a' && ch <= 'z':
		case ch >= 'A' && ch <= 'Z':
		case ch >= '0' && ch <= '9':
		case strings.ContainsRune("!#$%&'*+-.^_`|~", ch):
		default:
			return false
		}
	}
	return true
}
