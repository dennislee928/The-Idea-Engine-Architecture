package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ScrapedPost represents the normalized data structure sent to Kafka/LLM
type ScrapedPost struct {
	ID       string
	Platform string
	Title    string
	Content  string
	URL      string
}

// Scraper is the interface all platform crawlers must implement
type Scraper interface {
	Scrape(ctx context.Context, limit int) ([]ScrapedPost, error)
}

type DcardScraper struct {
	client *http.Client
	forum  string // e.g., "softwareengineer", "job"
}

func NewDcardScraper(forum string) *DcardScraper {
	return &DcardScraper{
		client: &http.Client{Timeout: 10 * time.Second},
		forum:  forum,
	}
}

func (d *DcardScraper) Scrape(ctx context.Context, limit int) ([]ScrapedPost, error) {
	// Fetch latest posts from a specific forum
	url := fmt.Sprintf("https://www.dcard.tw/_api/forums/%s/posts?limit=%d", d.forum, limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	// Mimic a real browser to avoid basic blocks
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Define an anonymous struct matching the Dcard JSON response format
	var dcardPosts []struct {
		ID      int    `json:"id"`
		Title   string `json:"title"`
		Excerpt string `json:"excerpt"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&dcardPosts); err != nil {
		return nil, err
	}

	var posts []ScrapedPost
	for _, p := range dcardPosts {
		posts = append(posts, ScrapedPost{
			ID:       fmt.Sprintf("%d", p.ID),
			Platform: "Dcard",
			Title:    p.Title,
			Content:  p.Excerpt, // Dcard's list API returns an excerpt. Detail API is needed for full content.
			URL:      fmt.Sprintf("https://www.dcard.tw/f/%s/p/%d", d.forum, p.ID),
		})
	}

	return posts, nil
}
