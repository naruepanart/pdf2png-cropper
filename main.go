package main

import (
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"

	"github.com/gen2brain/go-fitz"
	"image/png"
)

func main() {
	// Input PDF file
	inputPDF := "input.pdf"
	
	// Output directory for images
	outputDir := "output"

	// Create output directory if it doesn't exist
	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		log.Fatal(err)
	}

	// Open PDF document
	doc, err := fitz.New(inputPDF)
	if err != nil {
		log.Fatal(err)
	}
	defer doc.Close()

	// Get number of pages
	totalPages := doc.NumPage()
	fmt.Printf("Converting %d pages from %s\n", totalPages, inputPDF)

	// Convert each page to image
	for i := 0; i < totalPages; i++ {
		// Render page to image with 300 DPI
		img, err := doc.Image(i)
		if err != nil {
			log.Printf("Error rendering page %d: %v", i+1, err)
			continue
		}

		// Crop image to 4:3 aspect ratio
		croppedImg := cropTo4x3(img)

		// Create output file
		outputFile := filepath.Join(outputDir, fmt.Sprintf("page_%03d.png", i+1))
		file, err := os.Create(outputFile)
		if err != nil {
			log.Printf("Error creating file for page %d: %v", i+1, err)
			continue
		}

		// Save as PNG
		err = png.Encode(file, croppedImg)
		file.Close()

		if err != nil {
			log.Printf("Error saving page %d: %v", i+1, err)
			continue
		}

		fmt.Printf("Converted page %d/%d\n", i+1, totalPages)
	}

	fmt.Printf("Conversion complete! Images saved to %s/\n", outputDir)
}

// cropTo4x3 crops an image to 4:3 aspect ratio from the center
func cropTo4x3(img image.Image) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Calculate target dimensions for 4:3 aspect ratio
	var targetWidth, targetHeight int

	// Determine whether to use width or height as the limiting dimension
	if float64(width)/float64(height) > 4.0/3.0 {
		// Image is wider than 4:3, so use height as reference
		targetHeight = height
		targetWidth = height * 4 / 3
	} else {
		// Image is taller than 4:3, so use width as reference
		targetWidth = width
		targetHeight = width * 3 / 4
	}

	// Calculate crop coordinates to center the crop
	x0 := (width - targetWidth) / 2
	y0 := (height - targetHeight) / 2
	x1 := x0 + targetWidth
	y1 := y0 + targetHeight

	// Ensure coordinates are within bounds
	if x0 < 0 { x0 = 0 }
	if y0 < 0 { y0 = 0 }
	if x1 > width { x1 = width }
	if y1 > height { y1 = height }

	// Perform the crop
	cropped := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			cropped.Set(x-x0, y-y0, img.At(x, y))
		}
	}

	return cropped
}