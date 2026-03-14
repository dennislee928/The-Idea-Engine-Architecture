package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dennis_lee/idea-engine/backend/internal/analyzer"
	"github.com/dennis_lee/idea-engine/backend/internal/cache"
	"github.com/dennis_lee/idea-engine/backend/internal/queue"
	"github.com/dennis_lee/idea-engine/backend/internal/scraper"
	"github.com/dennis_lee/idea-engine/backend/internal/storage"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Initialize Infrastructure Components
	redisURL := "redis://localhost:6379/0"
	dedup, err := cache.NewDeduplicator(redisURL, 7*24*time.Hour) // 7 days TTL
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	kafkaQ := queue.NewKafkaQueue("localhost:9092", "raw-posts")

	dbURL := "postgres://idea_admin:idea_password@localhost:5432/idea_engine?sslmode=disable"
	db, err := storage.NewDB(dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to Postgres: %v", err)
	}

	gemini := analyzer.NewGeminiAnalyzer()
	dcardScraper := scraper.NewDcardScraper("softwareengineer")

	log.Println("Idea Engine Backend Started successfully.")

	// 2. Start Ingestion Worker (Scraper -> Redis -> Kafka)
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			log.Println("Starting Dcard scraping cycle...")
			posts, err := dcardScraper.Scrape(ctx, 30)
			if err != nil {
				log.Printf("Scrape error: %v", err)
				continue
			}

			for _, post := range posts {
				isDup, err := dedup.IsDuplicate(ctx, post.Platform, post.ID)
				if err != nil {
					log.Printf("Redis error: %v", err)
					continue
				}
				if isDup {
					continue // Skip already processed posts
				}

				// Push new post to Kafka
				msg := queue.Message{
					PostID:   post.ID,
					Platform: post.Platform,
					Content:  post.Content,
					URL:      post.URL,
				}
				if err := kafkaQ.Push(ctx, msg); err != nil {
					log.Printf("Kafka push error: %v", err)
				} else {
					log.Printf("Pushed new post %s to Kafka", post.ID)
				}
			}

			select {
			case <-ticker.C:
			case <-ctx.Done():
				return
			}
		}
	}()

	// 3. Start Intelligence Worker (Kafka -> LLM -> Postgres)
	go func() {
		for {
			msg, err := kafkaQ.Consume(ctx)
			if err != nil {
				log.Printf("Kafka consume error: %v", err)
				time.Sleep(2 * time.Second)
				continue
			}

			log.Printf("Analyzing post %s from %s...", msg.PostID, msg.Platform)
			insight, err := gemini.AnalyzeText(ctx, msg.Content)
			if err != nil || insight == nil {
				log.Printf("LLM analysis error: %v", err)
				continue
			}

			if err := db.SaveInsight(ctx, msg.Platform, msg.URL, insight); err != nil {
				log.Printf("Postgres save error: %v", err)
			}
		}
	}()

	// 4. Graceful Shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Shutting down gracefully...")
}
