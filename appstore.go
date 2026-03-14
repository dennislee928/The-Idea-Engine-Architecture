package main

import (
	"context"
	"encoding/xml"
	"net/http"
	"strings"
	"time"
)

type AppStoreScraper struct {
	client   *http.Client
	feedURLs []string
	keywords []string
}

func NewAppStoreScraper(feedURLs, keywords []string) *AppStoreScraper {
	return &AppStoreScraper{
		client:   &http.Client{Timeout: 15 * time.Second},
		feedURLs: feedURLs,
		keywords: keywords,
	}
}

func (a *AppStoreScraper) Name() string {
	return "app-store"
}

func (a *AppStoreScraper) Scrape(ctx context.Context, limit int) ([]ScrapedPost, error) {
	if len(a.feedURLs) == 0 {
		return nil, nil
	}

	var posts []ScrapedPost
	for _, feedURL := range a.feedURLs {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
		if err != nil {
			return nil, err
		}
		resp, err := a.client.Do(req)
		if err != nil {
			return nil, err
		}

		var atom struct {
			Title   string `xml:"title"`
			Entries []struct {
				ID      string `xml:"id"`
				Title   string `xml:"title"`
				Content string `xml:"content"`
				Summary string `xml:"summary"`
				Updated string `xml:"updated"`
				Author  struct {
					Name string `xml:"name"`
				} `xml:"author"`
				Link struct {
					Href string `xml:"href,attr"`
				} `xml:"link"`
			} `xml:"entry"`
		}

		err = xml.NewDecoder(resp.Body).Decode(&atom)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		channelTitle := normalizeText(atom.Title)
		for index, entry := range atom.Entries {
			if limit > 0 && index >= limit {
				break
			}

			content := normalizeText(strings.Join([]string{entry.Summary, entry.Content}, " "))
			matches := matchedKeywords(a.keywords, entry.Title, content)
			if len(a.keywords) > 0 && len(matches) == 0 {
				continue
			}

			posts = append(posts, ScrapedPost{
				ID:              ensurePostID("App Store", channelTitle, entry.ID, entry.Link.Href, entry.Title),
				Platform:        "App Store",
				Channel:         channelTitle,
				ContentKind:     "review",
				Title:           normalizeText(entry.Title),
				Content:         content,
				URL:             entry.Link.Href,
				Author:          entry.Author.Name,
				PublishedAt:     parseFlexibleTime(entry.Updated),
				MatchedKeywords: matches,
			})
		}
	}

	return posts, nil
}
