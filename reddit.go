package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type RedditScraper struct {
	client       *http.Client
	clientID     string
	clientSecret string
	userAgent    string
	subreddits   []string
	keywords     []string

	mu          sync.Mutex
	accessToken string
	expiresAt   time.Time
}

func NewRedditScraper(clientID, clientSecret, userAgent string, subreddits, keywords []string) *RedditScraper {
	return &RedditScraper{
		client:       &http.Client{Timeout: 15 * time.Second},
		clientID:     clientID,
		clientSecret: clientSecret,
		userAgent:    defaultString(userAgent, "idea-engine/0.1"),
		subreddits:   subreddits,
		keywords:     keywords,
	}
}

func (r *RedditScraper) Name() string {
	return "reddit"
}

func (r *RedditScraper) Enabled() bool {
	return strings.TrimSpace(r.clientID) != "" && strings.TrimSpace(r.clientSecret) != ""
}

func (r *RedditScraper) Scrape(ctx context.Context, limit int) ([]ScrapedPost, error) {
	if !r.Enabled() {
		return nil, nil
	}

	token, err := r.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	var posts []ScrapedPost
	for _, subreddit := range r.subreddits {
		endpoint := fmt.Sprintf("https://oauth.reddit.com/r/%s/new?limit=%d", url.PathEscape(subreddit), limit)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("User-Agent", r.userAgent)

		resp, err := r.client.Do(req)
		if err != nil {
			return nil, err
		}

		var payload struct {
			Data struct {
				Children []struct {
					Data struct {
						ID         string  `json:"id"`
						Title      string  `json:"title"`
						SelfText   string  `json:"selftext"`
						Permalink  string  `json:"permalink"`
						Author     string  `json:"author"`
						CreatedUTC float64 `json:"created_utc"`
					} `json:"data"`
				} `json:"children"`
			} `json:"data"`
		}

		err = json.NewDecoder(resp.Body).Decode(&payload)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		for _, child := range payload.Data.Children {
			matches := matchedKeywords(r.keywords, child.Data.Title, child.Data.SelfText)
			if len(r.keywords) > 0 && len(matches) == 0 {
				continue
			}

			posts = append(posts, ScrapedPost{
				ID:              child.Data.ID,
				Platform:        "Reddit",
				Channel:         "r/" + subreddit,
				ContentKind:     "post",
				Title:           normalizeText(child.Data.Title),
				Content:         normalizeText(child.Data.SelfText),
				URL:             "https://www.reddit.com" + child.Data.Permalink,
				Author:          child.Data.Author,
				PublishedAt:     time.Unix(int64(child.Data.CreatedUTC), 0).UTC(),
				MatchedKeywords: matches,
			})
		}
	}

	return posts, nil
}

func (r *RedditScraper) getAccessToken(ctx context.Context) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.accessToken != "" && time.Now().Before(r.expiresAt.Add(-30*time.Second)) {
		return r.accessToken, nil
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://www.reddit.com/api/v1/access_token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(r.clientID, r.clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", r.userAgent)

	resp, err := r.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var payload struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}

	r.accessToken = payload.AccessToken
	r.expiresAt = time.Now().Add(time.Duration(payload.ExpiresIn) * time.Second)
	return r.accessToken, nil
}
