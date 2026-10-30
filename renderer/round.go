package renderer

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"mime"
	"os"
	"path/filepath"

	"github.com/Pastir/footballdata/viewmodel"
)

// RenderRoundResultsPNG рендерит HTML-шаблон результатов прошедшего тура в PNG через headless Chromium.
func RenderRoundResultsPNG(
	vm viewmodel.RoundResultsViewModel,
	templatePath,
	outputPath string,
) error {
	htmlContent, err := renderRoundTemplate(vm, templatePath)
	if err != nil {
		return fmt.Errorf("renderRoundTemplate results: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "results-*.html")
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

// RenderRoundFixturesPNG рендерит HTML-шаблон расписания следующего тура в PNG через headless Chromium.
func RenderRoundFixturesPNG(
	vm viewmodel.RoundFixturesViewModel,
	templatePath,
	outputPath string,
) error {
	htmlContent, err := renderRoundTemplate(vm, templatePath)
	if err != nil {
		return fmt.Errorf("renderRoundTemplate fixtures: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "fixtures-*.html")
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

// RenderLeadersPNG рендерит HTML-шаблон лидеров сезона в PNG через headless Chromium.
func RenderLeadersPNG(
	vm viewmodel.PlayerLeadersViewModel,
	templatePath,
	outputPath string,
) error {
	htmlContent, err := renderRoundTemplate(vm, templatePath)
	if err != nil {
		return fmt.Errorf("renderRoundTemplate leaders: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "leaders-*.html")
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

func renderRoundTemplate(data any, templatePath string) (string, error) {
	templateDir := filepath.Dir(templatePath)

	funcMap := template.FuncMap{
		"dataURI": func(filename string) (template.URL, error) {
			filePath := filepath.Join(templateDir, filename)
			content, err := os.ReadFile(filePath)
			if err != nil {
				return "", nil
			}

			mimeType := mime.TypeByExtension(filepath.Ext(filename))
			if mimeType == "" {
				mimeType = "image/png"
			}

			uri := fmt.Sprintf(
				"data:%s;base64,%s",
				mimeType,
				base64.StdEncoding.EncodeToString(content),
			)
			return template.URL(uri), nil
		},
	}

	tmpl, err := template.
		New("round").
		Funcs(funcMap).
		ParseFiles(templatePath)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	templateName := filepath.Base(templatePath)
	if err = tmpl.ExecuteTemplate(&buf, templateName, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}
