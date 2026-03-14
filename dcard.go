package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Scraper is the interface all platform crawlers must implement
type Scraper interface {
	Name() string
	Scrape(ctx context.Context, limit int) ([]ScrapedPost, error)
}

type DcardScraper struct {
	client   *http.Client
	forums   []string
	keywords []string
}

func NewDcardScraper(forums, keywords []string) *DcardScraper {
	return &DcardScraper{
		client:   &http.Client{Timeout: 10 * time.Second},
		forums:   forums,
		keywords: keywords,
	}
}

func (d *DcardScraper) Name() string {
	return "dcard"
}

func (d *DcardScraper) Scrape(ctx context.Context, limit int) ([]ScrapedPost, error) {
	var posts []ScrapedPost
	for _, forum := range d.forums {
		listURL := fmt.Sprintf("https://www.dcard.tw/_api/forums/%s/posts?limit=%d", forum, limit)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

		resp, err := d.client.Do(req)
		if err != nil {
			return nil, err
		}

		var dcardPosts []struct {
			ID        int    `json:"id"`
			Title     string `json:"title"`
			Excerpt   string `json:"excerpt"`
			CreatedAt string `json:"createdAt"`
			School    string `json:"school"`
		}

		err = json.NewDecoder(resp.Body).Decode(&dcardPosts)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		for _, p := range dcardPosts {
			content := normalizeText(p.Excerpt)
			if detailContent, err := d.fetchPostContent(ctx, p.ID); err == nil && detailContent != "" {
				content = detailContent
			}

			matches := matchedKeywords(d.keywords, p.Title, content)
			if len(d.keywords) > 0 && len(matches) == 0 {
				continue
			}

			author := normalizeText(p.School)
			if strings.EqualFold(author, "anonymous") {
				author = ""
			}

			posts = append(posts, ScrapedPost{
				ID:              fmt.Sprintf("%d", p.ID),
				Platform:        "Dcard",
				Channel:         forum,
				ContentKind:     "post",
				Title:           normalizeText(p.Title),
				Content:         content,
				URL:             fmt.Sprintf("https://www.dcard.tw/f/%s/p/%d", forum, p.ID),
				Author:          author,
				PublishedAt:     parseFlexibleTime(p.CreatedAt),
				MatchedKeywords: matches,
			})
		}
	}

	return posts, nil
}

func (d *DcardScraper) fetchPostContent(ctx context.Context, postID int) (string, error) {
	detailURL := fmt.Sprintf("https://www.dcard.tw/_api/posts/%d", postID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, detailURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := d.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var payload struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	return normalizeText(payload.Content), nil
}
