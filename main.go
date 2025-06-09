package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"kcphysics/aiLCBWebsite/internal/models"
	"kcphysics/aiLCBWebsite/internal/pagegen"
	"kcphysics/aiLCBWebsite/internal/s3deploy"
	"kcphysics/aiLCBWebsite/internal/utils"
)

const (
	imagePath       = "assets/LCB-Saxes-Logos-v1-Primary.png"
	cssVarsPath     = "static/css/style.css"
	carouselDir     = "assets/carousel_photos" // This constant is not used in main.go anymore, but keeping it for now.
	outputDir       = "public"
	staticAssetsDir = "static"
)

type RehearsalsPageData struct {
	GMapsAPIKey string
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
	allMembers, err := utils.ReadMembersCSV("members.csv")
	if err != nil {
		log.Fatalf("Failed to read members CSV: %v", err)
	}

	instrumentSections, err := utils.LoadInstrumentSections("data/instruments", allMembers)
	if err != nil {
		log.Fatalf("Failed to load instrument sections: %v", err)
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Generate color scheme and update CSS
	utils.GenerateColorScheme(imagePath, cssVarsPath)

	// Copy static assets
	utils.CopyStaticAssets(staticAssetsDir, outputDir)
	utils.CopyAssets("assets", filepath.Join(outputDir, "assets"))

	// Generate pages
	pagegen.GeneratePage(outputDir, "index.html", nil, instrumentSections)
	pagegen.GeneratePage(outputDir, "about.html", nil, instrumentSections)
	pagegen.GeneratePage(outputDir, "membership.html", nil, instrumentSections)
	gMapsAPIKey := os.Getenv("LCB_GMAP_KEY")
	if gMapsAPIKey == "" {
		log.Println("Warning: LCB_GMAP_KEY environment variable not set. Google Maps embed may not work.")
	}

	rehearsalsData := RehearsalsPageData{
		GMapsAPIKey: gMapsAPIKey,
	}
	pagegen.GeneratePage(outputDir, "rehearsals.html", rehearsalsData, instrumentSections)
	pagegen.GeneratePage(outputDir, "calendar.html", nil, instrumentSections)
	pagegen.GeneratePage(outputDir, "error.html", nil, instrumentSections)

	// Generate instrument pages
	for _, section := range instrumentSections {
		data := struct {
			Title              string
			Instrument         models.Instrument
			InstrumentSections map[string]models.Instrument
		}{
			Title:              "Lexington Community Band",
			Instrument:         section,
			InstrumentSections: instrumentSections,
		}
		pagegen.GeneratePage(outputDir, "instrument.html", data, instrumentSections)
	}

	fmt.Println("Static site generation complete!")
}

func runDeploy(bucketName string) {
	// First, generate the site
	runBuild()
	s3deploy.DeploySite(bucketName, outputDir)

	// Create CloudFront distribution
	fmt.Println("Creating CloudFront distribution...")
	distID, err := s3deploy.CreateCloudFrontDistribution(bucketName)
	if err != nil {
		log.Fatalf("Failed to create CloudFront distribution: %v", err)
	}
	fmt.Printf("CloudFront distribution created/found with ID: %s\n", distID)
}
