package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type TranscriptFeedScraper struct {
	client   *http.Client
	feedURLs []string
	keywords []string
}

func NewTranscriptFeedScraper(feedURLs, keywords []string) *TranscriptFeedScraper {
	return &TranscriptFeedScraper{
		client:   &http.Client{Timeout: 15 * time.Second},
		feedURLs: feedURLs,
		keywords: keywords,
	}
}

func (t *TranscriptFeedScraper) Name() string {
	return "transcript-feed"
}

func (t *TranscriptFeedScraper) Scrape(ctx context.Context, limit int) ([]ScrapedPost, error) {
	if len(t.feedURLs) == 0 {
		return nil, nil
	}

	var posts []ScrapedPost
	for _, feedURL := range t.feedURLs {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
		if err != nil {
			return nil, err
		}
		resp, err := t.client.Do(req)
		if err != nil {
			return nil, err
		}

		var payload struct {
			Items []struct {
				ID          string `json:"id"`
				Platform    string `json:"platform"`
				Channel     string `json:"channel"`
				Title       string `json:"title"`
				Transcript  string `json:"transcript"`
				Content     string `json:"content"`
				URL         string `json:"url"`
				Author      string `json:"author"`
				PublishedAt string `json:"published_at"`
			} `json:"items"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		for index, item := range payload.Items {
			if limit > 0 && index >= limit {
				break
			}

			content := normalizeText(item.Transcript)
			if content == "" {
				content = normalizeText(item.Content)
			}

			matches := matchedKeywords(t.keywords, item.Title, content)
			if len(t.keywords) > 0 && len(matches) == 0 {
				continue
			}

			posts = append(posts, ScrapedPost{
				ID:              ensurePostID(item.Platform, item.Channel, item.ID, item.URL, item.Title),
				Platform:        defaultString(item.Platform, "Transcript Feed"),
				Channel:         item.Channel,
				ContentKind:     "transcript",
				Title:           normalizeText(item.Title),
				Content:         content,
				URL:             item.URL,
				Author:          item.Author,
				PublishedAt:     parseFlexibleTime(item.PublishedAt),
				MatchedKeywords: matches,
			})
		}
	}

	return posts, nil
}
