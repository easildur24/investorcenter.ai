package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// SIC code to sector mapping
// Based on Standard Industrial Classification codes
var sicToSector = map[string]string{
	// Agriculture, Forestry, Fishing (01-09)
	"01": "Consumer Defensive",
	"02": "Consumer Defensive",
	"07": "Consumer Defensive",
	"08": "Consumer Defensive",
	"09": "Consumer Defensive",

	// Mining (10-14)
	"10": "Energy",
	"12": "Energy",
	"13": "Energy",
	"14": "Basic Materials",

	// Construction (15-17)
	"15": "Industrials",
	"16": "Industrials",
	"17": "Industrials",

	// Manufacturing - Food, Tobacco, Textiles (20-23)
	"20": "Consumer Defensive",
	"21": "Consumer Defensive",
	"22": "Consumer Cyclical",
	"23": "Consumer Cyclical",

	// Manufacturing - Wood, Paper, Printing (24-27)
	"24": "Basic Materials",
	"25": "Consumer Cyclical",
	"26": "Basic Materials",
	"27": "Communication Services",

	// Manufacturing - Chemicals, Petroleum (28-29)
	"28": "Healthcare",      // Pharmaceuticals
	"29": "Energy",

	// Manufacturing - Rubber, Plastics, Leather (30-31)
	"30": "Basic Materials",
	"31": "Consumer Cyclical",

	// Manufacturing - Stone, Clay, Glass, Metals (32-34)
	"32": "Basic Materials",
	"33": "Basic Materials",
	"34": "Industrials",

	// Manufacturing - Machinery, Electronics (35-36)
	"35": "Technology",
	"36": "Technology",

	// Manufacturing - Transportation Equipment (37)
	"37": "Consumer Cyclical",

	// Manufacturing - Instruments, Misc (38-39)
	"38": "Healthcare",      // Medical instruments
	"39": "Consumer Cyclical",

	// Transportation (40-47)
	"40": "Industrials",
	"41": "Industrials",
	"42": "Industrials",
	"43": "Industrials",
	"44": "Industrials",
	"45": "Industrials",
	"46": "Energy",
	"47": "Industrials",

	// Communications (48)
	"48": "Communication Services",

	// Utilities (49)
	"49": "Utilities",

	// Wholesale Trade (50-51)
	"50": "Consumer Cyclical",
	"51": "Consumer Cyclical",

	// Retail Trade (52-59)
	"52": "Consumer Cyclical",
	"53": "Consumer Cyclical",
	"54": "Consumer Defensive",
	"55": "Consumer Cyclical",
	"56": "Consumer Cyclical",
	"57": "Consumer Cyclical",
	"58": "Consumer Cyclical",
	"59": "Consumer Cyclical",

	// Finance, Insurance, Real Estate (60-67)
	"60": "Financial Services",
	"61": "Financial Services",
	"62": "Financial Services",
	"63": "Financial Services",
	"64": "Financial Services",
	"65": "Real Estate",
	"67": "Financial Services",

	// Services (70-89)
	"70": "Consumer Cyclical",
	"72": "Consumer Cyclical",
	"73": "Technology",       // Business services, software
	"75": "Consumer Cyclical",
	"76": "Consumer Cyclical",
	"78": "Communication Services",
	"79": "Communication Services",
	"80": "Healthcare",
	"81": "Industrials",
	"82": "Consumer Cyclical",
	"83": "Consumer Defensive",
	"84": "Consumer Cyclical",
	"86": "Consumer Cyclical",
	"87": "Technology",       // Engineering, research services
	"89": "Industrials",

	// Public Administration (91-99)
	"91": "Industrials",
	"92": "Industrials",
	"93": "Industrials",
	"94": "Industrials",
	"95": "Industrials",
	"96": "Industrials",
	"97": "Industrials",
	"99": "Industrials",
}

