package copywriter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultTimeout = 10 * time.Second
)

// Client — интерфейс сервиса генерации текста через LLM.
type Client interface {
	GeneratePost(ctx context.Context, pack FactPack) (string, error)
}

// Config настройки для LLM API.
type Config struct {
	BaseURL string        // e.g. "https://api.deepseek.com" or "https://api.openai.com/v1"
	APIKey  string        // API токен
	Model   string        // e.g. "deepseek-chat" or "gpt-4o-mini"
	Timeout time.Duration // default 10s
}

// LLMClient реализует интерфейс Client для стандартных REST API OpenAI / DeepSeek.
type LLMClient struct {
	cfg        Config
	httpClient *http.Client
}

// NewLLMClient создаёт новый клиент к LLM API.
func NewLLMClient(cfg Config) *LLMClient {
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultTimeout
	}
	if cfg.Model == "" {
		cfg.Model = "deepseek-chat"
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.deepseek.com"
	}

	return &LLMClient{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: cfg.Timeout},
	}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// GeneratePost формирует текст поста через LLM, с автоматическим fallback при сбоях или таймауте.
func (c *LLMClient) GeneratePost(ctx context.Context, pack FactPack) (string, error) {
	// Если API ключ не задан, сразу возвращаем детерминированный fallback
	if c.cfg.APIKey == "" {
		return GenerateFallbackPost(pack), nil
	}

	userPrompt, err := BuildUserPrompt(pack)
	if err != nil {
		return GenerateFallbackPost(pack), fmt.Errorf("build prompt: %w", err)
	}

	reqBody := chatRequest{
		Model: c.cfg.Model,
		Messages: []chatMessage{
			{Role: "system", Content: SystemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.7,
		MaxTokens:   600,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return GenerateFallbackPost(pack), fmt.Errorf("marshal request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/chat/completions", c.cfg.BaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return GenerateFallbackPost(pack), fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.cfg.APIKey))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// При сетевой ошибке или таймауте возвращаем качественный fallback
		return GenerateFallbackPost(pack), nil
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil || resp.StatusCode != http.StatusOK {
		return GenerateFallbackPost(pack), nil
	}

	var chatResp chatResponse
	if err = json.Unmarshal(respBytes, &chatResp); err != nil || len(chatResp.Choices) == 0 {
		return GenerateFallbackPost(pack), nil
	}

	resultText := chatResp.Choices[0].Message.Content
	if err = ValidatePost(resultText, true); err != nil {
		// Если ответ модели не прошёл валидацию по длине, используем проверенный fallback
		return GenerateFallbackPost(pack), nil
	}

	return resultText, nil
}
