package collectors

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
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

	// Get npubs for querying relays
	npubs, err := nc.getNpubsFromUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get npubs: %w", err)
	}

	// Query relays for events and get valid npub count
	validNpubs, activePosters, notesByKind, err := nc.queryRelaysForEvents(ctx, npubs, targetDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query relays: %w", err)
	}

	data.UsersWithNpubs = validNpubs
	data.ActivePosters = activePosters
	data.NotesByKindPerDay = notesByKind

	return data, nil
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
func (nc *NostrCollector) queryRelaysForEvents(ctx context.Context, npubs []string, targetDate *time.Time) (int, int, []models.DailyNotes, error) {
	if len(npubs) == 0 {
		return 0, 0, []models.DailyNotes{}, nil
	}

	log.Printf("Querying %d npubs from %d relays (real implementation)", len(npubs), len(nc.relays))

	// Convert npubs to pubkeys
	pubkeys := make([]string, 0, len(npubs))
	validNpubs := 0
	for _, npub := range npubs {
		// Only process strings that look like npubs (start with npub1)
		if len(npub) > 5 && npub[:5] == "npub1" {
			if prefix, value, err := nip19.Decode(npub); err == nil && prefix == "npub" {
				if pubkey, ok := value.(string); ok {
					pubkeys = append(pubkeys, pubkey)
					validNpubs++
				}
			} else {
				log.Printf("Invalid npub format: %s", npub)
			}
		}
		// Skip non-npub entries silently (they're likely URLs, usernames, etc.)
	}

	log.Printf("Found %d valid npubs out of %d total entries", validNpubs, len(npubs))

	if len(pubkeys) == 0 {
		log.Printf("No valid pubkeys found from %d npubs", len(npubs))
		return validNpubs, 0, []models.DailyNotes{}, nil
	}

	// Query relays for events
	events, err := nc.queryRelays(ctx, pubkeys, targetDate)
	if err != nil {
		log.Printf("Error querying relays: %v", err)
		// Return empty data if relay querying fails
		return validNpubs, 0, []models.DailyNotes{}, err
	}

	// Process events to get active posters and notes by kind
	activePosters, notesByKind := nc.processEvents(events, targetDate)

	return validNpubs, activePosters, notesByKind, nil
}

// queryRelays queries all configured relays for events
func (nc *NostrCollector) queryRelays(ctx context.Context, pubkeys []string, targetDate *time.Time) ([]*nostr.Event, error) {
	var allEvents []*nostr.Event

	// Calculate time range (last 7 days)
	var since time.Time
	if targetDate != nil {
		since = targetDate.AddDate(0, 0, -7)
	} else {
		since = time.Now().AddDate(0, 0, -7)
	}
	until := time.Now()

	// Convert to nostr timestamps
	sinceTimestamp := nostr.Timestamp(since.Unix())
	untilTimestamp := nostr.Timestamp(until.Unix())

	// Query each relay
	for _, relayURL := range nc.relays {
		log.Printf("Querying relay: %s", relayURL)

		relay, err := nostr.RelayConnect(ctx, relayURL)
		if err != nil {
			log.Printf("Failed to connect to relay %s: %v", relayURL, err)
			continue
		}
		defer relay.Close()

		// Create filter for the pubkeys and time range
		filter := nostr.Filter{
			Authors: pubkeys,
			Since:   &sinceTimestamp,
			Until:   &untilTimestamp,
			Kinds:   []int{0, 1, 4, 30023, 397, 30398, 30399}, // Profile metadata, notes, encrypted DMs, long-form content, app-specific data, community posts, community post replies
		}

		// Query the relay
		events, err := relay.QuerySync(ctx, filter)
		if err != nil {
			log.Printf("Failed to query relay %s: %v", relayURL, err)
			continue
		}

		log.Printf("Found %d events from relay %s", len(events), relayURL)
		allEvents = append(allEvents, events...)
	}

	log.Printf("Total events found across all relays: %d", len(allEvents))
	return allEvents, nil
}

// processEvents processes the events to extract metrics
func (nc *NostrCollector) processEvents(events []*nostr.Event, targetDate *time.Time) (int, []models.DailyNotes) {
	// Track active posters (unique authors)
	activeAuthors := make(map[string]bool)

	// Track notes by kind and day
	notesByDay := make(map[string]map[string]int)

	// Use target date or current date for base
	var baseDate time.Time
	if targetDate != nil {
		baseDate = *targetDate
	} else {
		baseDate = time.Now()
	}

	// Initialize notesByDay for the last 7 days
	for i := 6; i >= 0; i-- {
		date := baseDate.AddDate(0, 0, -i).Format("2006-01-02")
		notesByDay[date] = map[string]int{
			"0":     0, // Profile metadata
			"1":     0, // Notes
			"4":     0, // Encrypted DMs
			"30023": 0, // Long-form content
			"397":   0, // App-specific data
			"30398": 0, // Community post
			"30399": 0, // Community post reply
		}
	}

	// Process each event
	for _, event := range events {
		// Track active authors
		activeAuthors[event.PubKey] = true

		// Get event date
		eventDate := time.Unix(int64(event.CreatedAt), 0).Format("2006-01-02")

		// Check if this date is within our range
		if dayData, exists := notesByDay[eventDate]; exists {
			kindStr := fmt.Sprintf("%d", event.Kind)
			if kindStr == "0" || kindStr == "1" || kindStr == "4" || kindStr == "30023" || kindStr == "397" || kindStr == "30398" || kindStr == "30399" {
				dayData[kindStr]++
			}
		}
	}

	// Convert to DailyNotes format
	var results []models.DailyNotes
	for i := 6; i >= 0; i-- {
		date := baseDate.AddDate(0, 0, -i).Format("2006-01-02")
		if dayData, exists := notesByDay[date]; exists {
			results = append(results, models.DailyNotes{
				Date:  date,
				Kinds: dayData,
			})
		}
	}

	return len(activeAuthors), results
}


// aggregateNotesByKind aggregates events by kind and day (placeholder for future implementation)
func (nc *NostrCollector) aggregateNotesByKind(events map[string]interface{}) []models.DailyNotes {
	// This will be implemented when nostr library dependencies are resolved
	return []models.DailyNotes{}
}
