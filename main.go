package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	cfg := LoadConfig()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dedup, err := NewDeduplicator(cfg.RedisURL, cfg.DedupTTL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	queue := NewKafkaQueue(cfg.KafkaBroker, cfg.KafkaTopic, cfg.KafkaGroupID)

	db, err := NewDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to Postgres: %v", err)
	}
	defer db.Close()

	analyzer := NewAnalyzer(cfg)
	embedder := NewEmbedder(cfg)
	broadcaster := NewBroadcaster()
	engine := NewEngine(
		cfg,
		dedup,
		queue,
		db,
		analyzer,
		embedder,
		buildSources(cfg),
		broadcaster,
	)

	router := NewRouter(db, broadcaster, cfg.InternalAPIToken, engine.RunIngestionOnce, embedder)
	server := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: router,
	}

	var wg sync.WaitGroup
	engine.StartIngestionLoop(ctx, &wg)
	engine.StartAnalysisWorkers(ctx, &wg)

	go func() {
		log.Printf("Idea Engine API listening on :%s", cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	log.Printf("Idea Engine started with provider=%s embedder=%s and sources=%d", analyzer.Name(), embedder.Name(), len(engine.sources))

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down gracefully...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP shutdown error: %v", err)
	}
	if err := queue.Close(); err != nil {
		log.Printf("Kafka close error: %v", err)
	}

	wg.Wait()
}

func buildSources(cfg Config) []Scraper {
	sources := []Scraper{
		NewDcardScraper(cfg.DcardForums, cfg.DcardKeywords),
	}

	reddit := NewRedditScraper(
		cfg.RedditClientID,
		cfg.RedditClientSecret,
		cfg.RedditUserAgent,
		cfg.RedditSubreddits,
		cfg.RedditKeywords,
	)
	if reddit.Enabled() {
		sources = append(sources, reddit)
	}

	if len(cfg.AppStoreFeeds) > 0 {
		sources = append(sources, NewAppStoreScraper(cfg.AppStoreFeeds, cfg.AppStoreKeywords))
	}

	if len(cfg.TranscriptFeedURLs) > 0 {
		sources = append(sources, NewTranscriptFeedScraper(cfg.TranscriptFeedURLs, cfg.TranscriptKeywords))
	}

	return sources
}
