package publisher

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTruncateHTMLCaption(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxRunes int
	}{
		{
			name:     "short string without tags",
			input:    "Hello World",
			maxRunes: 20,
		},
		{
			name:     "short string with closed tags",
			input:    "<b>Hello</b> <i>World</i>",
			maxRunes: 30,
		},
		{
			name:     "long string with open i tag at truncation",
			input:    "Start " + strings.Repeat("text ", 200) + "<i>open italic text that continues very long</i>",
			maxRunes: 100,
		},
		{
			name:     "multiple nested tags truncated",
			input:    "<b>Bold Start <i>Italic text that goes on and on and on and on and on</i></b>",
			maxRunes: 40,
		},
		{
			name:     "deeply nested tags at tight limit",
			input:    "<b>Bold <i>Italic <code>Code <pre>Preformatted text that is quite lengthy</pre></code></i></b>",
			maxRunes: 35,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateHTMLCaption(tt.input, tt.maxRunes)
			if utf8.RuneCountInString(got) > tt.maxRunes {
				t.Errorf("length %d exceeds maxRunes %d: %q", utf8.RuneCountInString(got), tt.maxRunes, got)
			}
			unclosed := getUnclosedTags(got)
			if len(unclosed) > 0 {
				t.Errorf("expected all tags to be closed, but got unclosed: %v in %q", unclosed, got)
			}
		})
	}
}

func TestInlineKeyboardSerialization(t *testing.T) {
	kb := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{
					Text:   "📊 Турнирная таблица",
					WebApp: &WebAppInfo{URL: "https://s.pastir.me/?tab=table"},
				},
				{
					Text:   "📅 Матчи 6-го тура",
					WebApp: &WebAppInfo{URL: "https://s.pastir.me/?tab=fixtures"},
				},
			},
		},
	}

	data, err := json.Marshal(kb)
	if err != nil {
		t.Fatalf("failed to marshal keyboard: %v", err)
	}

	expectedSubstrings := []string{
		`"text":"📊 Турнирная таблица"`,
		`"web_app":{"url":"https://s.pastir.me/?tab=table"}`,
		`"text":"📅 Матчи 6-го тура"`,
		`"web_app":{"url":"https://s.pastir.me/?tab=fixtures"}`,
	}

	out := string(data)
	for _, substr := range expectedSubstrings {
		if !strings.Contains(out, substr) {
			t.Errorf("serialized JSON missing %q: %s", substr, out)
		}
	}
}