// More specific 4-digit SIC mappings for accuracy
var sic4ToSector = map[string]string{
	// Specific tech companies
	"3571": "Technology",  // Electronic Computers
	"3572": "Technology",  // Computer Storage Devices
	"3575": "Technology",  // Computer Terminals
	"3576": "Technology",  // Computer Communication Equipment
	"3577": "Technology",  // Computer Peripheral Equipment
	"3578": "Technology",  // Calculating and Accounting Machines
	"3579": "Technology",  // Office Machines
	"3661": "Technology",  // Telephone and Telegraph Apparatus
	"3663": "Technology",  // Radio and TV Broadcasting Equipment
	"3669": "Technology",  // Communications Equipment
	"3674": "Technology",  // Semiconductors
	"3825": "Technology",  // Instruments for Measuring
	"7370": "Technology",  // Computer Programming Services
	"7371": "Technology",  // Computer Programming Services
	"7372": "Technology",  // Prepackaged Software
	"7373": "Technology",  // Computer Integrated Systems Design
	"7374": "Technology",  // Computer Processing Services
	"7375": "Technology",  // Information Retrieval Services
	"7376": "Technology",  // Computer Facilities Management
	"7377": "Technology",  // Computer Rental and Leasing
	"7378": "Technology",  // Computer Maintenance and Repair
	"7379": "Technology",  // Computer Related Services

	// Specific healthcare
	"2834": "Healthcare",  // Pharmaceutical Preparations
	"2835": "Healthcare",  // In Vitro and In Vivo Diagnostics
	"2836": "Healthcare",  // Biological Products
	"3826": "Healthcare",  // Laboratory Analytical Instruments
	"3841": "Healthcare",  // Surgical and Medical Instruments
	"3842": "Healthcare",  // Orthopedic, Prosthetic Appliances
	"3843": "Healthcare",  // Dental Equipment and Supplies
	"3844": "Healthcare",  // X-Ray Apparatus and Tubes
	"3845": "Healthcare",  // Electromedical Equipment
	"8011": "Healthcare",  // Offices of Doctors
	"8021": "Healthcare",  // Offices of Dentists
	"8031": "Healthcare",  // Offices of Osteopathic Physicians
	"8041": "Healthcare",  // Offices of Chiropractors
	"8042": "Healthcare",  // Offices of Optometrists
	"8049": "Healthcare",  // Offices of Health Practitioners
	"8051": "Healthcare",  // Skilled Nursing Care Facilities
	"8052": "Healthcare",  // Intermediate Care Facilities
	"8059": "Healthcare",  // Nursing and Personal Care
	"8062": "Healthcare",  // General Medical and Surgical Hospitals
	"8063": "Healthcare",  // Psychiatric Hospitals
	"8069": "Healthcare",  // Specialty Hospitals
	"8071": "Healthcare",  // Medical Laboratories
	"8072": "Healthcare",  // Dental Laboratories
	"8082": "Healthcare",  // Home Health Care Services
	"8092": "Healthcare",  // Kidney Dialysis Centers
	"8093": "Healthcare",  // Specialty Outpatient Facilities
	"8099": "Healthcare",  // Health and Allied Services

	// Specific communication services
	"4812": "Communication Services",  // Radiotelephone Communications
	"4813": "Communication Services",  // Telephone Communications
	"4822": "Communication Services",  // Telegraph and Other Message
	"4832": "Communication Services",  // Radio Broadcasting Stations
	"4833": "Communication Services",  // Television Broadcasting Stations
	"4841": "Communication Services",  // Cable and Other Pay Television
	"7812": "Communication Services",  // Motion Picture Production
	"7819": "Communication Services",  // Services Allied to Motion Picture
	"7822": "Communication Services",  // Motion Picture Distribution
	"7829": "Communication Services",  // Services Allied to Motion Picture
	"7832": "Communication Services",  // Motion Picture Theaters
	"7841": "Communication Services",  // Video Tape Rental
	"7941": "Communication Services",  // Professional Sports Clubs

	// E-commerce and internet retail
	"5961": "Consumer Cyclical",  // Catalog and Mail-Order Houses (includes e-commerce)

	// Automotive
	"3711": "Consumer Cyclical",  // Motor Vehicles and Passenger Car Bodies
	"3713": "Consumer Cyclical",  // Truck and Bus Bodies
	"3714": "Consumer Cyclical",  // Motor Vehicle Parts and Accessories
	"3715": "Consumer Cyclical",  // Truck Trailers
	"3716": "Consumer Cyclical",  // Motor Homes
	"5511": "Consumer Cyclical",  // Motor Vehicle Dealers (New and Used)
	"5521": "Consumer Cyclical",  // Motor Vehicle Dealers (Used Only)
	"5531": "Consumer Cyclical",  // Auto and Home Supply Stores
}

