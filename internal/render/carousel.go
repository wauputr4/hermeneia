package render

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

type CarouselRenderer struct{}

func (r CarouselRenderer) Render(ctx context.Context, content CarouselContent, outputDir string) ([]OutputFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, err
	}

	files := make([]OutputFile, 0, len(content.Slides)+1)
	for i, slide := range content.Slides {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		path := filepath.Join(outputDir, fmt.Sprintf("slide-%02d.png", i+1))
		if err := renderSlide(path, slide, i+1, len(content.Slides)); err != nil {
			return nil, err
		}
		files = append(files, OutputFile{Kind: "carousel_png", Path: path})
	}

	captionPath := filepath.Join(outputDir, "caption.txt")
	if err := os.WriteFile(captionPath, []byte(strings.TrimSpace(content.Caption)+"\n"), 0o644); err != nil {
		return nil, err
	}
	files = append(files, OutputFile{Kind: "caption_text", Path: captionPath})
	return files, nil
}

func renderSlide(path string, slide CarouselSlide, index, total int) error {
	const width = 1080
	const height = 1350

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	fill(img, image.Rect(0, 0, width, height), color.RGBA{R: 246, G: 244, B: 237, A: 255})
	fill(img, image.Rect(0, 0, width, 120), color.RGBA{R: 23, G: 35, B: 38, A: 255})
	fill(img, image.Rect(0, 120, 28, height), color.RGBA{R: 232, G: 88, B: 61, A: 255})
	fill(img, image.Rect(70, 190, 1010, 196), color.RGBA{R: 26, G: 154, B: 164, A: 255})
	fill(img, image.Rect(70, 1120, 1010, 1126), color.RGBA{R: 23, G: 35, B: 38, A: 255})

	drawText(img, 70, 54, "HERMENEIA", 4, color.RGBA{R: 246, G: 244, B: 237, A: 255}, 28)
	drawText(img, 845, 54, fmt.Sprintf("%02d/%02d", index, total), 4, color.RGBA{R: 246, G: 244, B: 237, A: 255}, 12)
	drawText(img, 70, 235, slide.Type, 5, color.RGBA{R: 232, G: 88, B: 61, A: 255}, 16)

	y := 330
	headlineScale := 8
	if len(slide.Headline) > 70 {
		headlineScale = 7
	}
	for _, line := range wrapText(slide.Headline, charsForWidth(900, headlineScale)) {
		drawText(img, 70, y, line, headlineScale, color.RGBA{R: 23, G: 35, B: 38, A: 255}, 0)
		y += headlineScale*9 + 18
	}

	y += 30
	for _, line := range wrapText(slide.Body, charsForWidth(900, 5)) {
		if y > 1050 {
			break
		}
		drawText(img, 74, y, line, 5, color.RGBA{R: 64, G: 76, B: 78, A: 255}, 0)
		y += 5*9 + 14
	}

	drawText(img, 70, 1180, "TEMPLATE: CAROUSEL/AI-NEWS-CLEAN", 4, color.RGBA{R: 64, G: 76, B: 78, A: 255}, 0)

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, img)
}

func fill(img draw.Image, rect image.Rectangle, c color.Color) {
	draw.Draw(img, rect, &image.Uniform{C: c}, image.Point{}, draw.Src)
}

