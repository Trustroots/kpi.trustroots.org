package main

import (
	"bufio"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"kpi.trustroots.org/collectors"
)

func main() {
	// Parse command line flags
	var once = flag.Bool("once", false, "Run once and exit (don't start the hourly scheduler)")
	var dateStr = flag.String("date", "", "Run for a specific date (YYYY-MM-DD format)")
	flag.Parse()

	// Load configuration from environment variables
	cfg := loadConfig()

	// Parse date if provided
	var targetDate *time.Time
	if *dateStr != "" {
		parsedDate, err := time.Parse("2006-01-02", *dateStr)
		if err != nil {
			log.Fatalf("Invalid date format '%s'. Use YYYY-MM-DD format: %v", *dateStr, err)
		}
		targetDate = &parsedDate
		log.Printf("Running for specific date: %s", targetDate.Format("2006-01-02"))
	}

	// Initialize MongoDB collector
	mongoCollector, err := collectors.NewMongoCollector(cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		log.Fatalf("Failed to initialize MongoDB collector: %v", err)
	}
	defer mongoCollector.Close()

	// Initialize Nostr collector
	nostrCollector := collectors.NewNostrCollector(cfg.NostrRelays, mongoCollector.GetDatabase())

	// Initialize aggregator
	aggregator := collectors.NewAggregator(mongoCollector, nostrCollector)

	// Run collection
	log.Println("Running KPI collection...")
	if err := runCollection(aggregator, cfg.OutputPath, targetDate); err != nil {
		log.Fatalf("Collection failed: %v", err)
	}
	log.Println("Collection completed successfully")

	// If --once flag is set, exit after one run
	if *once {
		log.Println("Exiting after single run (--once flag)")
		return
	}

	// Set up ticker for periodic updates
	ticker := time.NewTicker(cfg.UpdateInterval)
	defer ticker.Stop()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("KPI service started. Updating every %v. Output: %s", cfg.UpdateInterval, cfg.OutputPath)

	// Main loop
	for {
		select {
		case <-ticker.C:
			log.Println("Running scheduled KPI collection...")
			if err := runCollection(aggregator, cfg.OutputPath, nil); err != nil {
				log.Printf("Scheduled collection failed: %v", err)
			} else {
				log.Println("Scheduled collection completed successfully")
			}

		case sig := <-sigChan:
			log.Printf("Received signal %v, shutting down gracefully...", sig)
			return
		}
	}
}

// runCollection performs a single KPI data collection cycle
func runCollection(aggregator *collectors.Aggregator, outputPath string, targetDate *time.Time) error {
	start := time.Now()

	// Collect all data
	data, err := aggregator.CollectAllData(targetDate)
	if err != nil {
		return err
	}

	// Save to file
	if err := aggregator.SaveToFile(data, outputPath); err != nil {
		return err
	}

	duration := time.Since(start)
	log.Printf("Collection completed in %v", duration)

	return nil
}

// Config holds all configuration for the KPI service
type Config struct {
	MongoURI       string
	MongoDB        string
	NostrRelays    []string
	OutputPath     string
	UpdateInterval time.Duration
}

// loadConfig loads configuration from .env file or environment variables
func loadConfig() *Config {
	// Try to load from .env file first
	config := loadConfigFromFile(".env")

	// If .env file doesn't exist or is empty, fall back to environment variables
	if config == nil {
		config = &Config{
			MongoURI:       getEnv("MONGO_URI", "mongodb://localhost:27017"),
			MongoDB:        getEnv("MONGO_DB", "trustroots"),
			NostrRelays:    strings.Split(getEnv("NOSTR_RELAYS", "wss://relay.trustroots.org,wss://relay.nomadwiki.org"), ","),
			OutputPath:     getEnv("OUTPUT_PATH", "public/kpi.json"),
			UpdateInterval: time.Duration(getEnvInt("UPDATE_INTERVAL_MINUTES", 60)) * time.Minute,
		}
	}

	// Clean up relay URLs
	for i, relay := range config.NostrRelays {
		config.NostrRelays[i] = strings.TrimSpace(relay)
	}

	// Ensure output path is absolute
	config.OutputPath = resolveOutputPath(config.OutputPath)

	return config
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets an environment variable as integer with a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// loadConfigFromFile loads configuration from a .env file
func loadConfigFromFile(filename string) *Config {
	file, err := os.Open(filename)
	if err != nil {
		return nil
	}
	defer file.Close()

	config := &Config{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key=value pairs
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "MONGO_URI":
			config.MongoURI = value
		case "MONGO_DB":
			config.MongoDB = value
		case "NOSTR_RELAYS":
			config.NostrRelays = strings.Split(value, ",")
		case "OUTPUT_PATH":
			config.OutputPath = value
		case "UPDATE_INTERVAL_MINUTES":
			if intValue, err := strconv.Atoi(value); err == nil {
				config.UpdateInterval = time.Duration(intValue) * time.Minute
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading config file: %v", err)
		return nil
	}

	return config
}

// resolveOutputPath converts relative paths to absolute paths
// If the path is already absolute, it returns it as-is
// If the path is relative, it resolves it relative to the current working directory
func resolveOutputPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		// If we can't get the working directory, return the path as-is
		return path
	}

	// Resolve the path relative to the current working directory
	return filepath.Join(cwd, path)
}
