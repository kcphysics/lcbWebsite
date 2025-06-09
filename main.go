package main

import (
	"encoding/csv" // Import for CSV parsing
	"encoding/json"
	"fmt"
	"html/template"
	"image"        // Import for color.Color
	"image/color"  // Re-add image/color
	_ "image/jpeg" // Import for JPEG decoding
	_ "image/png"  // Import for PNG decoding
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"context" // Import for context
	"flag"    // Import the flag package
	"mime"    // Import for mime type detection

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Member represents a single member from the CSV
type Member struct {
	Name    string
	Section string
	DayJob  string
}

// Instrument represents an instrument section
type Instrument struct {
	Name        string
	Description string
	ImagePath   string
	Members     []Member
	URL         string // New field for the page URL
}

var allMembers []Member
var instrumentSections map[string]Instrument

const (
	imagePath         = "assets/LCB-Saxes-Logos-v1-Primary.png"
	cssVarsPath       = "static/css/style.css"
	carouselDir       = "assets/carousel_photos"
	outputDir         = "public"
	staticAssetsDir   = "static"
	googleCalendarURL = "https://calendar.google.com/calendar/embed?src=85b15423a562f0de4c47230084c68a284b5481593ec04121be2ef24f5a2926a2%40group.calendar.google.com&ctz=America%2FNew_York"
	googleMapURL      = "https://www.google.com/maps/embed/v1/place?q=320%20Corley%20Mill%20Rd%2C%20Lexington%2C%20SC%2029072"
)

func loadInstrumentSections(dirPath string, allMembers []Member) (map[string]Instrument, error) {
	instrumentMap := make(map[string]Instrument)

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

		var instrument Instrument
		if err := json.Unmarshal(content, &instrument); err != nil {
			return nil, fmt.Errorf("failed to unmarshal instrument JSON from %s: %w", filePath, err)
		}

		// Populate members and URLs for each instrument section
		instrument.URL = strings.ToLower(strings.ReplaceAll(instrument.Name, " ", "_")) + ".html"
		var sectionMembers []Member
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

func main() {
	buildCmd := flag.NewFlagSet("build", flag.ExitOnError)

	deployCmd := flag.NewFlagSet("deploy", flag.ExitOnError)
	deployBucket := deployCmd.String("bucket", "", "S3 bucket name to deploy to")

	if len(os.Args) < 2 {
		fmt.Println("No command provided. Defaulting to 'build'.")
		runBuild()
		return
	}

	switch os.Args[1] {
	case "build":
		buildCmd.Parse(os.Args[2:])
		runBuild()
	case "deploy":
		deployCmd.Parse(os.Args[2:])
		if *deployBucket == "" {
			fmt.Println("Error: S3 bucket name is required for deploy command. Use --bucket <bucket-name>")
			os.Exit(1)
		}
		runDeploy(*deployBucket)
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		fmt.Println("Usage: go run main.go [command]")
		fmt.Println("Commands:")
		fmt.Println("  build  - Generates the static site")
		fmt.Println("  deploy - Deploys the static site to an S3 bucket")
		fmt.Println("           Usage: go run main.go deploy --bucket <bucket-name>")
		os.Exit(1)
	}
}

func runBuild() {
	fmt.Println("Starting static site generation...")

	// Read members data first, as it's needed for instrument sections
	allMembers, err := readMembersCSV("members.csv")
	if err != nil {
		log.Fatalf("Failed to read members CSV: %v", err)
	}

	instrumentSections, err = loadInstrumentSections("data/instruments", allMembers)
	if err != nil {
		log.Fatalf("Failed to load instrument sections: %v", err)
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Generate color scheme and update CSS
	generateColorScheme()

	// Copy static assets
	copyStaticAssets(staticAssetsDir, outputDir)
	copyAssets("assets", filepath.Join(outputDir, "assets"))

	// Generate index.html
	generateIndexPage(outputDir)
	generateAboutPage(outputDir)
	generateMembershipPage(outputDir)
	generateRehearsalsPage(outputDir)
	generateCalendarPage(outputDir)
	generateErrorPage(outputDir)

	// Generate instrument pages
	for _, section := range instrumentSections {
		generateInstrumentPage(outputDir, section)
	}

	fmt.Println("Static site generation complete!")
}

func readMembersCSV(filePath string) ([]Member, error) {
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

	var members []Member
	// Skip header row
	for i, record := range records {
		if i == 0 {
			continue
		}
		if len(record) >= 3 {
			members = append(members, Member{
				Name:    record[0],
				Section: record[1],
				DayJob:  record[2],
			})
		}
	}
	return members, nil
}

func generateInstrumentPage(outputDir string, instrument Instrument) {
	tmpl, err := template.ParseFiles(
		"templates/instrument.html",
		"templates/partials/header.html",
		"templates/partials/navbar.html",
		"templates/partials/footer.html",
		"templates/partials/instrument_dropdown.html", // Add this line
	)
	if err != nil {
		log.Fatalf("Failed to parse instrument template for %s: %v", instrument.Name, err)
	}

	outputPath := filepath.Join(outputDir, strings.ToLower(strings.ReplaceAll(instrument.Name, " ", "_"))+".html")
	file, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("Failed to create output file for %s: %v", instrument.Name, err)
	}
	defer file.Close()

	data := struct {
		Title              string
		Instrument         Instrument
		InstrumentSections map[string]Instrument
	}{
		Title:              "Lexington Community Band",
		Instrument:         instrument,
		InstrumentSections: instrumentSections,
	}
	if err := tmpl.Execute(file, data); err != nil {
		log.Fatalf("Failed to execute template for %s: %v", instrument.Name, err)
	}
	fmt.Printf("Generated %s\n", outputPath)
}

func generateCalendarPage(outputDir string) {
	tmpl, err := template.ParseFiles(
		"templates/calendar.html",
		"templates/partials/header.html",
		"templates/partials/navbar.html",
		"templates/partials/footer.html",
		"templates/partials/instrument_dropdown.html", // Add this line
	)
	if err != nil {
		log.Fatalf("Failed to parse calendar template: %v", err)
	}

	outputPath := filepath.Join(outputDir, "calendar.html")
	file, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()

	data := struct {
		Title              string
		CalendarURL        string
		InstrumentSections map[string]Instrument
	}{
		Title:              "Lexington Community Band",
		CalendarURL:        googleCalendarURL,
		InstrumentSections: instrumentSections,
	}

	if err := tmpl.Execute(file, data); err != nil {
		log.Fatalf("Failed to execute template: %v", err)
	}
	fmt.Printf("Generated %s\n", outputPath)
}

func generateAboutPage(outputDir string) {
	tmpl, err := template.ParseFiles(
		"templates/about.html",
		"templates/partials/header.html",
		"templates/partials/navbar.html",
		"templates/partials/footer.html",
		"templates/partials/instrument_dropdown.html", // Add this line
	)
	if err != nil {
		log.Fatalf("Failed to parse about template: %v", err)
	}

	outputPath := filepath.Join(outputDir, "about.html")
	file, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()

	data := struct {
		Title              string
		InstrumentSections map[string]Instrument
	}{
		Title:              "Lexington Community Band",
		InstrumentSections: instrumentSections,
	}

	if err := tmpl.Execute(file, data); err != nil {
		log.Fatalf("Failed to execute template: %v", err)
	}
	fmt.Printf("Generated %s\n", outputPath)
}

func generateMembershipPage(outputDir string) {
	tmpl, err := template.ParseFiles(
		"templates/membership.html",
		"templates/partials/header.html",
		"templates/partials/navbar.html",
		"templates/partials/footer.html",
		"templates/partials/instrument_dropdown.html", // Add this line
	)
	if err != nil {
		log.Fatalf("Failed to parse membership template: %v", err)
	}

	outputPath := filepath.Join(outputDir, "membership.html")
	file, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()

	data := struct {
		Title              string
		InstrumentSections map[string]Instrument
	}{
		Title:              "Lexington Community Band",
		InstrumentSections: instrumentSections,
	}

	if err := tmpl.Execute(file, data); err != nil {
		log.Fatalf("Failed to execute template: %v", err)
	}
	fmt.Printf("Generated %s\n", outputPath)
}

func generateRehearsalsPage(outputDir string) {
	tmpl, err := template.ParseFiles(
		"templates/rehearsals.html",
		"templates/partials/header.html",
		"templates/partials/navbar.html",
		"templates/partials/footer.html",
		"templates/partials/instrument_dropdown.html", // Add this line
	)
	if err != nil {
		log.Fatalf("Failed to parse rehearsals template: %v", err)
	}

	outputPath := filepath.Join(outputDir, "rehearsals.html")
	file, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()

	data := struct {
		Title              string
		InstrumentSections map[string]Instrument
	}{
		Title:              "Lexington Community Band",
		InstrumentSections: instrumentSections,
	}

	if err := tmpl.Execute(file, data); err != nil {
		log.Fatalf("Failed to execute template: %v", err)
	}
	fmt.Printf("Generated %s\n", outputPath)
}

func generateColorScheme() {
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

func copyStaticAssets(src, dst string) {
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

func copyAssets(src, dst string) {
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

func runDeploy(bucketName string) {
	fmt.Printf("Starting deployment to S3 bucket: %s...\n", bucketName)

	// First, generate the site
	runBuild()

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Failed to load AWS SDK config: %v", err)
	}

	uploader := manager.NewUploader(s3.NewFromConfig(cfg))

	err = filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(outputDir, path)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", path, err)
		}
		defer file.Close()

		// Determine Content-Type
		contentType := mime.TypeByExtension(filepath.Ext(path))
		if contentType == "" {
			contentType = "application/octet-stream" // Default if type is unknown
		}

		_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
			Bucket:      aws.String(bucketName),
			Key:         aws.String(relPath),
			Body:        file,
			ContentType: aws.String(contentType),
		})
		if err != nil {
			return fmt.Errorf("failed to upload %s to S3: %w", relPath, err)
		}

		fmt.Printf("Uploaded %s to s3://%s/%s\n", relPath, bucketName, relPath)
		return nil
	})

	if err != nil {
		log.Fatalf("Deployment failed: %v", err)
	}

	fmt.Println("Deployment complete!")
}

