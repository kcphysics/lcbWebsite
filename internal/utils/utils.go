package utils

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"kcphysics/aiLCBWebsite/internal/models" // Import the models package
)

// readMembersCSV reads member data from a CSV file.
func ReadMembersCSV(filePath string) ([]models.Member, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("unable to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 // Allow variable number of fields

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("unable to read CSV records: %w", err)
	}

	var members []models.Member
	// Skip header row
	for i, record := range records {
		if i == 0 {
			continue
		}
		if len(record) >= 3 {
			members = append(members, models.Member{
				Name:    record[0],
				Section: record[1],
				DayJob:  record[2],
			})
		}
	}
	return members, nil
}

// loadInstrumentSections loads instrument data from JSON files.
func LoadInstrumentSections(dirPath string, allMembers []models.Member) (map[string]models.Instrument, error) {
	instrumentMap := make(map[string]models.Instrument)

	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read instrument directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(dirPath, file.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read instrument JSON file %s: %w", filePath, err)
		}

		var instrument models.Instrument
		if err := json.Unmarshal(content, &instrument); err != nil {
			return nil, fmt.Errorf("failed to unmarshal instrument JSON from %s: %w", filePath, err)
		}

		// Populate members and URLs for each instrument section
		instrument.URL = strings.ToLower(strings.ReplaceAll(instrument.Name, " ", "_")) + ".html"
		var sectionMembers []models.Member
		for _, member := range allMembers {
			if member.Section == instrument.Name {
				sectionMembers = append(sectionMembers, member)
			}
		}
		instrument.Members = sectionMembers
		instrumentMap[instrument.Name] = instrument
	}
	return instrumentMap, nil
}

// GenerateColorScheme extracts dominant colors from an image and updates CSS variables.
func GenerateColorScheme(imagePath, cssVarsPath string) {
	fmt.Printf("Generating color scheme from %s...\n", imagePath)
	file, err := os.Open(imagePath)
	if err != nil {
		log.Fatalf("Failed to open image: %v", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		log.Fatalf("Failed to decode image: %v", err)
	}

	// Custom dominant color extraction
	colors := getDominantColors(img, 5)

	var cssVars []string
	for i, c := range colors {
		r, g, b, _ := c.RGBA()
		// Convert to 0-255 range
		r, g, b = r/257, g/257, b/257
		cssVars = append(cssVars, fmt.Sprintf("--color-%d: rgb(%d, %d, %d);", i+1, r, g, b))
	}

	// Read existing CSS, replace color variables section
	cssContent, err := os.ReadFile(cssVarsPath)
	if err != nil {
		log.Fatalf("Failed to read CSS file: %v", err)
	}

	cssString := string(cssContent)
	startMarker := "/* Color variables will be generated here */"
	endMarker := "/* End Color variables */" // Add an end marker for easier replacement

	// If markers exist, replace content between them
	if strings.Contains(cssString, startMarker) {
		if strings.Contains(cssString, endMarker) {
			startIndex := strings.Index(cssString, startMarker) + len(startMarker)
			endIndex := strings.Index(cssString, endMarker)
			cssString = cssString[:startIndex] + "\n" + strings.Join(cssVars, "\n") + "\n    " + cssString[endIndex:]
		} else {
			// If only start marker, append to it
			cssString = strings.Replace(cssString, startMarker, startMarker+"\n    "+strings.Join(cssVars, "\n")+"\n    "+endMarker, 1)
		}
	} else {
		// If no markers, just append to the root
		cssString = strings.Replace(cssString, ":root {", ":root {\n    "+strings.Join(cssVars, "\n"), 1)
	}

	if err := os.WriteFile(cssVarsPath, []byte(cssString), 0644); err != nil {
		log.Fatalf("Failed to write CSS variables to file: %v", err)
	}
	fmt.Printf("Generated color scheme and updated %s\n", cssVarsPath)
}

// copyStaticAssets copies static assets from src to dst.
func CopyStaticAssets(src, dst string) {
	fmt.Printf("Copying static assets from %s to %s...\n", src, dst)
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		destFile, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, srcFile)
		return err
	})

	if err != nil {
		log.Fatalf("Failed to copy static assets: %v", err)
	}
	fmt.Println("Static assets copied successfully.")
}

// copyAssets copies assets from src to dst.
func CopyAssets(src, dst string) {
	fmt.Printf("Copying assets from %s to %s...\n", src, dst)
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		destFile, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, srcFile)
		return err
	})

	if err != nil {
		log.Fatalf("Failed to copy assets: %v", err)
	}
	fmt.Println("Assets copied successfully.")
}

// getDominantColors extracts dominant colors from an image.
func getDominantColors(img image.Image, count int) []color.Color {
	bounds := img.Bounds()
	colorCounts := make(map[color.Color]int)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.At(x, y)
			colorCounts[c]++
		}
	}

	// Sort colors by frequency
	type colorFreq struct {
		color color.Color
		freq  int
	}
	var freqs []colorFreq
	for c, f := range colorCounts {
		freqs = append(freqs, colorFreq{color: c, freq: f})
	}

	// Simple sort, could be optimized for large images
	for i := 0; i < len(freqs); i++ {
		for j := i + 1; j < len(freqs); j++ {
			if freqs[i].freq < freqs[j].freq {
				freqs[i], freqs[j] = freqs[j], freqs[i]
			}
		}
	}

	// Return top 'count' colors
	var dominantColors []color.Color
	for i := 0; i < count && i < len(freqs); i++ {
		dominantColors = append(dominantColors, freqs[i].color)
	}
	return dominantColors
}
