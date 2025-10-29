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

func main() {
	var specificPage int
	var err error

	// Check command line arguments
	if len(os.Args) >= 2 {
		// Try to parse the first argument as a page number
		specificPage, err = strconv.Atoi(os.Args[1])
		if err != nil {
			fmt.Printf("Error: Invalid page number '%s'. Please provide a valid number.\n", os.Args[1])
			fmt.Println("Usage: ./pdf2png-cropper [page_number]")
			fmt.Println("If no page number is provided, all pages will be processed.")
			return
		}

		// Validate page number (1-based for user input)
		if specificPage < 1 {
			fmt.Printf("Error: Page number must be positive, got %d\n", specificPage)
			return
		}

		fmt.Printf("Will process only page %d\n", specificPage)
	} else {
		// No arguments provided, process all pages
		specificPage = 0
		fmt.Println("Will process all pages")
	}

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
		if err := processPDF(pdfFile, specificPage); err != nil {
			log.Printf("Error processing %s: %v", pdfFile, err)
		}
	}

	fmt.Println("All PDF files processed")
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

	if specificPage > 0 {
		// Process only the specified page
		targetPage := specificPage - 1 // Adjust for 0-based indexing
		if specificPage > totalPages {
			return fmt.Errorf("page %d does not exist (PDF has only %d pages)", specificPage, totalPages)
		}

		fmt.Printf("Converting page %d from %s\n", specificPage, pdfFile)
		if err := convertPage(doc, targetPage, baseName); err != nil {
			return fmt.Errorf("error converting page %d: %v", specificPage, err)
		}
	} else {
		// Process all pages
		fmt.Printf("Converting all %d pages from %s\n", totalPages, pdfFile)
		for i := 0; i < totalPages; i++ {
			if err := convertPage(doc, i, baseName); err != nil {
				log.Printf("Error converting page %d: %v", i+1, err)
			}
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
	resizedImg := resizeTo1440x1080(croppedImg)

	outputFile := filepath.Join(baseName, fmt.Sprintf("page_%03d.png", pageNum+1))

	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	if err := png.Encode(file, resizedImg); err != nil {
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
	// Use exact 4:3 ratio calculation
	desiredRatio := 4.0 / 3.0
	currentRatio := float64(width) / float64(height)

	if currentRatio > desiredRatio {
		// Image is wider than 4:3, crop width
		targetHeight = height
		targetWidth = int(float64(height) * desiredRatio)
	} else {
		// Image is taller than 4:3, crop height
		targetWidth = width
		targetHeight = int(float64(width) / desiredRatio)
	}
	return targetWidth, targetHeight
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

func resizeTo1440x1080(img image.Image) image.Image {
	// Force exact 1440x1080 output
	targetWidth := 1440
	targetHeight := 1080

	// Create destination image
	dst := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))

	// Use high-quality scaler
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)

	return dst
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
