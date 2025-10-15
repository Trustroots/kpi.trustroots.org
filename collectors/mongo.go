package collectors

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"kpi.trustroots.org/models"
)

// MongoCollector handles MongoDB data collection
type MongoCollector struct {
	client   *mongo.Client
	database *mongo.Database
}

// NewMongoCollector creates a new MongoDB collector
func NewMongoCollector(uri, dbName string) (*MongoCollector, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Configure read-only connection with read preference and read concern
	clientOptions := options.Client().ApplyURI(uri).
		SetReadPreference(readpref.SecondaryPreferred()). // Prefer secondary for read-only operations
		SetReadConcern(readconcern.Local())               // Use local read concern for better performance

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Test the connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	return &MongoCollector{
		client:   client,
		database: client.Database(dbName),
	}, nil
}

// Close closes the MongoDB connection
func (mc *MongoCollector) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return mc.client.Disconnect(ctx)
}

// GetDatabase returns the database instance
func (mc *MongoCollector) GetDatabase() *mongo.Database {
	return mc.database
}

// CollectTrustrootsData collects all Trustroots metrics
func (mc *MongoCollector) CollectTrustrootsData(targetDate *time.Time) (*models.TrustrootsData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	data := &models.TrustrootsData{}

	// Collect messages per day
	messages, err := mc.collectMessagesPerDay(ctx, targetDate)
	if err != nil {
		return nil, fmt.Errorf("failed to collect messages: %w", err)
	}
	data.MessagesPerDay = messages

	// Collect reviews per day
	reviews, err := mc.collectReviewsPerDay(ctx, targetDate)
	if err != nil {
		return nil, fmt.Errorf("failed to collect reviews: %w", err)
	}
	data.ReviewsPerDay = reviews

	// Collect thread votes per day
	votes, err := mc.collectThreadVotesPerDay(ctx, targetDate)
	if err != nil {
		return nil, fmt.Errorf("failed to collect thread votes: %w", err)
	}
	data.ThreadVotesPerDay = votes

	// Collect time to first reply per day
	replyTimes, err := mc.collectTimeToFirstReplyPerDay(ctx, targetDate)
	if err != nil {
		return nil, fmt.Errorf("failed to collect reply times: %w", err)
	}
	data.TimeToFirstReplyPerDay = replyTimes

	return data, nil
}

// CollectUsersWithNpubs counts users with valid npubs
func (mc *MongoCollector) CollectUsersWithNpubs() (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{
		"nostrNpub": bson.M{
			"$exists": true,
			"$ne":     "",
		},
	}

	count, err := mc.database.Collection("users").CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count users with npubs: %w", err)
	}

	return int(count), nil
}

