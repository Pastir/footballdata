package publisher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	tgAPISendPhoto       = "https://api.telegram.org/bot%s/sendPhoto"
	tgAPISendDocument    = "https://api.telegram.org/bot%s/sendDocument"
	tgAPISendMessage     = "https://api.telegram.org/bot%s/sendMessage"
	tgAPISendRichMessage = "https://api.telegram.org/bot%s/sendRichMessage"
	httpTimeout          = 30 * time.Second
)

// RichBlockTableCell описывает отдельную ячейку нативной таблицы Telegram (Bot API 10.1+).
type RichBlockTableCell struct {
	Text     string `json:"text"`
	IsHeader bool   `json:"is_header,omitempty"`
	Colspan  int    `json:"colspan,omitempty"`
	Rowspan  int    `json:"rowspan,omitempty"`
	Align    string `json:"align,omitempty"`
	Valign   string `json:"valign,omitempty"`
}

// RichBlock описывает структурный блок Rich Message (таблица, заголовок, цитата).
type RichBlock struct {
	Type       string                 `json:"type"`
	Size       int                    `json:"size,omitempty"`
	Text       string                 `json:"text,omitempty"`
	IsBordered bool                   `json:"is_bordered,omitempty"`
	IsStriped  bool                   `json:"is_striped,omitempty"`
	IsCompact  bool                   `json:"is_compact,omitempty"`
	Cells      [][]RichBlockTableCell `json:"cells,omitempty"`
}

// RichMessagePayload описывает набор блоков нативного Rich Message в Telegram.
type RichMessagePayload struct {
	Blocks []RichBlock `json:"blocks"`
}

// WebAppInfo описывает WebApp ссылку для кнопки.
type WebAppInfo struct {
	URL string `json:"url"`
}

// InlineKeyboardButton описывает кнопку инлайн-клавиатуры Telegram.
type InlineKeyboardButton struct {
	Text   string      `json:"text"`
	WebApp *WebAppInfo `json:"web_app,omitempty"`
	URL    string      `json:"url,omitempty"`
}

// InlineKeyboardMarkup описывает инлайн-клавиатуру Telegram.
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// LinkPreviewOptions описывает параметры генерации превью ссылки (Bot API 7.0+).
type LinkPreviewOptions struct {
	IsDisabled       bool   `json:"is_disabled,omitempty"`
	URL              string `json:"url,omitempty"`
	PreferSmallMedia bool   `json:"prefer_small_media,omitempty"`
	PreferLargeMedia bool   `json:"prefer_large_media,omitempty"`
	ShowAboveText    bool   `json:"show_above_text,omitempty"`
}

// tgMessageResult содержит идентификатор созданного сообщения в Telegram.
type tgMessageResult struct {
	MessageID int64 `json:"message_id"`
}

// tgResponse — разбор ответа Telegram Bot API.
type tgResponse struct {
	OK          bool            `json:"ok"`
	Description string          `json:"description"`
	Result      tgMessageResult `json:"result"`
}

// SendPhoto публикует PNG-файл в Telegram-канал через Bot API (sendPhoto).
func SendPhoto(ctx context.Context, botToken, channelID, caption, pngPath string) error {
	_, err := SendPhotoWithResult(ctx, botToken, channelID, caption, pngPath)
	return err
}

// SendPhotoWithResult публикует PNG-файл в Telegram-канал и возвращает ID созданного сообщения.
func SendPhotoWithResult(ctx context.Context, botToken, channelID, caption, pngPath string) (int64, error) {
	return sendFile(ctx, botToken, channelID, caption, pngPath, "photo", tgAPISendPhoto, nil)
}

// SendPhotoWithKeyboard публикует PNG-файл с инлайн-клавиатурой (WebApp кнопками).
func SendPhotoWithKeyboard(ctx context.Context, botToken, channelID, caption, pngPath string, keyboard *InlineKeyboardMarkup) (int64, error) {
	return sendFile(ctx, botToken, channelID, caption, pngPath, "photo", tgAPISendPhoto, keyboard)
}

// SendDocument публикует файл в Telegram-канал как документ (sendDocument)
// без какого-либо сжатия и конвертации со стороны Telegram.
func SendDocument(ctx context.Context, botToken, channelID, caption, filePath string) error {
	_, err := SendDocumentWithResult(ctx, botToken, channelID, caption, filePath)
	return err
}

