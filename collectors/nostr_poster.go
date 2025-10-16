package collectors

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"kpi.trustroots.org/models"
)

// NostrPoster handles posting stats to Nostr
type NostrPoster struct {
	relays []string
	nsec   string
}

// NewNostrPoster creates a new Nostr poster
func NewNostrPoster(relays []string, nsec string) *NostrPoster {
	return &NostrPoster{
		relays: relays,
		nsec:   nsec,
	}
}

// PostStats posts daily stats to Nostr
func (np *NostrPoster) PostStats(data *models.KPIData) error {
	if np.nsec == "" {
		log.Println("NSEC_STATS not configured, skipping Nostr post")
		return nil
	}

	// Decode the nsec to get the private key
	_, privateKey, err := nip19.Decode(np.nsec)
	if err != nil {
		return fmt.Errorf("failed to decode nsec: %w", err)
	}

	// Convert private key to string
	privateKeyStr := privateKey.(string)

	// Get the public key from the private key
	pubKey, err := nostr.GetPublicKey(privateKeyStr)
	if err != nil {
		return fmt.Errorf("failed to get public key: %w", err)
	}

	// Format the stats message
	message := np.formatStatsMessage(data)

	// Create the event
	event := &nostr.Event{
		Kind:      1, // Text note
		Content:   message,
		CreatedAt: nostr.Timestamp(data.Generated.Unix()),
		Tags: nostr.Tags{
			{"t", "stats"},
		},
	}

	// Set the pubkey
	event.PubKey = pubKey

	// Sign the event
	if err := event.Sign(privateKeyStr); err != nil {
		return fmt.Errorf("failed to sign event: %w", err)
	}

	// Post to all relays
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	successCount := 0
	for _, relayURL := range np.relays {
		relay, err := nostr.RelayConnect(ctx, relayURL)
		if err != nil {
			log.Printf("Failed to connect to relay %s: %v", relayURL, err)
			continue
		}

		// Publish the event
		_, err = relay.Publish(ctx, *event)
		relay.Close()

		if err != nil {
			log.Printf("Failed to publish to relay %s: %v", relayURL, err)
		} else {
			successCount++
			log.Printf("Successfully posted stats to relay %s", relayURL)
		}
	}

	if successCount == 0 {
		return fmt.Errorf("failed to post to any relay")
	}

	log.Printf("Successfully posted stats to %d/%d relays", successCount, len(np.relays))
	return nil
}

// formatStatsMessage formats the stats data into a readable message
func (np *NostrPoster) formatStatsMessage(data *models.KPIData) string {
	// Get yesterday's date
	yesterday := data.Generated.AddDate(0, 0, -1).Format("2006-01-02")

	// Calculate yesterday's message count
	var yesterdayMessages int
	for _, msg := range data.Trustroots.MessagesPerDay {
		if msg.Date == yesterday {
			yesterdayMessages = msg.Count
			break
		}
	}

	// Calculate yesterday's review counts
	var yesterdayPositiveReviews, yesterdayNegativeReviews int
	for _, review := range data.Trustroots.ReviewsPerDay {
		if review.Date == yesterday {
			yesterdayPositiveReviews = review.Positive
			yesterdayNegativeReviews = review.Negative
			break
		}
	}

	// Calculate yesterday's thread vote counts
	var yesterdayUpvotes, yesterdayDownvotes int
	for _, vote := range data.Trustroots.ThreadVotesPerDay {
		if vote.Date == yesterday {
			yesterdayUpvotes = vote.Upvotes
			yesterdayDownvotes = vote.Downvotes
			break
		}
	}

	// Calculate yesterday's notes count
	var yesterdayNotes int
	for _, notes := range data.Nostroots.NotesByKindPerDay {
		if notes.Date == yesterday {
			// Sum all kinds of notes
			for _, count := range notes.Kinds {
				yesterdayNotes += count
			}
			break
		}
	}

	// Format the message
	message := fmt.Sprintf(`Yesterday on Trustroots: %d messages, %d positive reviews, %d negative reviews, %d upvotes, %d downvotes

Nostroots: %d npub users, %d active posters, %d notes

More #stats at https://kpi.trustroots.org/`,
		yesterdayMessages,
		yesterdayPositiveReviews,
		yesterdayNegativeReviews,
		yesterdayUpvotes,
		yesterdayDownvotes,
		data.Nostroots.UsersWithNpubs,
		data.Nostroots.ActivePosters,
		yesterdayNotes)

	return message
}
