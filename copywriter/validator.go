package copywriter

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	MaxCaptionLength = 1024 // Максимальная длина подписи к фото в Telegram
	MaxMessageLength = 4096 // Максимальная длина отдельного сообщения в Telegram
)

// ValidatePost проверяет сгенерированный текст на корректность и ограничения Telegram.
func ValidatePost(text string, isCaption bool) error {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return fmt.Errorf("post text is empty")
	}

	length := utf8.RuneCountInString(trimmed)

	if isCaption && length > MaxCaptionLength {
		return fmt.Errorf("caption length %d exceeds Telegram limit %d", length, MaxCaptionLength)
	}

	if !isCaption && length > MaxMessageLength {
		return fmt.Errorf("message length %d exceeds Telegram limit %d", length, MaxMessageLength)
	}

	return nil
}
