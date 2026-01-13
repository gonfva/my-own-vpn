// +build ignore

package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

// Simple icon generator for system tray icons
// Creates solid colored shield-like icons at 32x32 and 64x64

func main() {
	icons := map[string]color.RGBA{
		"disconnected": {R: 128, G: 128, B: 128, A: 255}, // Grey
		"connecting":   {R: 255, G: 165, B: 0, A: 255},   // Orange
		"connected":    {R: 0, G: 200, B: 0, A: 255},     // Green
		"error":        {R: 220, G: 53, B: 69, A: 255},   // Red
	}

	sizes := []int{32, 64}
	assetsDir := "assets"

	for name, c := range icons {
		for _, size := range sizes {
			img := createShieldIcon(size, c)
			filename := filepath.Join(assetsDir, name+".png")
			if size == 64 {
				filename = filepath.Join(assetsDir, name+"_64.png")
			}

			f, err := os.Create(filename)
			if err != nil {
				panic(err)
			}
			if err := png.Encode(f, img); err != nil {
				f.Close()
				panic(err)
			}
			f.Close()
		}
	}
}

func createShieldIcon(size int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// Fill with transparent background
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.Set(x, y, color.Transparent)
		}
	}

	// Draw a simple shield shape
	centerX := size / 2
	topMargin := size / 8
	bottomMargin := size / 8
	sideMargin := size / 6

	for y := topMargin; y < size-bottomMargin; y++ {
		// Calculate width at this y position (narrower at bottom)
		progress := float64(y-topMargin) / float64(size-topMargin-bottomMargin)
		var halfWidth int
		if progress < 0.6 {
			halfWidth = centerX - sideMargin
		} else {
			// Taper toward bottom
			taper := (progress - 0.6) / 0.4
			halfWidth = int(float64(centerX-sideMargin) * (1 - taper*0.7))
		}

		for x := centerX - halfWidth; x <= centerX+halfWidth; x++ {
			if x >= 0 && x < size {
				img.Set(x, y, c)
			}
		}
	}

	return img
}
