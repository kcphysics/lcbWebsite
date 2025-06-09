# Lexington Community Band Static Site Generator

This project is a static site generator built in Go, designed to create a website for the Lexington Community Band. It generates a complete static website that can be easily hosted on platforms like AWS S3.

## Features

*   **Dynamic Content Generation:** Pages are generated from Go templates, allowing for dynamic content injection.
*   **Color Scheme Generation:** Automatically extracts a color scheme from `assets/LCB-Saxes-Logos-v1-Primary.png` and applies it to the site's CSS using CSS variables, ensuring consistent branding.
*   **Responsive Design with Bootstrap:** Utilizes Bootstrap 5 for a modern, responsive, and mobile-first design.
*   **Image Carousel:** Features a dynamic image carousel on the homepage using images from the `assets/carousel_photos` directory.
*   **Dedicated Pages:** Includes pre-built pages for:
    *   **About Us:** Provides information about the band, including a placeholder image.
    *   **Membership:** Offers details about joining the band.
    *   **Rehearsals:** Displays rehearsal schedule information, including a link to Google Maps for location.
    *   **Calendar:** Embeds a Google Calendar for upcoming events.
    *   **Instrument Sections:** A dropdown menu in the navbar leads to individual pages for various instrument sections (e.g., Flutes, Bass Clarinets, Tenor Saxophones, Percussion). Each page features an instrument image, description, and a table of sample members with their day jobs, driven by a CSV file.
*   **Modular Templating:** Common components like headers, footers, and navigation bars are refactored into partial templates for improved maintainability and code reusability.
*   **Command-Line Interface (CLI):** Provides simple commands to generate and deploy the site.

## Getting Started

### Prerequisites

*   Go (version 1.16 or higher) installed on your system.
*   AWS CLI configured with credentials if you plan to deploy to S3.

### Local Site Generation

To generate the static website for local viewing:

1.  **Clone the repository:**
    ```bash
    git clone [repository_url]
    cd aiLCBWebsite
    ```
2.  **Install Go dependencies:**
    ```bash
    go mod tidy
    ```
3.  **Generate the site:**
    ```bash
    go run main.go build
    ```
    This command will create a `public` directory in the root of the project, containing all the generated HTML, CSS, and image files.

### Viewing the Site Locally

After generating the site, you can open the `index.html` file in your web browser:

```bash
open public/index.html
```

**Note on Local Navigation:**
When viewing the site locally using the `file://` protocol, navigation between pages (e.g., clicking links in the navbar or dropdown) might not work as expected. This is a common security restriction in web browsers that prevents JavaScript from navigating between local files. The site is fully functional and all links will work correctly when hosted on a web server.

### Deploying to AWS S3

To deploy the generated static site to an AWS S3 bucket:

1.  **Ensure AWS CLI is configured:** Make sure your AWS credentials are set up correctly on your system.
2.  **Run the deploy command:**
    ```bash
    go run main.go deploy --bucket your-s3-bucket-name
    ```
    Replace `your-s3-bucket-name` with the actual name of your S3 bucket. This command will first generate the latest version of the site and then upload all contents of the `public` directory to the specified S3 bucket.

    **Important:** Ensure your S3 bucket is configured for static website hosting.

## Project Structure

```
.
├── assets/
│   ├── band_inspiration.png
│   ├── carousel_photos/
│   │   ├── header.jpg
│   │   ├── horns.avif
│   │   └── tubas.avif
│   ├── favicon.ico
│   ├── john_immerso.jpg
│   ├── LCB-Saxes-Logos-v1-Primary.png
│   └── instruments/
│       ├── bass_clarinet.jpg
│       ├── flute.jpg
│       ├── percussion.jpg
│       └── tenor_saxophone.jpg
├── members.csv
├── templates/
│   ├── about.html
│   ├── calendar.html
│   ├── index.html
│   ├── instrument.html
│   ├── membership.html
│   ├── rehearsals.html
│   └── partials/
│       ├── footer.html
│       ├── header.html
│       ├── instrument_dropdown.html
│       └── navbar.html
├── static/
│   ├── css/
│   │   ├── bootstrap.min.css
│   │   └── style.css
│   └── js/
│       └── bootstrap.bundle.min.js
└── main.go
