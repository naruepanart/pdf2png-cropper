package main

import (
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"

	"image/png"

	"github.com/gen2brain/go-fitz"
)

func main() {
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	pdfFiles, err := filepath.Glob(filepath.Join(currentDir, "*.pdf"))
	if err != nil {
		log.Fatal(err)
	}

	if len(pdfFiles) == 0 {
		fmt.Println("No PDF files found in the current directory")
		return
	}

	fmt.Printf("Found %d PDF file(s) to process\n", len(pdfFiles))

	for _, pdfFile := range pdfFiles {
		if err := processPDF(pdfFile); err != nil {
			log.Printf("Error processing %s: %v", pdfFile, err)
		}
	}

	fmt.Println("All PDF files processed")
}

func processPDF(pdfFile string) error {
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

	fmt.Printf("Converting %d pages from %s\n", totalPages, pdfFile)

	for i := 0; i < totalPages; i++ {
		if err := convertPage(doc, i, baseName); err != nil {
			log.Printf("Error converting page %d: %v", i+1, err)
		}
	}

	fmt.Printf("Conversion complete for %s! Images saved to %s/\n", pdfFile, baseName)
	return nil
}

func convertPage(doc *fitz.Document, pageNum int, baseName string) error {
	img, err := doc.Image(pageNum)
	if err != nil {
		return fmt.Errorf("render page: %w", err)
	}

	croppedImg := cropTo4x3(img)
	outputFile := filepath.Join(baseName, fmt.Sprintf("page_%03d.png", pageNum+1))

	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	if err := png.Encode(file, croppedImg); err != nil {
		return fmt.Errorf("encode PNG: %w", err)
	}

	fmt.Printf("Converted page %d/%d\n", pageNum+1, doc.NumPage())
	return nil
}

func cropTo4x3(img image.Image) image.Image {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	targetWidth, targetHeight := calculateCropDimensions(width, height)
	x0, y0 := (width-targetWidth)/2, (height-targetHeight)/2
	x1, y1 := x0+targetWidth, y0+targetHeight

	// Ensure bounds
	x0, y0 = max(x0, 0), max(y0, 0)
	x1, y1 = min(x1, width), min(y1, height)

	return cropImage(img, x0, y0, x1, y1)
}

func calculateCropDimensions(width, height int) (targetWidth, targetHeight int) {
	if float64(width)/float64(height) > 4.0/3.0 {
		return height * 4 / 3, height
	}
	return width, width * 3 / 4
}

func cropImage(img image.Image, x0, y0, x1, y1 int) image.Image {
	cropped := image.NewRGBA(image.Rect(0, 0, x1-x0, y1-y0))
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			cropped.Set(x-x0, y-y0, img.At(x, y))
		}
	}
	return cropped
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
