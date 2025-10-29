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
	specificPage := parseArgs()

	pdfFiles := findPDFs()
	if len(pdfFiles) == 0 {
		fmt.Println("No PDF files found in current directory")
		return
	}

	processFiles(pdfFiles, specificPage)
	fmt.Println("All PDF files processed")
}

func parseArgs() int {
	if len(os.Args) < 2 {
		fmt.Println("Processing all pages")
		return 0
	}

	page, err := strconv.Atoi(os.Args[1])
	if err != nil || page < 1 {
		log.Fatalf("Error: Invalid page number '%s'. Must be a positive integer.", os.Args[1])
	}

	fmt.Printf("Processing only page %d\n", page)
	return page
}

func findPDFs() []string {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.pdf"))
	if err != nil {
		log.Fatal(err)
	}

	return files
}

func processFiles(pdfFiles []string, specificPage int) {
	for _, file := range pdfFiles {
		if err := processPDF(file, specificPage); err != nil {
			log.Printf("Error processing %s: %v", file, err)
		}
	}
}

func processPDF(pdfFile string, specificPage int) error {
	doc, err := fitz.New(pdfFile)
	if err != nil {
		return err
	}
	defer doc.Close()

	baseName := filepath.Base(pdfFile[:len(pdfFile)-len(filepath.Ext(pdfFile))])
	if err := os.MkdirAll(baseName, 0755); err != nil {
		return err
	}

	totalPages := doc.NumPage()
	pagesToProcess := getPagesToProcess(specificPage, totalPages)

	fmt.Printf("Converting %d pages from %s\n", len(pagesToProcess), pdfFile)

	for _, pageNum := range pagesToProcess {
		if err := convertPage(doc, pageNum, baseName); err != nil {
			log.Printf("Error converting page %d: %v", pageNum+1, err)
		}
	}

	fmt.Printf("Conversion complete for %s! Images saved to %s/\n", pdfFile, baseName)
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
	for i := 0; i < totalPages; i++ {
		pages[i] = i
	}
	return pages
}

func convertPage(doc *fitz.Document, pageNum int, baseName string) error {
	img, err := doc.Image(pageNum)
	if err != nil {
		return fmt.Errorf("render page: %w", err)
	}

	// Use high-quality CatmullRom scaler for better image quality
	cropped := cropToAspect(img, aspectRatio)
	resized := resizeImage(cropped, targetWidth, targetHeight)

	outputFile := filepath.Join(baseName, fmt.Sprintf("page_%03d.png", pageNum+1))
	if err := savePNG(resized, outputFile); err != nil {
		return fmt.Errorf("save image: %w", err)
	}

	fmt.Printf("Converted page %d/%d\n", pageNum+1, doc.NumPage())
	return nil
}

func cropToAspect(img image.Image, ratio float64) image.Image {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	targetWidth, targetHeight := calculateCropDimensions(width, height, ratio)

	x0 := (width - targetWidth) / 2
	y0 := (height - targetHeight) / 2
	x1 := x0 + targetWidth
	y1 := y0 + targetHeight

	// Ensure bounds are within image
	x0 = max(x0, 0)
	y0 = max(y0, 0)
	x1 = min(x1, width)
	y1 = min(y1, height)

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

	// Use CatmullRom for higher quality scaling (better than ApproxBiLinear)
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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
