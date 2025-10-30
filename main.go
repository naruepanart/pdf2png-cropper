package main

import (
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"image/png"

	"github.com/gen2brain/go-fitz"
	"golang.org/x/image/draw"
)

const (
	targetWidth  = 1440
	targetHeight = 1080
	aspectRatio  = 4.0 / 3.0
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	specificPage := parseArgs()

	pdfFiles, err := findPDFs()
	if err != nil {
		return fmt.Errorf("finding PDFs: %w", err)
	}

	if len(pdfFiles) == 0 {
		fmt.Println("No PDF files found in current directory")
		return nil
	}

	return processFiles(pdfFiles, specificPage)
}

func parseArgs() int {
	if len(os.Args) < 2 {
		return 0
	}

	page, err := strconv.Atoi(os.Args[1])
	if err != nil || page < 1 {
		log.Printf("Warning: Invalid page number '%s', processing all pages", os.Args[1])
		return 0
	}

	fmt.Printf("Processing only page %d\n", page)
	return page
}

func findPDFs() ([]string, error) {
	entries, err := os.ReadDir(".")
	if err != nil {
		return nil, err
	}

	var pdfFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".pdf" {
			pdfFiles = append(pdfFiles, entry.Name())
		}
	}

	return pdfFiles, nil
}

func processFiles(pdfFiles []string, specificPage int) error {
	for _, file := range pdfFiles {
		if err := processPDF(file, specificPage); err != nil {
			log.Printf("Error processing %s: %v", file, err)
		}
	}
	return nil
}

func processPDF(pdfFile string, specificPage int) error {
	doc, err := fitz.New(pdfFile)
	if err != nil {
		return fmt.Errorf("opening PDF: %w", err)
	}
	defer doc.Close()

	baseName := filepath.Base(pdfFile[:len(pdfFile)-len(filepath.Ext(pdfFile))])
	if err := os.MkdirAll(baseName, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	totalPages := doc.NumPage()
	pagesToProcess := getPagesToProcess(specificPage, totalPages)

	if len(pagesToProcess) == 0 {
		return nil
	}

	fmt.Printf("Converting %s (%d pages)\n", pdfFile, len(pagesToProcess))

	for _, pageNum := range pagesToProcess {
		if err := convertPage(doc, pageNum, baseName); err != nil {
			log.Printf("Error converting page %d: %v", pageNum+1, err)
		}
	}

	return nil
}

func getPagesToProcess(specificPage, totalPages int) []int {
	if specificPage > 0 {
		if specificPage > totalPages {
			log.Printf("Page %d does not exist (PDF has only %d pages)", specificPage, totalPages)
			return nil
		}
		return []int{specificPage - 1}
	}

	pages := make([]int, totalPages)
	for i := range totalPages {
		pages[i] = i
	}
	return pages
}

func convertPage(doc *fitz.Document, pageNum int, baseName string) error {
	img, err := doc.Image(pageNum)
	if err != nil {
		return fmt.Errorf("rendering page: %w", err)
	}

	cropped := cropToAspect(img, aspectRatio)
	resized := resizeImage(cropped, targetWidth, targetHeight)

	outputFile := filepath.Join(baseName, fmt.Sprintf("page_%03d.png", pageNum+1))
	if err := savePNG(resized, outputFile); err != nil {
		return fmt.Errorf("saving image: %w", err)
	}

	return nil
}

func cropToAspect(img image.Image, ratio float64) image.Image {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	targetWidth, targetHeight := calculateCropDimensions(width, height, ratio)

	x0 := (width - targetWidth) / 2
	y0 := (height - targetHeight) / 2

	// Ensure bounds are within image
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}

	x1 := x0 + targetWidth
	y1 := y0 + targetHeight

	if x1 > width {
		x1 = width
	}
	if y1 > height {
		y1 = height
	}

	return img.(interface {
		SubImage(r image.Rectangle) image.Image
	}).SubImage(image.Rect(x0, y0, x1, y1))
}

func calculateCropDimensions(width, height int, ratio float64) (int, int) {
	currentRatio := float64(width) / float64(height)

	if currentRatio > ratio {
		// Image is wider than target ratio, crop width
		return int(float64(height) * ratio), height
	}
	// Image is taller than target ratio, crop height
	return width, int(float64(width) / ratio)
}

func resizeImage(img image.Image, width, height int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)
	return dst
}

func savePNG(img image.Image, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}
