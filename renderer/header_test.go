package renderer_test

import (
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/Pastir/footballdata/renderer"
	"github.com/Pastir/footballdata/viewmodel"
)

func TestRenderMatchHeaderPNG(t *testing.T) {
	templatePath := filepath.Join("..", "template", "match_header_template.html")
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		t.Skip("template not found, skipping")
	}

	vm := viewmodel.MatchHeaderViewModel{
		LeagueName:   "АПЛ",
		RoundTitle:   "ТУР 6",
		BadgeTitle:   "МАТЧ ТУРА",
		HomeTeamName: "БРЕНТФОРД",
		HomeTeamCode: "BRE",
		AwayTeamName: "ЛИВЕРПУЛЬ",
		AwayTeamCode: "LIV",
		KickoffTime:  "14:00",
		Venue:        "Gtech Community Stadium",
		Timezone:     "MSK",
	}

	outputPath := filepath.Join(t.TempDir(), "test_match_header.png")
	err := renderer.RenderMatchHeaderPNG(vm, templatePath, outputPath)
	if err != nil {
		t.Fatalf("RenderMatchHeaderPNG failed: %v", err)
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("output file does not exist: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("output file is empty")
	}
}

func TestSerieAHeaderIntegration(t *testing.T) {
	p8 := filepath.Join("..", "template", "headers", "8.png")
	f8, err := os.Open(p8)
	if err != nil {
		t.Skipf("template/headers/8.png not found: %v", err)
	}
	defer f8.Close()

	cfg8, _, err := image.DecodeConfig(f8)
	if err != nil {
		t.Fatalf("failed to decode 8.png config: %v", err)
	}
	t.Logf("template/headers/8.png dimensions: %dx%d", cfg8.Width, cfg8.Height)

	// Source candidate generated with official Serie A logo & panoramic spread
	srcCandidate := "/home/user/.gemini/antigravity-ide/brain/4793a2f7-6e9b-4e79-be68-726c62b1d156/serie_a_panoramic_official_1791990986976.jpg"
	if _, err := os.Stat(srcCandidate); os.IsNotExist(err) {
		t.Skipf("source candidate not found: %v", err)
	}

	fSrc, err := os.Open(srcCandidate)
	if err != nil {
		t.Fatalf("failed to open candidate: %v", err)
	}
	defer fSrc.Close()

	srcImg, _, err := image.Decode(fSrc)
	if err != nil {
		t.Fatalf("failed to decode candidate: %v", err)
	}

	srcB := srcImg.Bounds()
	t.Logf("Candidate dimensions: %dx%d", srcB.Dx(), srcB.Dy())

	targetW := cfg8.Width
	targetH := cfg8.Height
	if targetW == 0 || targetH == 0 {
		targetW = 2043
		targetH = 770
	}

	// We crop top and bottom slightly (from 3% to 97%) to keep all slogans and bottom laser line
	yMin := float64(srcB.Min.Y) + float64(srcB.Dy())*0.03
	yMax := float64(srcB.Min.Y) + float64(srcB.Dy())*0.97
	srcHRange := yMax - yMin

	_ = color.RGBA{}
	_ = png.FormatError("")
	t.Logf("uniformScale: %.3f", float64(targetH)/srcHRange)

	xSplit1Src := 850.0
	xSplit2Src := 960.0

	// Target positions for the split:
	// Left section (0 .. xSplit1Src) scales uniformly by uniformScale
	uniformScale := float64(targetH) / srcHRange
	xSplit1Dst := xSplit1Src * uniformScale

	// Right section (xSplit2Src .. srcW) keeps the exact same uniformScale anchored to the right:
	srcW := float64(srcB.Dx())
	rightSecWidthDst := (srcW - xSplit2Src) * uniformScale
	xSplit2Dst := float64(targetW) - rightSecWidthDst

	dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))

	for dy := 0; dy < targetH; dy++ {
		sy := yMin + (float64(dy)/float64(targetH))*srcHRange
		iy := int(sy)
		fy := sy - float64(iy)
		iy1 := iy + 1
		if iy1 >= srcB.Max.Y {
			iy1 = srcB.Max.Y - 1
		}

		for dx := 0; dx < targetW; dx++ {
			fdx := float64(dx)
			var sx float64
			if fdx <= xSplit1Dst {
				// Left section: 100% uniform scaling (zero distortion)
				sx = float64(srcB.Min.X) + fdx/uniformScale
			} else if fdx >= xSplit2Dst {
				// Right section: 100% uniform scaling anchored to the right (zero distortion)
				sx = float64(srcB.Min.X) + xSplit2Src + (fdx-xSplit2Dst)/uniformScale
			} else {
				// Middle empty stadium gap (between x=850 and x=960): smooth interpolation of dark stadium
				tMid := (fdx - xSplit1Dst) / (xSplit2Dst - xSplit1Dst)
				sx = float64(srcB.Min.X) + xSplit1Src + tMid*(xSplit2Src-xSplit1Src)
			}

			ix := int(sx)
			fx := sx - float64(ix)
			ix1 := ix + 1
			if ix1 >= srcB.Max.X {
				ix1 = srcB.Max.X - 1
			}

			c00 := srcImg.At(ix, iy)
			c10 := srcImg.At(ix1, iy)
			c01 := srcImg.At(ix, iy1)
			c11 := srcImg.At(ix1, iy1)

			r00, g00, b00, _ := c00.RGBA()
			r10, g10, b10, _ := c10.RGBA()
			r01, g01, b01, _ := c01.RGBA()
			r11, g11, b11, _ := c11.RGBA()

			interp := func(v00, v10, v01, v11 uint32) uint8 {
				top := float64(v00)*(1.0-fx) + float64(v10)*fx
				bot := float64(v01)*(1.0-fx) + float64(v11)*fx
				val := top*(1.0-fy) + bot*fy
				return uint8(val / 257)
			}

			dst.SetRGBA(dx, dy, color.RGBA{
				R: interp(r00, r10, r01, r11),
				G: interp(g00, g10, g01, g11),
				B: interp(b00, b10, b01, b11),
				A: 255,
			})
		}
	}

	out384Path := filepath.Join("..", "template", "headers", "384.png")
	outF, err := os.Create(out384Path)
	if err != nil {
		t.Fatalf("failed to create 384.png: %v", err)
	}
	defer outF.Close()

	if err := png.Encode(outF, dst); err != nil {
		t.Fatalf("failed to encode 384.png: %v", err)
	}
	t.Logf("Successfully created %s (%dx%d) with uniform aspect ratio", out384Path, targetW, targetH)
}
