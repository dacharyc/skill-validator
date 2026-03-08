package judge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// defaultHTTPClient is used for all LLM API calls. It sets a timeout so
// that a hanging upstream doesn't block the caller indefinitely.
var defaultHTTPClient = &http.Client{Timeout: 30 * time.Second}

// LLMClient is the interface for making LLM API calls.
type LLMClient interface {
	// Complete sends a system prompt and user content to the LLM and returns the text response.
	Complete(ctx context.Context, systemPrompt, userContent string) (string, error)
	// Provider returns the provider name (e.g. "anthropic", "openai").
	Provider() string
	// Model returns the model identifier.
	ModelName() string
}

// ClientOptions holds configuration for creating an LLM client.
type ClientOptions struct {
	Provider          string // "anthropic" or "openai"
	APIKey            string // Required
	BaseURL           string // Optional; defaults per provider
	Model             string // Optional; defaults per provider
	MaxTokensStyle    string // "auto", "max_tokens", or "max_completion_tokens"
	MaxResponseTokens int    // Maximum tokens in the LLM response; 0 defaults to 500
}

// NewClient creates an LLMClient for the given options.
// If Model is empty, a default is chosen per provider.
// For the openai provider, BaseURL defaults to "https://api.openai.com/v1" if empty.
func NewClient(opts ClientOptions) (LLMClient, error) {
	if opts.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	maxResp := opts.MaxResponseTokens
	if maxResp <= 0 {
		maxResp = 500
	}

	switch strings.ToLower(opts.Provider) {
	case "anthropic":
		model := opts.Model
		if model == "" {
			model = "claude-sonnet-4-5-20250929"
		}
		baseURL := "https://api.anthropic.com"
		if opts.BaseURL != "" {
			baseURL = strings.TrimRight(opts.BaseURL, "/")
		}
		return &anthropicClient{apiKey: opts.APIKey, model: model, baseURL: baseURL, maxTokens: maxResp}, nil
	case "openai":
		model := opts.Model
		if model == "" {
			model = "gpt-5.2"
		}
		baseURL := opts.BaseURL
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
		baseURL = strings.TrimRight(baseURL, "/")
		return &openaiClient{apiKey: opts.APIKey, baseURL: baseURL, model: model, maxTokensStyle: opts.MaxTokensStyle, maxTokens: maxResp}, nil
	default:
		return nil, fmt.Errorf("unsupported provider %q (use \"anthropic\" or \"openai\")", opts.Provider)
	}
}

// --- Anthropic client ---

type anthropicClient struct {
	apiKey    string
	model     string
	baseURL   string
	maxTokens int
}

func (c *anthropicClient) Provider() string  { return "anthropic" }
func (c *anthropicClient) ModelName() string { return c.model }

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *anthropicClient) Complete(ctx context.Context, systemPrompt, userContent string) (string, error) {
	reqBody := anthropicRequest{
		Model:     c.model,
		MaxTokens: c.maxTokens,
		System:    systemPrompt,
		Messages: []anthropicMessage{
			{Role: "user", Content: userContent},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result anthropicResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("API error: %s", result.Error.Message)
	}

	if len(result.Content) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	return result.Content[0].Text, nil
}

// --- OpenAI-compatible client ---

type openaiClient struct {
	apiKey         string
	baseURL        string
	model          string
	maxTokensStyle string
	maxTokens      int
}

func (c *openaiClient) Provider() string  { return "openai" }
func (c *openaiClient) ModelName() string { return c.model }

type openaiRequest struct {
	Model               string          `json:"model"`
	MaxTokens           int             `json:"max_tokens,omitempty"`
	MaxCompletionTokens int             `json:"max_completion_tokens,omitempty"`
	Messages            []openaiMessage `json:"messages"`
}

// useMaxCompletionTokens reports whether the given model requires
// "max_completion_tokens" instead of the older "max_tokens" parameter.
// OpenAI's o-series reasoning models and gpt-5+ models require this.
func useMaxCompletionTokens(model string) bool {
	m := strings.ToLower(model)
	// o1, o3, o4-mini, etc.
	if strings.HasPrefix(m, "o") {
		return true
	}
	// gpt-5 and later
	if strings.HasPrefix(m, "gpt-5") {
		return true
	}
	return false
}

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *openaiClient) Complete(ctx context.Context, systemPrompt, userContent string) (string, error) {
	messages := []openaiMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userContent},
	}

	reqBody := openaiRequest{
		Model:    c.model,
		Messages: messages,
	}
	switch c.maxTokensStyle {
	case "max_completion_tokens":
		reqBody.MaxCompletionTokens = c.maxTokens
	case "max_tokens":
		reqBody.MaxTokens = c.maxTokens
	default: // "auto" or empty
		if useMaxCompletionTokens(c.model) {
			reqBody.MaxCompletionTokens = c.maxTokens
		} else {
			reqBody.MaxTokens = c.maxTokens
		}
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result openaiResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("API error: %s", result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	return result.Choices[0].Message.Content, nil
}