// SendDocumentWithResult публикует файл в Telegram-канал как документ и возвращает ID созданного сообщения.
func SendDocumentWithResult(ctx context.Context, botToken, channelID, caption, filePath string) (int64, error) {
	return sendFile(ctx, botToken, channelID, caption, filePath, "document", tgAPISendDocument, nil)
}

// SendTextMessageWithKeyboard публикует текстовое HTML-сообщение с инлайн-клавиатурой (WebApp/URL).
func SendTextMessageWithKeyboard(
	ctx context.Context,
	botToken, channelID, text string,
	keyboard *InlineKeyboardMarkup,
) (int64, error) {
	return SendTextMessageWithKeyboardAndPreview(ctx, botToken, channelID, text, keyboard, nil)
}

// SendTextMessageWithKeyboardAndPreview публикует текстовое HTML-сообщение с инлайн-клавиатурой и настройками превью ссылки (LinkPreviewOptions).
func SendTextMessageWithKeyboardAndPreview(
	ctx context.Context,
	botToken, channelID, text string,
	keyboard *InlineKeyboardMarkup,
	preview *LinkPreviewOptions,
) (int64, error) {
	reqBody := map[string]interface{}{
		"chat_id":    channelID,
		"text":       text,
		"parse_mode": "HTML",
	}
	if keyboard != nil {
		reqBody["reply_markup"] = keyboard
	}
	if preview != nil {
		reqBody["link_preview_options"] = preview
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return 0, fmt.Errorf("marshal message body: %w", err)
	}

	url := fmt.Sprintf(tgAPISendMessage, botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBytes))
	if err != nil {
		return 0, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("read response: %w", err)
	}

	var tgResp tgResponse
	if err = json.Unmarshal(respBytes, &tgResp); err != nil {
		return 0, fmt.Errorf("parse response (status %d): %w", resp.StatusCode, err)
	}

	if !tgResp.OK {
		return 0, fmt.Errorf("telegram api error: %s", tgResp.Description)
	}

	return tgResp.Result.MessageID, nil
}

// SendRichMessage публикует сообщение через нативный Rich Message API Telegram (Bot API 10.1+),
// поддерживающий нативные таблицы (InputRichBlockTable) и структурированные блоки.
func SendRichMessage(
	ctx context.Context,
	botToken, channelID string,
	richMessage RichMessagePayload,
	keyboard *InlineKeyboardMarkup,
) (int64, error) {
	reqBody := map[string]interface{}{
		"chat_id":      channelID,
		"rich_message": richMessage,
	}
	if keyboard != nil {
		reqBody["reply_markup"] = keyboard
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return 0, fmt.Errorf("marshal rich message body: %w", err)
	}

	url := fmt.Sprintf(tgAPISendRichMessage, botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBytes))
	if err != nil {
		return 0, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("read response: %w", err)
	}

	var tgResp tgResponse
	if err = json.Unmarshal(respBytes, &tgResp); err != nil {
		return 0, fmt.Errorf("parse response (status %d): %w", resp.StatusCode, err)
	}

	if !tgResp.OK {
		return 0, fmt.Errorf("telegram api error: %s", tgResp.Description)
	}

	return tgResp.Result.MessageID, nil
}