// collectMessagesPerDay aggregates messages by day for the last 7 days
func (mc *MongoCollector) collectMessagesPerDay(ctx context.Context, targetDate *time.Time) ([]models.DailyCount, error) {
	// Use target date or current date
	var baseDate time.Time
	if targetDate != nil {
		baseDate = *targetDate
	} else {
		baseDate = time.Now()
	}
	sevenDaysAgo := baseDate.AddDate(0, 0, -7).Truncate(24 * time.Hour)

	pipeline := []bson.M{
		{
			"$match": bson.M{
				"created": bson.M{
					"$gte": sevenDaysAgo,
				},
			},
		},
		{
			"$group": bson.M{
				"_id": bson.M{
					"$dateToString": bson.M{
						"format": "%Y-%m-%d",
						"date":   "$created",
					},
				},
				"count": bson.M{"$sum": 1},
			},
		},
		{
			"$sort": bson.M{"_id": 1},
		},
	}

	cursor, err := mc.database.Collection("messages").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []models.DailyCount
	for cursor.Next(ctx) {
		var result struct {
			ID    string `bson:"_id"`
			Count int    `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			log.Printf("Error decoding message result: %v", err)
			continue
		}
		results = append(results, models.DailyCount{
			Date:  result.ID,
			Count: result.Count,
		})
	}

	return results, cursor.Err()
}

// collectReviewsPerDay aggregates experiences by recommendation and day
func (mc *MongoCollector) collectReviewsPerDay(ctx context.Context, targetDate *time.Time) ([]models.DailyReview, error) {
	// Use target date or current date
	var baseDate time.Time
	if targetDate != nil {
		baseDate = *targetDate
	} else {
		baseDate = time.Now()
	}
	sevenDaysAgo := baseDate.AddDate(0, 0, -7).Truncate(24 * time.Hour)

	pipeline := []bson.M{
		{
			"$match": bson.M{
				"created": bson.M{
					"$gte": sevenDaysAgo,
				},
				"recommend": bson.M{
					"$in": []string{"yes", "no"},
				},
			},
		},
		{
			"$group": bson.M{
				"_id": bson.M{
					"date": bson.M{
						"$dateToString": bson.M{
							"format": "%Y-%m-%d",
							"date":   "$created",
						},
					},
					"recommend": "$recommend",
				},
				"count": bson.M{"$sum": 1},
			},
		},
		{
			"$group": bson.M{
				"_id": "$_id.date",
				"reviews": bson.M{
					"$push": bson.M{
						"recommend": "$_id.recommend",
						"count":     "$count",
					},
				},
			},
		},
		{
			"$sort": bson.M{"_id": 1},
		},
	}

	cursor, err := mc.database.Collection("experiences").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []models.DailyReview
	for cursor.Next(ctx) {
		var result struct {
			ID      string `bson:"_id"`
			Reviews []struct {
				Recommend string `bson:"recommend"`
				Count     int    `bson:"count"`
			} `bson:"reviews"`
		}
		if err := cursor.Decode(&result); err != nil {
			log.Printf("Error decoding review result: %v", err)
			continue
		}

		review := models.DailyReview{Date: result.ID}
		for _, r := range result.Reviews {
			if r.Recommend == "yes" {
				review.Positive = r.Count
			} else if r.Recommend == "no" {
				review.Negative = r.Count
			}
		}
		results = append(results, review)
	}

	return results, cursor.Err()
}

// collectThreadVotesPerDay aggregates reference thread votes by day
func (mc *MongoCollector) collectThreadVotesPerDay(ctx context.Context, targetDate *time.Time) ([]models.DailyVote, error) {
	// Use target date or current date
	var baseDate time.Time
	if targetDate != nil {
		baseDate = *targetDate
	} else {
		baseDate = time.Now()
	}
	sevenDaysAgo := baseDate.AddDate(0, 0, -7).Truncate(24 * time.Hour)

	pipeline := []bson.M{
		{
			"$match": bson.M{
				"created": bson.M{
					"$gte": sevenDaysAgo,
				},
			},
		},
		{
			"$group": bson.M{
				"_id": bson.M{
					"date": bson.M{
						"$dateToString": bson.M{
							"format": "%Y-%m-%d",
							"date":   "$created",
						},
					},
					"reference": "$reference",
				},
				"count": bson.M{"$sum": 1},
			},
		},
		{
			"$group": bson.M{
				"_id": "$_id.date",
				"votes": bson.M{
					"$push": bson.M{
						"reference": "$_id.reference",
						"count":     "$count",
					},
				},
			},
		},
		{
			"$sort": bson.M{"_id": 1},
		},
	}

	cursor, err := mc.database.Collection("referencethreads").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []models.DailyVote
	for cursor.Next(ctx) {
		var result struct {
			ID    string `bson:"_id"`
			Votes []struct {
				Reference string `bson:"reference"`
				Count     int    `bson:"count"`
			} `bson:"votes"`
		}
		if err := cursor.Decode(&result); err != nil {
			log.Printf("Error decoding vote result: %v", err)
			continue
		}

		vote := models.DailyVote{Date: result.ID}
		for _, v := range result.Votes {
			if v.Reference == "yes" {
				vote.Upvotes = v.Count
			} else if v.Reference == "no" {
				vote.Downvotes = v.Count
			}
		}
		results = append(results, vote)
	}

	return results, cursor.Err()
}

// collectTimeToFirstReplyPerDay calculates average time to first reply
func (mc *MongoCollector) collectTimeToFirstReplyPerDay(ctx context.Context, targetDate *time.Time) ([]models.DailyTime, error) {
	// Use target date or current date
	var baseDate time.Time
	if targetDate != nil {
		baseDate = *targetDate
	} else {
		baseDate = time.Now()
	}
	sevenDaysAgo := baseDate.AddDate(0, 0, -7).Truncate(24 * time.Hour)

	pipeline := []bson.M{
		{
			"$match": bson.M{
				"firstMessageCreated": bson.M{
					"$gte": sevenDaysAgo,
				},
				"timeToFirstReply": bson.M{
					"$exists": true,
					"$ne":     nil,
				},
			},
		},
		{
			"$group": bson.M{
				"_id": bson.M{
					"$dateToString": bson.M{
						"format": "%Y-%m-%d",
						"date":   "$firstMessageCreated",
					},
				},
				"avgMs": bson.M{"$avg": "$timeToFirstReply"},
			},
		},
		{
			"$sort": bson.M{"_id": 1},
		},
	}

	cursor, err := mc.database.Collection("messagestats").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []models.DailyTime
	for cursor.Next(ctx) {
		var result struct {
			ID    string  `bson:"_id"`
			AvgMs float64 `bson:"avgMs"`
		}
		if err := cursor.Decode(&result); err != nil {
			log.Printf("Error decoding reply time result: %v", err)
			continue
		}
		results = append(results, models.DailyTime{
			Date:  result.ID,
			AvgMs: int64(result.AvgMs),
		})
	}

	return results, cursor.Err()
}
