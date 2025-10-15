package collectors

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"kpi.trustroots.org/models"
)

// Aggregator combines data from all collectors
type Aggregator struct {
	mongoCollector *MongoCollector
	nostrCollector *NostrCollector
}

// NewAggregator creates a new aggregator
func NewAggregator(mongoCollector *MongoCollector, nostrCollector *NostrCollector) *Aggregator {
	return &Aggregator{
		mongoCollector: mongoCollector,
		nostrCollector: nostrCollector,
	}
}

// CollectAllData collects all KPI data
func (a *Aggregator) CollectAllData(targetDate *time.Time) (*models.KPIData, error) {
	// Collect Trustroots data
	trustrootsData, err := a.mongoCollector.CollectTrustrootsData(targetDate)
	if err != nil {
		return nil, fmt.Errorf("failed to collect Trustroots data: %w", err)
	}

	// Collect Nostroots data
	nostrootsData, err := a.nostrCollector.CollectNostrootsData(targetDate)
	if err != nil {
		return nil, fmt.Errorf("failed to collect Nostroots data: %w", err)
	}

	// Use target date or current time
	var generatedTime time.Time
	if targetDate != nil {
		generatedTime = targetDate.UTC()
	} else {
		generatedTime = time.Now().UTC()
	}

	// Combine all data
	kpiData := &models.KPIData{
		Generated:  generatedTime,
		Trustroots: *trustrootsData,
		Nostroots:  *nostrootsData,
	}

	return kpiData, nil
}

// SaveToFile saves KPI data to JSON file
func (a *Aggregator) SaveToFile(data *models.KPIData, outputPath string) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
