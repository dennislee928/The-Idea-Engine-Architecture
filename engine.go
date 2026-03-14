package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

type Analyzer interface {
	AnalyzeText(ctx context.Context, post ScrapedPost) (*Insight, error)
	Name() string
}

type Engine struct {
	cfg         Config
	dedup       *Deduplicator
	queue       *KafkaQueue
	db          *DB
	analyzer    Analyzer
	sources     []Scraper
	broadcaster *Broadcaster
}

func NewEngine(cfg Config, dedup *Deduplicator, queue *KafkaQueue, db *DB, analyzer Analyzer, sources []Scraper, broadcaster *Broadcaster) *Engine {
	return &Engine{
		cfg:         cfg,
		dedup:       dedup,
		queue:       queue,
		db:          db,
		analyzer:    analyzer,
		sources:     sources,
		broadcaster: broadcaster,
	}
}

func (e *Engine) RunIngestionOnce(ctx context.Context) (IngestionResult, error) {
	result := IngestionResult{Sources: len(e.sources)}

	for _, source := range e.sources {
		posts, err := source.Scrape(ctx, e.cfg.IngestionBatchSize)
		if err != nil {
			errMsg := fmt.Sprintf("%s scrape failed: %v", source.Name(), err)
			log.Println(errMsg)
			result.Errors = append(result.Errors, errMsg)
			continue
		}

		result.Fetched += len(posts)
		for _, post := range posts {
			message := post.ToQueueMessage()
			isDuplicate, err := e.dedup.IsDuplicate(ctx, message.Platform, message.ID)
			if err != nil {
				errMsg := fmt.Sprintf("%s dedup failed for %s: %v", source.Name(), message.ID, err)
				log.Println(errMsg)
				result.Errors = append(result.Errors, errMsg)
				continue
			}
			if isDuplicate {
				result.SkippedDuplicates++
				continue
			}

			if err := e.queue.Push(ctx, message); err != nil {
				errMsg := fmt.Sprintf("%s enqueue failed for %s: %v", source.Name(), message.ID, err)
				log.Println(errMsg)
				result.Errors = append(result.Errors, errMsg)
				continue
			}
			result.Enqueued++
		}
	}

	if len(result.Errors) > 0 {
		return result, errors.New("ingestion completed with partial failures")
	}

	return result, nil
}

func (e *Engine) StartIngestionLoop(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		log.Println("Starting ingestion scheduler...")
		e.logIngestionResult(e.RunIngestionOnce(ctx))

		ticker := time.NewTicker(e.cfg.IngestionInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				e.logIngestionResult(e.RunIngestionOnce(ctx))
			}
		}
	}()
}

func (e *Engine) StartAnalysisWorkers(ctx context.Context, wg *sync.WaitGroup) {
	workerCount := e.cfg.IntelligenceWorkers
	if workerCount < 1 {
		workerCount = 1
	}

	for workerID := 1; workerID <= workerCount; workerID++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			log.Printf("Intelligence worker %d started", id)
			for {
				msg, err := e.queue.Consume(ctx)
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					log.Printf("Worker %d consume error: %v", id, err)
					time.Sleep(2 * time.Second)
					continue
				}

				post := msg.ToScrapedPost()
				insight, err := e.analyzer.AnalyzeText(ctx, post)
				if err != nil {
					log.Printf("Worker %d analysis error for %s: %v", id, msg.ID, err)
					continue
				}

				record, err := e.db.SaveInsight(ctx, post, insight, e.analyzer.Name())
				if err != nil {
					log.Printf("Worker %d save error for %s: %v", id, msg.ID, err)
					continue
				}

				if !record.IsExplicitContent {
					e.broadcaster.Broadcast(record)
				}
			}
		}(workerID)
	}
}

func (e *Engine) logIngestionResult(result IngestionResult, err error) {
	log.Printf(
		"Ingestion cycle complete: sources=%d fetched=%d enqueued=%d duplicates=%d errors=%d",
		result.Sources,
		result.Fetched,
		result.Enqueued,
		result.SkippedDuplicates,
		len(result.Errors),
	)
	if err != nil {
		log.Printf("Ingestion finished with warnings: %v", err)
	}
}
