package renderer

import (
	"github.com/Pastir/footballdata/viewmodel"
)

// RenderBanterCardPNG рендерит HTML-шаблон карточки аномалии в квадратный PNG 1200x1200px через headless Chromium.
func RenderBanterCardPNG(
	vm viewmodel.BanterCardViewModel,
	templatePath,
	outputPath string,
) error {
	return renderTemplateToPNG(vm, templatePath, "banter-card-*.html", outputPath)
}

