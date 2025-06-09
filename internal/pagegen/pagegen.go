package pagegen

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"

	"kcphysics/aiLCBWebsite/internal/models" // Import the models package
)

const (
	googleCalendarURL = "https://calendar.google.com/calendar/embed?src=85b15423a562f0de4c47230084c68a284b5481593ec04121be2ef24f5a2926a2%40group.calendar.google.com&ctz=America%2FNew_York"
	googleMapURL      = "https://www.google.com/maps/embed?pb=!1m18!1m12!1m3!1d3300.7997000000003!2d-81.23456789999999!3d33.99999999999999!2m3!1f0!2f0!3f0!3m2!1i1024!2i768!4f13.1!3m3!1m2!1s0x88f8d9e0b0b0b0b0%3A0x0!2s320%20Corley%20Mill%20Rd%2C%20Lexington%2C%20SC%2029072!5e0!3m2!1sen!2sus!4v1678888888888!5m2!1sen!2sus"
)

// GeneratePage is a generic function to parse templates and generate HTML pages.
func GeneratePage(outputDir, templateName string, data interface{}, instrumentSections map[string]models.Instrument) {
	tmpl, err := template.ParseFiles(
		filepath.Join("templates", templateName),
		"templates/partials/header.html",
		"templates/partials/navbar.html",
		"templates/partials/footer.html",
		"templates/partials/instrument_dropdown.html",
	)
	if err != nil {
		log.Fatalf("Failed to parse template %s: %v", templateName, err)
	}

	outputPath := filepath.Join(outputDir, templateName)
	// Special handling for instrument pages to create a clean URL
	if templateName == "instrument.html" {
		instrumentData, ok := data.(struct {
			Title              string
			Instrument         models.Instrument
			InstrumentSections map[string]models.Instrument
		})
		if ok {
			outputPath = filepath.Join(outputDir, strings.ToLower(strings.ReplaceAll(instrumentData.Instrument.Name, " ", "_"))+".html")
		}
	}

	file, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("Failed to create output file %s: %v", outputPath, err)
	}
	defer file.Close()

	// Create a generic data structure that includes InstrumentSections
	googleAnalyticsID := os.Getenv("LCB_GOOGLE_ANALYTICS_ID")

	pageData := struct {
		Title              string
		Instrument         models.Instrument // Only for instrument pages
		CalendarURL        string            // Only for calendar page
		MapURL             string            // Only for about page
		InstrumentSections map[string]models.Instrument
		GoogleAnalyticsID  string
		GMapsAPIKey        string // Added for rehearsals page
	}{
		Title:              "Lexington Community Band",
		InstrumentSections: instrumentSections,
		GoogleAnalyticsID:  googleAnalyticsID,
		GMapsAPIKey:        os.Getenv("LCB_GMAP_KEY"), // Populate from env var
	}

	// Populate specific fields based on templateName
	switch templateName {
	case "instrument.html":
		instrumentData, ok := data.(struct {
			Title              string
			Instrument         models.Instrument
			InstrumentSections map[string]models.Instrument
		})
		if ok {
			pageData.Instrument = instrumentData.Instrument
		}
	case "calendar.html":
		pageData.CalendarURL = googleCalendarURL
	case "about.html":
		pageData.MapURL = googleMapURL
	}

	if err := tmpl.Execute(file, pageData); err != nil {
		log.Fatalf("Failed to execute template %s: %v", templateName, err)
	}
	fmt.Printf("Generated %s\n", outputPath)
}
