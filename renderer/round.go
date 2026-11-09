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
	return renderTemplateToPNG(vm, templatePath, "results-*.html", outputPath)
}

// RenderRoundFixturesPNG рендерит HTML-шаблон расписания следующего тура в PNG через headless Chromium.
func RenderRoundFixturesPNG(
	vm viewmodel.RoundFixturesViewModel,
	templatePath,
	outputPath string,
) error {
	return renderTemplateToPNG(vm, templatePath, "fixtures-*.html", outputPath)
}

// RenderLeadersPNG рендерит HTML-шаблон лидеров сезона в PNG через headless Chromium.
func RenderLeadersPNG(
	vm viewmodel.PlayerLeadersViewModel,
	templatePath,
	outputPath string,
) error {
	return renderTemplateToPNG(vm, templatePath, "leaders-*.html", outputPath)
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