func sendFile(ctx context.Context, botToken, channelID, caption, filePath, formField, endpointFmt string, keyboard *InlineKeyboardMarkup) (int64, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	// Собираем multipart/form-data.
	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	// chat_id
	if err = w.WriteField("chat_id", channelID); err != nil {
		return 0, fmt.Errorf("write chat_id field: %w", err)
	}

	// caption (опционально)
	if caption != "" {
		caption = truncateHTMLCaption(caption, 1024)
		if err = w.WriteField("caption", caption); err != nil {
			return 0, fmt.Errorf("write caption field: %w", err)
		}
		if err = w.WriteField("parse_mode", "HTML"); err != nil {
			return 0, fmt.Errorf("write parse_mode field: %w", err)
		}
	}

	// reply_markup (опционально)
	if keyboard != nil {
		kbData, err := json.Marshal(keyboard)
		if err != nil {
			return 0, fmt.Errorf("marshal keyboard: %w", err)
		}
		if err = w.WriteField("reply_markup", string(kbData)); err != nil {
			return 0, fmt.Errorf("write reply_markup field: %w", err)
		}
	}

	// photo / document — сам файл
	part, err := w.CreateFormFile(formField, filepath.Base(filePath))
	if err != nil {
		return 0, fmt.Errorf("create form file: %w", err)
	}

	if _, err = io.Copy(part, f); err != nil {
		return 0, fmt.Errorf("copy file to form: %w", err)
	}

	if err = w.Close(); err != nil {
		return 0, fmt.Errorf("close multipart writer: %w", err)
	}

	// HTTP POST.
	url := fmt.Sprintf(endpointFmt, botToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return 0, fmt.Errorf("new request: %w", err)
	}

	req.Header.Set("Content-Type", w.FormDataContentType())

	client := &http.Client{Timeout: httpTimeout}

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("read response: %w", err)
	}

	var tgResp tgResponse
	if err = json.Unmarshal(respBody, &tgResp); err != nil {
		return 0, fmt.Errorf("parse response (status %d): %w", resp.StatusCode, err)
	}

	if !tgResp.OK {
		return 0, fmt.Errorf("telegram api error: %s", tgResp.Description)
	}

	return tgResp.Result.MessageID, nil
}

var tgTagRegex = regexp.MustCompile(`(?i)</?([a-z0-9]+)(?:\s+[^>]*)?>`)

// truncateHTMLCaption обрезает текст подписи до maxRunes с сохранением валидной HTML-разметки Telegram.
func truncateHTMLCaption(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}

	// Telegram поддерживает ограниченный набор тегов: b, strong, i, em, u, ins, s, strike, del, a, code, pre
	// Резервируем место под многоточие "..." и закрывающие теги (например "</i></b>")
	reserve := 30
	targetLen := maxRunes - reserve
	if targetLen < 10 {
		targetLen = maxRunes - 5
	}

	// Ищем наиболее подходящую границу обрезки (перенос строки или пробел)
	cutIdx := targetLen
	for i := targetLen; i > targetLen/2; i-- {
		if runes[i] == '\n' {
			cutIdx = i
			break
		}
	}
	if cutIdx == targetLen {
		for i := targetLen; i > targetLen/2; i-- {
			if runes[i] == ' ' {
				cutIdx = i
				break
			}
		}
	}

	// Не режем внутри открывающегося/закрывающегося тега <...>
	inTag := false
	for i := 0; i < cutIdx; i++ {
		if runes[i] == '<' {
			inTag = true
		} else if runes[i] == '>' {
			inTag = false
		}
	}
	if inTag {
		for cutIdx > 0 && runes[cutIdx] != '<' {
			cutIdx--
		}
	}

	sub := string(runes[:cutIdx])
	openTags := getUnclosedTags(sub)

	// Вычисляем суммарную длину суффикса закрывающих тегов и "..."
	tagsLen := 3 // для "..."
	for _, tag := range openTags {
		tagsLen += len(tag) + 3 // для "</" + tag + ">"
	}

	// Если sub + tagsLen превышает maxRunes, укорачиваем sub с конца, избегая разрезания тегов
	for cutIdx > 0 && utf8.RuneCountInString(string(runes[:cutIdx]))+tagsLen > maxRunes {
		cutIdx--
	}
	// Не оставляем незакрытый открывающийся тег на срезе
	for cutIdx > 0 && runes[cutIdx-1] == '<' {
		cutIdx--
	}

	sub = string(runes[:cutIdx])
	openTags = getUnclosedTags(sub)

	var b strings.Builder
	b.WriteString(strings.TrimRight(sub, " \t\r\n"))
	b.WriteString("...")
	for i := len(openTags) - 1; i >= 0; i-- {
		b.WriteString("</")
		b.WriteString(openTags[i])
		b.WriteString(">")
	}

	return b.String()
}

func getUnclosedTags(s string) []string {
	var stack []string
	matches := tgTagRegex.FindAllStringSubmatch(s, -1)
	for _, m := range matches {
		full := m[0]
		tagName := strings.ToLower(m[1])
		if strings.HasPrefix(full, "</") {
			for i := len(stack) - 1; i >= 0; i-- {
				if stack[i] == tagName {
					stack = append(stack[:i], stack[i+1:]...)
					break
				}
			}
		} else if !strings.HasSuffix(full, "/>") && tagName != "br" && tagName != "hr" {
			stack = append(stack, tagName)
		}
	}
	return stack
}

