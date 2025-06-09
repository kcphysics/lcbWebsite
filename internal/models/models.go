package models

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
