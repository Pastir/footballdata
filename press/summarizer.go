package press

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Pastir/footballdata/config"
)

// Summarizer выполняет генерацию русскоязычной выжимки через LLM (Gemini, DeepSeek, OpenAI)
type Summarizer struct {
	llmCfg     *config.LLMConf
	httpClient *http.Client
}

// NewSummarizer инициализирует суммаризатор на базе активной конфигурации
func NewSummarizer(llmCfg *config.LLMConf) *Summarizer {
	return &Summarizer{
		llmCfg:     llmCfg,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatRequest struct {
	Model       string              `json:"model"`
	Messages    []openAIChatMessage `json:"messages"`
	Temperature float64             `json:"temperature"`
	MaxTokens   int                 `json:"max_tokens"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message openAIChatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// GenerateSummary формирует сочный интригующий пост на русском языке для Telegram
func (s *Summarizer) GenerateSummary(ctx context.Context, a Article) (string, error) {
	if s.llmCfg == nil || s.llmCfg.Providers == nil {
		return "", fmt.Errorf("llm config not initialized")
	}

	providerName := s.llmCfg.ActiveProvider
	if providerName == "" {
		providerName = "gemini"
	}

	provider, ok := s.llmCfg.Providers[providerName]
	if !ok || provider.APIKey == "" {
		return "", fmt.Errorf("provider %s not configured or api_key empty", providerName)
	}

	systemPrompt := `Ты — главный спортивный редактор и футбольный инсайдер Telegram-канала Football Data Lab.
Твоя задача — превратить зарубежную новость или инсайд в сочный, интригующий, живой пост для русскоязычной футбольной аудитории.

Правила оформления:
1. Заголовок: Сделай броский заголовок с 1-2 тематическими эмодзи (например: 🚨 ИНСАЙД, 🔥 ТРАНСФЕРНАЯ БОМБА, ⚡️ ОФИЦИАЛЬНО).
2. Тело поста: 2-3 коротких абзаца. Без канцелярита и воды. Сразу к сути: что произошло, почему это важно для клуба/игрока, какие финансовые или турнирные последствия.
3. Стиль: Уверенный, ироничный, экспертный тон (как в The Athletic или топовых футбольных пабликах).
4. Завершение: Задай короткий интригующий вопрос к подписчикам для обсуждения в комментариях.
5. Не выдумывай факты, которых нет в источнике. Не ставь хэштеги в конце.`

	userContent := fmt.Sprintf(
		"Источник: %s\nЗаголовок: %s\nКраткое содержание: %s",
		a.Source,
		a.OriginalTitle,
		a.OriginalLead,
	)

	reqPayload := openAIChatRequest{
		Model: provider.Model,
		Messages: []openAIChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userContent},
		},
		Temperature: 0.7,
		MaxTokens:   2500,
	}

	bodyBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	baseURL := strings.TrimRight(provider.BaseURL, "/")
	endpoint := fmt.Sprintf("%s/chat/completions", baseURL)

	var respBytes []byte
	var lastStatus int

	for attempt := 1; attempt <= 3; attempt++ {
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
		if err != nil {
			return "", fmt.Errorf("create http request: %w", err)
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", provider.APIKey))

		resp, err := s.httpClient.Do(httpReq)
		if err != nil {
			if attempt == 3 {
				return "", fmt.Errorf("request to %s: %w", providerName, err)
			}
			time.Sleep(2 * time.Second)
			continue
		}

		respBytes, err = io.ReadAll(resp.Body)
		lastStatus = resp.StatusCode
		resp.Body.Close()

		if lastStatus == http.StatusOK {
			break
		}

		// 503 (high demand) или 429 (rate limit) — делаем короткую паузу и повторяем
		if (lastStatus == http.StatusServiceUnavailable || lastStatus == http.StatusTooManyRequests) && attempt < 3 {
			time.Sleep(2 * time.Second)
			continue
		}

		return "", fmt.Errorf("provider %s status %d: %s", providerName, lastStatus, string(respBytes))
	}

	var chatResp openAIChatResponse
	if err = json.Unmarshal(respBytes, &chatResp); err != nil {
		return "", fmt.Errorf("unmarshal chat response: %w", err)
	}

	if len(chatResp.Choices) == 0 || chatResp.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("empty choice from provider %s", providerName)
	}

	return strings.TrimSpace(chatResp.Choices[0].Message.Content), nil
}
