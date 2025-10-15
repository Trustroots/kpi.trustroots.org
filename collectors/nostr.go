package collectors

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"kpi.trustroots.org/models"
)

// NostrCollector handles Nostr relay data collection
type NostrCollector struct {
	relays []string
	mongo  *mongo.Database
}

// NewNostrCollector creates a new Nostr collector
func NewNostrCollector(relays []string, mongoDB *mongo.Database) *NostrCollector {
	return &NostrCollector{
		relays: relays,
		mongo:  mongoDB,
	}
}

// CollectNostrootsData collects all Nostr-related metrics
func (nc *NostrCollector) CollectNostrootsData(targetDate *time.Time) (*models.NostrootsData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	data := &models.NostrootsData{}

	// Get users with npubs from MongoDB
	usersWithNpubs, err := nc.getUsersWithNpubs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get users with npubs: %w", err)
	}
	data.UsersWithNpubs = usersWithNpubs

	// Get npubs for querying relays
	npubs, err := nc.getNpubsFromUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get npubs: %w", err)
	}

	// Query relays for events
	activePosters, notesByKind, err := nc.queryRelaysForEvents(ctx, npubs, targetDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query relays: %w", err)
	}

	data.ActivePosters = activePosters
	data.NotesByKindPerDay = notesByKind

	return data, nil
}

// getUsersWithNpubs counts users with valid npubs
func (nc *NostrCollector) getUsersWithNpubs(ctx context.Context) (int, error) {
	filter := bson.M{
		"nostrNpub": bson.M{
			"$exists": true,
			"$ne":     "",
		},
	}

	count, err := nc.mongo.Collection("users").CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}

	return int(count), nil
}

// getNpubsFromUsers retrieves all npubs from users
func (nc *NostrCollector) getNpubsFromUsers(ctx context.Context) ([]string, error) {
	filter := bson.M{
		"nostrNpub": bson.M{
			"$exists": true,
			"$ne":     "",
		},
	}

	projection := bson.M{"nostrNpub": 1}

	cursor, err := nc.mongo.Collection("users").Find(ctx, filter, options.Find().SetProjection(projection))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var npubs []string
	for cursor.Next(ctx) {
		var user struct {
			NostrNpub string `bson:"nostrNpub"`
		}
		if err := cursor.Decode(&user); err != nil {
			log.Printf("Error decoding user npub: %v", err)
			continue
		}
		if user.NostrNpub != "" {
			npubs = append(npubs, user.NostrNpub)
		}
	}

	return npubs, cursor.Err()
}

// queryRelaysForEvents queries all relays for events by the given npubs
func (nc *NostrCollector) queryRelaysForEvents(ctx context.Context, npubs []string, targetDate *time.Time) (int, []models.DailyNotes, error) {
	if len(npubs) == 0 {
		return 0, []models.DailyNotes{}, nil
	}

	// For now, return mock data since nostr library has dependency issues
	// TODO: Implement actual nostr relay queries when dependencies are resolved
	// Querying for kinds: 0 (profile metadata), 1 (notes), 30023 (long-form content)
	log.Printf("Querying %d npubs from %d relays (mock implementation)", len(npubs), len(nc.relays))

	// Mock data for testing
	activePosters := len(npubs) / 5 // Assume 20% of users with npubs are active
	if activePosters == 0 && len(npubs) > 0 {
		activePosters = 1
	}

	// Generate mock notes data for the last 7 days
	notesByKind := nc.generateMockNotesData(targetDate)

	return activePosters, notesByKind, nil
}

// generateMockNotesData generates mock notes data for testing
func (nc *NostrCollector) generateMockNotesData(targetDate *time.Time) []models.DailyNotes {
	var results []models.DailyNotes

	// Use target date or current date
	var baseDate time.Time
	if targetDate != nil {
		baseDate = *targetDate
	} else {
		baseDate = time.Now()
	}

	// Generate data for the last 7 days
	for i := 6; i >= 0; i-- {
		date := baseDate.AddDate(0, 0, -i).Format("2006-01-02")

		// Mock some activity
		kinds := make(map[string]int)
		if i%2 == 0 { // Some days have more activity
			kinds["0"] = 2 + i     // Profile metadata
			kinds["1"] = 15 + i*2  // Notes
			kinds["30023"] = 3 + i // Long-form content
		} else {
			kinds["0"] = 1
			kinds["1"] = 8 + i
			kinds["30023"] = 1
		}

		results = append(results, models.DailyNotes{
			Date:  date,
			Kinds: kinds,
		})
	}

	return results
}

// aggregateNotesByKind aggregates events by kind and day (placeholder for future implementation)
func (nc *NostrCollector) aggregateNotesByKind(events map[string]interface{}) []models.DailyNotes {
	// This will be implemented when nostr library dependencies are resolved
	return []models.DailyNotes{}
}
