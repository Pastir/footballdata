package renderer

import (
	"github.com/Pastir/footballdata/viewmodel"
)

// RenderExpressSlipPNG рендерит HTML-шаблон графического купона экспресса в PNG через headless Chromium.
func RenderExpressSlipPNG(
	vm viewmodel.ExpressSlipViewModel,
	templatePath,
	outputPath string,
) error {
	return renderTemplateToPNG(vm, templatePath, "express-slip-*.html", outputPath)
}
