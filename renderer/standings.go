package renderer

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"html/template"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"

	"github.com/Pastir/footballdata/viewmodel"
)

// RenderStandingsPNG рендерит HTML-шаблон в PNG-файл через headless Chromium.
// templatePath — путь к football_table_template.html.
// outputPath — куда сохранить standings.png.
func RenderStandingsPNG(
	vm viewmodel.TableViewModel,
	templatePath,
	outputPath string,
) error {
	// 1. Рендер HTML через html/template.
	htmlContent, err := renderHTML(vm, templatePath)
	if err != nil {
		return fmt.Errorf("renderHTML: %w", err)
	}

	// 2. Запись во временный HTML-файл.
	tmpFile, err := os.CreateTemp("", "standings-*.html")
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

	// 3. Скриншот .canvas через headless Chrome.
	pngBytes, err := screenshotHTML(tmpFile.Name())
	if err != nil {
		return fmt.Errorf("screenshotHTML: %w", err)
	}

	// 4. Сохранение PNG.
	if err = os.WriteFile(outputPath, pngBytes, 0644); err != nil {
		return fmt.Errorf("write PNG: %w", err)
	}

	return nil
}

// renderHTML выполняет html/template рендер и возвращает HTML-строку.
func renderHTML(
	vm viewmodel.TableViewModel,
	templatePath string,
) (string, error) {
	templateDir := filepath.Dir(templatePath)

	funcMap := template.FuncMap{
		"zoneClass": func(zone string) string {
			if zone == "" {
				return ""
			}

			return "zone-" + zone
		},
		"topHalf": func(teams []viewmodel.TeamRow) []viewmodel.TeamRow {
			if len(teams) == 0 {
				return nil
			}
			mid := (len(teams) + 1) / 2
			return teams[:mid]
		},
		"bottomHalf": func(teams []viewmodel.TeamRow) []viewmodel.TeamRow {
			if len(teams) == 0 {
				return nil
			}
			mid := (len(teams) + 1) / 2
			if mid >= len(teams) {
				return nil
			}
			return teams[mid:]
		},
		// dataURI читает файл из директории шаблона и возвращает data URI.
		// Используется для встраивания статических изображений (trophy.jpg)
		// напрямую в HTML, чтобы headless Chrome мог их отобразить
		// независимо от расположения временного HTML-файла.
		"dataURI": func(filename string) (template.URL, error) {
			filePath := filepath.Join(templateDir, filename)

			data, err := os.ReadFile(filePath)
			if err != nil {
				// Возвращаем пустую строку, чтобы не сломать рендер
				// при отсутствии файла.
				return "", nil //nolint:nilerr
			}

			mimeType := mime.TypeByExtension(filepath.Ext(filename))
			if mimeType == "" {
				mimeType = "image/jpeg"
			}

			dataURI := fmt.Sprintf(
				"data:%s;base64,%s",
				mimeType,
				base64.StdEncoding.EncodeToString(data),
			)

			return template.URL(dataURI), nil
		},
	}

	tmpl, err := template.
		New("standings").
		Funcs(funcMap).
		ParseFiles(templatePath)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer

	// ParseFiles присваивает шаблону имя файла.
	templateName := filepath.Base(templatePath)

	if err = tmpl.ExecuteTemplate(
		&buf,
		templateName,
		vm,
	); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}

