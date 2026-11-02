package config_test

import (
	"os"
	"testing"

	"github.com/Pastir/footballdata/config"
)

func TestConfigInitWithLLM(t *testing.T) {
	os.Args = []string{"test", "-conf", "../config.yaml"}
	cfg := config.Conf{}

	if err := cfg.Init(); err != nil {
		t.Fatalf("cfg.Init() failed: %v", err)
	}

	if cfg.LLM == nil {
		t.Fatal("cfg.LLM is nil; expected llm section to be parsed from config.yaml")
	}

	if cfg.LLM.ActiveProvider != "gemini" {
		t.Errorf("expected ActiveProvider to be 'gemini', got '%s'", cfg.LLM.ActiveProvider)
	}

	provider, exists := cfg.LLM.Providers["gemini"]
	if !exists {
		t.Fatal("expected 'gemini' provider in cfg.LLM.Providers")
	}

	if provider.APIKey == "" || provider.APIKey == "СЮДА_ВСТАВЬ_КЛЮЧ_GEMINI" {
		t.Logf("Notice: gemini APIKey is placeholder or empty")
	} else {
		t.Logf("Success: gemini APIKey is populated (length: %d chars)", len(provider.APIKey))
	}
}