const tgAPIPinChatMessage = "https://api.telegram.org/bot%s/pinChatMessage"

// PinChatMessage закрепляет сообщение в Telegram-чате/канале.
func PinChatMessage(ctx context.Context, botToken, channelID string, messageID int64, disableNotification bool) error {
	reqBody := map[string]interface{}{
		"chat_id":              channelID,
		"message_id":           messageID,
		"disable_notification": disableNotification,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal pin message body: %w", err)
	}

	url := fmt.Sprintf(tgAPIPinChatMessage, botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBytes))
	if err != nil {
		return fmt.Errorf("new pin request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http post pin: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read pin response: %w", err)
	}

	var tgResp struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err = json.Unmarshal(respBytes, &tgResp); err != nil {
		return fmt.Errorf("parse pin response (status %d): %w", resp.StatusCode, err)
	}

	if !tgResp.OK {
		return fmt.Errorf("telegram pin api error: %s", tgResp.Description)
	}

	return nil
}

const tgAPISendMediaGroup = "https://api.telegram.org/bot%s/sendMediaGroup"

type tgInputMediaPhoto struct {
	Type      string `json:"type"`
	Media     string `json:"media"`
	Caption   string `json:"caption,omitempty"`
	ParseMode string `json:"parse_mode,omitempty"`
}

type tgMediaGroupResponse struct {
	OK          bool              `json:"ok"`
	Description string            `json:"description"`
	Result      []tgMessageResult `json:"result"`
}

// SendMediaGroup публикует альбом фотографий (до 10 шт.) в Telegram.
// caption прикрепляется к первой фотографии (до 1024 символов).
func SendMediaGroup(ctx context.Context, botToken, channelID string, mediaPaths []string, caption string) ([]int64, error) {
	if len(mediaPaths) < 2 || len(mediaPaths) > 10 {
		return nil, fmt.Errorf("sendMediaGroup requires between 2 and 10 media items, got %d", len(mediaPaths))
	}

	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	if err := w.WriteField("chat_id", channelID); err != nil {
		return nil, fmt.Errorf("write chat_id: %w", err)
	}

	var mediaItems []tgInputMediaPhoto
	for i, p := range mediaPaths {
		attachName := fmt.Sprintf("photo_%d", i)
		item := tgInputMediaPhoto{
			Type:  "photo",
			Media: fmt.Sprintf("attach://%s", attachName),
		}
		if i == 0 && caption != "" {
			item.Caption = truncateHTMLCaption(caption, 1024)
			item.ParseMode = "HTML"
		}
		mediaItems = append(mediaItems, item)

		f, err := os.Open(p)
		if err != nil {
			return nil, fmt.Errorf("open media file %s: %w", p, err)
		}
		defer f.Close()

		part, err := w.CreateFormFile(attachName, filepath.Base(p))
		if err != nil {
			return nil, fmt.Errorf("create form file for %s: %w", attachName, err)
		}
		if _, err = io.Copy(part, f); err != nil {
			return nil, fmt.Errorf("copy media data %s: %w", attachName, err)
		}
	}

	mediaJSON, err := json.Marshal(mediaItems)
	if err != nil {
		return nil, fmt.Errorf("marshal media array: %w", err)
	}
	if err := w.WriteField("media", string(mediaJSON)); err != nil {
		return nil, fmt.Errorf("write media field: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	url := fmt.Sprintf(tgAPISendMediaGroup, botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return nil, fmt.Errorf("new mediaGroup request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http post mediaGroup: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var tgResp tgMediaGroupResponse
	if err := json.Unmarshal(respBytes, &tgResp); err != nil {
		return nil, fmt.Errorf("parse response (status %d): %w", resp.StatusCode, err)
	}

	if !tgResp.OK {
		return nil, fmt.Errorf("telegram mediaGroup error: %s", tgResp.Description)
	}

	var msgIDs []int64
	for _, m := range tgResp.Result {
		msgIDs = append(msgIDs, m.MessageID)
	}

	return msgIDs, nil
}



