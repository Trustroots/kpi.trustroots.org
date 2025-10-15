package models

import (
	"encoding/json"
	"time"
)

// KPIData represents the complete KPI data structure
type KPIData struct {
	Generated  time.Time      `json:"generated"`
	Trustroots TrustrootsData `json:"trustroots"`
	Nostroots  NostrootsData  `json:"nostroots"`
}

// TrustrootsData contains all Trustroots-specific metrics
type TrustrootsData struct {
	MessagesPerDay         []DailyCount  `json:"messagesPerDay"`
	ReviewsPerDay          []DailyReview `json:"reviewsPerDay"`
	ThreadVotesPerDay      []DailyVote   `json:"threadVotesPerDay"`
	TimeToFirstReplyPerDay []DailyTime   `json:"timeToFirstReplyPerDay"`
}

// NostrootsData contains all Nostr-specific metrics
type NostrootsData struct {
	UsersWithNpubs    int          `json:"usersWithNpubs"`
	ActivePosters     int          `json:"activePosters"`
	NotesByKindPerDay []DailyNotes `json:"notesByKindPerDay"`
}

// DailyCount represents a count for a specific day
type DailyCount struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// DailyReview represents review counts for a specific day
type DailyReview struct {
	Date     string `json:"date"`
	Positive int    `json:"positive"`
	Negative int    `json:"negative"`
}

// DailyVote represents thread vote counts for a specific day
type DailyVote struct {
	Date      string `json:"date"`
	Upvotes   int    `json:"upvotes"`
	Downvotes int    `json:"downvotes"`
}

// DailyTime represents average time for a specific day
type DailyTime struct {
	Date  string `json:"date"`
	AvgMs int64  `json:"avgMs"`
}

// DailyNotes represents note counts by kind for a specific day
type DailyNotes struct {
	Date  string         `json:"date"`
	Kinds map[string]int `json:"-"`
}

// MarshalJSON custom marshaling for DailyNotes to flatten kinds
func (dn DailyNotes) MarshalJSON() ([]byte, error) {
	// Create a map with date and all kinds
	result := make(map[string]interface{})
	result["date"] = dn.Date
	for kind, count := range dn.Kinds {
		result["kind"+kind] = count
	}
	return json.Marshal(result)
}
