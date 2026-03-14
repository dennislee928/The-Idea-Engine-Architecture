package main

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type Deduplicator struct {
	client *redis.Client
	ttl    time.Duration
}

func NewDeduplicator(redisURL string, ttl time.Duration) (*Deduplicator, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)
	// Ping to test connection
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return &Deduplicator{
		client: client,
		ttl:    ttl,
	}, nil
}

// IsDuplicate checks if a postID has been processed. If not, it marks it as processed.
// Returns true if it's a duplicate, false if it's new.
func (d *Deduplicator) IsDuplicate(ctx context.Context, platform, postID string) (bool, error) {
	key := "ingest:" + platform + ":" + postID
	// SetNX sets the key only if it does not exist.
	// Returns true if key was set (meaning it's NEW), false if not set (meaning it's a DUPLICATE).
	isNew, err := d.client.SetNX(ctx, key, "1", d.ttl).Result()
	if err != nil {
		return false, err
	}
	return !isNew, nil
}