func generateErrorPage(outputDir string) {
	tmpl, err := template.ParseFiles(
		"templates/error.html",
		"templates/partials/header.html",
		"templates/partials/navbar.html",
		"templates/partials/footer.html",
		"templates/partials/instrument_dropdown.html",
	)
	if err != nil {
		log.Fatalf("Failed to parse error template: %v", err)
	}

	outputPath := filepath.Join(outputDir, "error.html")
	file, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()

	data := struct {
		Title              string
		InstrumentSections map[string]Instrument
	}{
		Title:              "Lexington Community Band",
		InstrumentSections: instrumentSections,
	}

	if err := tmpl.Execute(file, data); err != nil {
		log.Fatalf("Failed to execute template: %v", err)
	}
	fmt.Printf("Generated %s\n", outputPath)
}

func generateIndexPage(outputDir string) {
	tmpl, err := template.ParseFiles(
		"templates/index.html",
		"templates/partials/header.html",
		"templates/partials/navbar.html",
		"templates/partials/footer.html",
		"templates/partials/instrument_dropdown.html",
	)
	if err != nil {
		log.Fatalf("Failed to parse index template: %v", err)
	}

	outputPath := filepath.Join(outputDir, "index.html")
	file, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()

	data := struct {
		Title              string
		InstrumentSections map[string]Instrument
	}{
		Title:              "Lexington Community Band",
		InstrumentSections: instrumentSections,
	}

	if err := tmpl.Execute(file, data); err != nil {
		log.Fatalf("Failed to execute template: %v", err)
	}
	fmt.Printf("Generated %s\n", outputPath)
}