// screenshotHTML запускает headless Chrome, открывает HTML через file://
// и делает PNG-скриншот всего элемента .canvas.
//
// Размер PNG определяется фактическим размером .canvas.
// Если элемент выше viewport, CaptureBeyondViewport позволяет снять
// его полностью без обрезки.
func screenshotHTML(htmlFilePath string) ([]byte, error) {
	// Получаем абсолютный путь.
	absPath, err := filepath.Abs(htmlFilePath)
	if err != nil {
		return nil, fmt.Errorf(
			"resolve html path: %w",
			err,
		)
	}

	// Корректно создаём file:// URL.
	fileURL := (&url.URL{
		Scheme: "file",
		Path:   absPath,
	}).String()

	// Настройки Chrome.
	opts := append(
		chromedp.DefaultExecAllocatorOptions[:],

		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("hide-scrollbars", true),

		// Это viewport браузера, а не размер итогового PNG.
		chromedp.WindowSize(2200, 4500),
	)

	if customPath := os.Getenv("CHROME_PATH"); customPath != "" {
		opts = append(opts, chromedp.ExecPath(customPath))
	} else if _, statErr := os.Stat("/opt/google/chrome/google-chrome"); statErr == nil {
		opts = append(opts, chromedp.ExecPath("/opt/google/chrome/google-chrome"))
	} else if _, statErr := os.Stat("/usr/bin/google-chrome-stable"); statErr == nil {
		opts = append(opts, chromedp.ExecPath("/usr/bin/google-chrome-stable"))
	} else if _, statErr := os.Stat("/usr/bin/google-chrome"); statErr == nil {
		opts = append(opts, chromedp.ExecPath("/usr/bin/google-chrome"))
	} else if _, statErr := os.Stat("/usr/bin/chromium"); statErr == nil {
		opts = append(opts, chromedp.ExecPath("/usr/bin/chromium"))
	} else if _, statErr := os.Stat("/usr/bin/chromium-browser"); statErr == nil {
		opts = append(opts, chromedp.ExecPath("/usr/bin/chromium-browser"))
	} else if _, statErr := os.Stat("/snap/bin/chromium"); statErr == nil {
		opts = append(opts, chromedp.ExecPath("/snap/bin/chromium"))
	}

	// Запускаем Chrome.
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(
		context.Background(),
		opts...,
	)
	defer cancelAlloc()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// 60 секунд более чем достаточно для локального HTML.
	ctx, cancelTimeout := context.WithTimeout(
		ctx,
		60*time.Second,
	)
	defer cancelTimeout()

	// Фактические координаты и размер .canvas.
	var box struct {
		X      float64 `json:"x"`
		Y      float64 `json:"y"`
		Width  float64 `json:"width"`
		Height float64 `json:"height"`
	}

	var pngBuf []byte

	err = chromedp.Run(
		ctx,

		// Открываем временный HTML.
		chromedp.Navigate(fileURL),

		// Ждём появления самого canvas.
		chromedp.WaitVisible(
			".canvas",
			chromedp.ByQuery,
		),

		// Ждём загрузки шрифтов и изображений.
		chromedp.Evaluate(`
			(async () => {
				if (document.fonts && document.fonts.ready) {
					await document.fonts.ready;
				}

				const images = Array.from(document.images);

				await Promise.all(
					images.map(img => {
						if (img.complete) {
							return Promise.resolve();
						}

						return new Promise(resolve => {
							img.addEventListener(
								"load",
								resolve,
								{ once: true }
							);

							img.addEventListener(
								"error",
								resolve,
								{ once: true }
							);
						});
					})
				);

				await new Promise(resolve => {
					requestAnimationFrame(() => {
						requestAnimationFrame(resolve);
					});
				});

				return true;
			})()
		`, nil),

		// Получаем реальный размер .canvas.
		chromedp.Evaluate(`
			(() => {
				const el = document.querySelector(".canvas");

				if (!el) {
					throw new Error(".canvas not found");
				}

				const r = el.getBoundingClientRect();

				return {
					x: r.left + window.scrollX,
					y: r.top + window.scrollY,
					width: r.width,
					height: r.height
				};
			})()
		`, &box),

		// Снимаем .canvas целиком.
		chromedp.ActionFunc(
			func(ctx context.Context) error {
				if box.Width <= 0 || box.Height <= 0 {
					return fmt.Errorf(
						"invalid canvas size: %.0fx%.0f",
						box.Width,
						box.Height,
					)
				}

				var captureErr error

				pngBuf, captureErr = page.
					CaptureScreenshot().
					WithFormat(
						page.CaptureScreenshotFormatPng,
					).
					WithCaptureBeyondViewport(true).
					WithClip(&page.Viewport{
						X:      box.X,
						Y:      box.Y,
						Width:  box.Width,
						Height: box.Height,
						Scale:  2,
					}).
					Do(ctx)

				if captureErr != nil {
					return fmt.Errorf(
						"capture screenshot: %w",
						captureErr,
					)
				}

				return nil
			},
		),
	)

	if err != nil {
		return nil, fmt.Errorf(
			"chromedp run: %w",
			err,
		)
	}

	if len(pngBuf) == 0 {
		return nil, fmt.Errorf(
			"chromedp returned empty PNG",
		)
	}

	return pngBuf, nil
}

// RenderTop5StandingsPNG рендерит компактный постер «Топ-5 лидеров + Зона вылета» в PNG.
func RenderTop5StandingsPNG(
	vm viewmodel.Top5TableViewModel,
	templatePath,
	outputPath string,
) error {
	htmlContent, err := renderTop5HTML(vm, templatePath)
	if err != nil {
		return fmt.Errorf("renderTop5HTML: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "top5-standings-*.html")
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

func renderTop5HTML(
	vm viewmodel.Top5TableViewModel,
	templatePath string,
) (string, error) {
	templateDir := filepath.Dir(templatePath)

	funcMap := template.FuncMap{
		"posBadgeClass": func(pos int) string {
			switch pos {
			case 1:
				return "pos-gold"
			case 2:
				return "pos-silver"
			case 3:
				return "pos-bronze"
			case 4:
				return "pos-ucl"
			case 5:
				return "pos-uel"
			default:
				return "pos-silver"
			}
		},
		"formClass": func(f string) string {
			switch f {
			case "W", "w", "В", "в":
				return "form-w"
			case "D", "d", "Н", "н":
				return "form-d"
			case "L", "l", "П", "п":
				return "form-l"
			default:
				return "form-d"
			}
		},
		"dataURI": func(filename string) (template.URL, error) {
			filePath := filepath.Join(templateDir, filename)
			data, err := os.ReadFile(filePath)
			if err != nil {
				return "", nil
			}
			mimeType := mime.TypeByExtension(filepath.Ext(filename))
			if mimeType == "" {
				mimeType = "image/jpeg"
			}
			dataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(data))
			return template.URL(dataURI), nil
		},
	}

	tmpl, err := template.
		New("top5_standings").
		Funcs(funcMap).
		ParseFiles(templatePath)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	templateName := filepath.Base(templatePath)

	if err = tmpl.ExecuteTemplate(&buf, templateName, vm); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}