type PolygonResponse struct {
	Results struct {
		SICCode        string `json:"sic_code"`
		SICDescription string `json:"sic_description"`
	} `json:"results"`
	Status string `json:"status"`
}

func getSectorFromSIC(sicCode string) string {
	if sicCode == "" {
		return ""
	}

	// Try 4-digit match first
	if sector, ok := sic4ToSector[sicCode]; ok {
		return sector
	}

	// Try 2-digit prefix
	if len(sicCode) >= 2 {
		prefix := sicCode[:2]
		if sector, ok := sicToSector[prefix]; ok {
			return sector
		}
	}

	return ""
}

func main() {
	// Database connection
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	polygonAPIKey := os.Getenv("POLYGON_API_KEY")

	if dbHost == "" || dbUser == "" || dbPassword == "" || dbName == "" {
		log.Fatal("Database environment variables not set")
	}
	if polygonAPIKey == "" {
		log.Fatal("POLYGON_API_KEY environment variable not set")
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Connected to database")

	// Get tickers that need sector data
	rows, err := db.Query(`
		SELECT symbol FROM tickers
		WHERE (sector IS NULL OR sector = '')
		AND asset_type IN ('CS', 'PFD', 'stock')
		AND symbol NOT LIKE 'I:%'
		AND symbol NOT LIKE 'X:%'
		AND symbol NOT LIKE 'C:%'
		ORDER BY symbol
	`)
	if err != nil {
		log.Fatalf("Failed to query tickers: %v", err)
	}
	defer rows.Close()

	var symbols []string
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		symbols = append(symbols, symbol)
	}

	log.Printf("Found %d tickers without sector data", len(symbols))

	client := &http.Client{Timeout: 10 * time.Second}
	updated := 0
	failed := 0
	noSector := 0

	for i, symbol := range symbols {
		// Rate limit: 5 requests per second (Polygon free tier)
		time.Sleep(200 * time.Millisecond)

		url := fmt.Sprintf("https://api.polygon.io/v3/reference/tickers/%s?apiKey=%s", symbol, polygonAPIKey)
		resp, err := client.Get(url)
		if err != nil {
			log.Printf("[%d/%d] Error fetching %s: %v", i+1, len(symbols), symbol, err)
			failed++
			continue
		}

		var data PolygonResponse
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			resp.Body.Close()
			log.Printf("[%d/%d] Error decoding response for %s: %v", i+1, len(symbols), symbol, err)
			failed++
			continue
		}
		resp.Body.Close()

		sicCode := data.Results.SICCode
		sector := getSectorFromSIC(sicCode)

		if sector == "" {
			if sicCode != "" {
				log.Printf("[%d/%d] No sector mapping for %s (SIC: %s - %s)", i+1, len(symbols), symbol, sicCode, data.Results.SICDescription)
			} else {
				log.Printf("[%d/%d] No SIC code for %s", i+1, len(symbols), symbol)
			}
			noSector++
			continue
		}

		// Update database
		_, err = db.Exec("UPDATE tickers SET sector = $1 WHERE symbol = $2", sector, symbol)
		if err != nil {
			log.Printf("[%d/%d] Error updating %s: %v", i+1, len(symbols), symbol, err)
			failed++
			continue
		}

		log.Printf("[%d/%d] Updated %s: SIC %s -> %s", i+1, len(symbols), symbol, sicCode, sector)
		updated++

		// Progress update every 100
		if (i+1)%100 == 0 {
			log.Printf("Progress: %d/%d (updated: %d, no sector: %d, failed: %d)",
				i+1, len(symbols), updated, noSector, failed)
		}
	}

	log.Printf("Completed: %d updated, %d no sector mapping, %d failed out of %d total",
		updated, noSector, failed, len(symbols))

	// Refresh the materialized view
	log.Println("Refreshing screener_data materialized view...")
	_, err = db.Exec("REFRESH MATERIALIZED VIEW screener_data")
	if err != nil {
		log.Printf("Warning: Failed to refresh materialized view: %v", err)
		log.Println("You may need to run: REFRESH MATERIALIZED VIEW screener_data;")
	} else {
		log.Println("Materialized view refreshed successfully")
	}
}
