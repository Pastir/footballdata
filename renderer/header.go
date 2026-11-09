package renderer

import (
	"github.com/Pastir/footballdata/viewmodel"
)

// RenderMatchHeaderPNG рендерит HTML-шаблон шапки матча тура в PNG через headless Chromium.
func RenderMatchHeaderPNG(
	vm viewmodel.MatchHeaderViewModel,
	templatePath,
	outputPath string,
) error {
	return renderTemplateToPNG(vm, templatePath, "match-header-*.html", outputPath)
}
