package renderer

import (
	"fmt"
	"os"

	"github.com/Pastir/footballdata/viewmodel"
)

// RenderMatchHeaderPNG рендерит HTML-шаблон шапки матча тура в PNG через headless Chromium.
func RenderMatchHeaderPNG(
	vm viewmodel.MatchHeaderViewModel,
	templatePath,
	outputPath string,
) error {
	htmlContent, err := renderRoundTemplate(vm, templatePath)
	if err != nil {
		return fmt.Errorf("renderRoundTemplate header: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "match-header-*.html")
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
