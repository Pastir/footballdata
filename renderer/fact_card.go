package renderer

import (
	"github.com/Pastir/footballdata/viewmodel"
)

// RenderFactCardPNG рендерит HTML-шаблон фактов и серий в квадратный PNG 1200x1200px через headless Chromium.
func RenderFactCardPNG(
	vm viewmodel.FactCardViewModel,
	templatePath,
	outputPath string,
) error {
	return renderTemplateToPNG(vm, templatePath, "fact-card-*.html", outputPath)
}
