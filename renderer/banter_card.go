package renderer

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/Pastir/footballdata/viewmodel"
)

// RenderBanterCardPNG рендерит HTML-шаблон карточки аномалии в квадратный PNG 1200x1200px через headless Chromium.
func RenderBanterCardPNG(
	vm viewmodel.BanterCardViewModel,
	templatePath,
	outputPath string,
) error {
	htmlContent, err := renderBanterHTML(vm, templatePath)
	if err != nil {
		return fmt.Errorf("renderBanterHTML: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "banter-card-*.html")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err = tmpFile.WriteString(htmlContent); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write temp file: %w", err)
	}

	if err = tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	pngBytes, err := screenshotHTML(tmpFile.Name())
	if err != nil {
		return fmt.Errorf("screenshotHTML: %w", err)
	}

	if err = os.WriteFile(outputPath, pngBytes, 0644); err != nil {
		return fmt.Errorf("write PNG: %w", err)
	}

	return nil
}

func renderBanterHTML(
	vm viewmodel.BanterCardViewModel,
	templatePath string,
) (string, error) {
	tmpl, err := template.
		New(filepath.Base(templatePath)).
		ParseFiles(templatePath)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, vm); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}