func wrapText(text string, maxChars int) []string {
	words := strings.Fields(strings.ToUpper(text))
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	var current string
	for _, word := range words {
		if current == "" {
			current = word
			continue
		}
		if len(current)+1+len(word) > maxChars {
			lines = append(lines, current)
			current = word
			continue
		}
		current += " " + word
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func charsForWidth(width, scale int) int {
	charWidth := 6 * scale
	if charWidth == 0 {
		return width
	}
	return width / charWidth
}

func drawText(img draw.Image, x, y int, text string, scale int, c color.Color, maxChars int) {
	if maxChars > 0 && len(text) > maxChars {
		text = text[:maxChars]
	}
	cursor := x
	for _, r := range strings.ToUpper(text) {
		if r == ' ' {
			cursor += 4 * scale
			continue
		}
		pattern, ok := glyphs[r]
		if !ok {
			pattern = glyphs['?']
		}
		for row, bits := range pattern {
			for col, bit := range bits {
				if bit != '1' {
					continue
				}
				fill(img, image.Rect(cursor+col*scale, y+row*scale, cursor+(col+1)*scale, y+(row+1)*scale), c)
			}
		}
		cursor += 6 * scale
	}
}

var glyphs = map[rune][]string{
	'A': {"01110", "10001", "10001", "11111", "10001", "10001", "10001"},
	'B': {"11110", "10001", "10001", "11110", "10001", "10001", "11110"},
	'C': {"01111", "10000", "10000", "10000", "10000", "10000", "01111"},
	'D': {"11110", "10001", "10001", "10001", "10001", "10001", "11110"},
	'E': {"11111", "10000", "10000", "11110", "10000", "10000", "11111"},
	'F': {"11111", "10000", "10000", "11110", "10000", "10000", "10000"},
	'G': {"01111", "10000", "10000", "10011", "10001", "10001", "01111"},
	'H': {"10001", "10001", "10001", "11111", "10001", "10001", "10001"},
	'I': {"11111", "00100", "00100", "00100", "00100", "00100", "11111"},
	'J': {"00111", "00010", "00010", "00010", "00010", "10010", "01100"},
	'K': {"10001", "10010", "10100", "11000", "10100", "10010", "10001"},
	'L': {"10000", "10000", "10000", "10000", "10000", "10000", "11111"},
	'M': {"10001", "11011", "10101", "10101", "10001", "10001", "10001"},
	'N': {"10001", "11001", "10101", "10011", "10001", "10001", "10001"},
	'O': {"01110", "10001", "10001", "10001", "10001", "10001", "01110"},
	'P': {"11110", "10001", "10001", "11110", "10000", "10000", "10000"},
	'Q': {"01110", "10001", "10001", "10001", "10101", "10010", "01101"},
	'R': {"11110", "10001", "10001", "11110", "10100", "10010", "10001"},
	'S': {"01111", "10000", "10000", "01110", "00001", "00001", "11110"},
	'T': {"11111", "00100", "00100", "00100", "00100", "00100", "00100"},
	'U': {"10001", "10001", "10001", "10001", "10001", "10001", "01110"},
	'V': {"10001", "10001", "10001", "10001", "01010", "01010", "00100"},
	'W': {"10001", "10001", "10001", "10101", "10101", "11011", "10001"},
	'X': {"10001", "01010", "00100", "00100", "00100", "01010", "10001"},
	'Y': {"10001", "01010", "00100", "00100", "00100", "00100", "00100"},
	'Z': {"11111", "00001", "00010", "00100", "01000", "10000", "11111"},
	'0': {"01110", "10001", "10011", "10101", "11001", "10001", "01110"},
	'1': {"00100", "01100", "00100", "00100", "00100", "00100", "01110"},
	'2': {"01110", "10001", "00001", "00010", "00100", "01000", "11111"},
	'3': {"11110", "00001", "00001", "01110", "00001", "00001", "11110"},
	'4': {"00010", "00110", "01010", "10010", "11111", "00010", "00010"},
	'5': {"11111", "10000", "10000", "11110", "00001", "00001", "11110"},
	'6': {"01111", "10000", "10000", "11110", "10001", "10001", "01110"},
	'7': {"11111", "00001", "00010", "00100", "01000", "01000", "01000"},
	'8': {"01110", "10001", "10001", "01110", "10001", "10001", "01110"},
	'9': {"01110", "10001", "10001", "01111", "00001", "00001", "11110"},
	'-': {"00000", "00000", "00000", "11111", "00000", "00000", "00000"},
	'/': {"00001", "00010", "00010", "00100", "01000", "01000", "10000"},
	':': {"00000", "00100", "00100", "00000", "00100", "00100", "00000"},
	'.': {"00000", "00000", "00000", "00000", "00000", "01100", "01100"},
	',': {"00000", "00000", "00000", "00000", "01100", "00100", "01000"},
	'?': {"01110", "10001", "00001", "00010", "00100", "00000", "00100"},
	'#': {"01010", "11111", "01010", "01010", "11111", "01010", "00000"},
	'&': {"01100", "10010", "10100", "01000", "10101", "10010", "01101"},
	'(': {"00010", "00100", "01000", "01000", "01000", "00100", "00010"},
	')': {"01000", "00100", "00010", "00010", "00010", "00100", "01000"},
	'!': {"00100", "00100", "00100", "00100", "00100", "00000", "00100"},
}
