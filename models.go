package main

import (
	"crypto/sha1"
	"encoding/hex"
	"strings"
	"time"
)

type ScrapedPost struct {
	ID              string    `json:"id"`
	Platform        string    `json:"platform"`
	Channel         string    `json:"channel"`
	ContentKind     string    `json:"content_kind"`
	Title           string    `json:"title"`
	Content         string    `json:"content"`
	URL             string    `json:"url"`
	Author          string    `json:"author"`
	PublishedAt     time.Time `json:"published_at"`
	MatchedKeywords []string  `json:"matched_keywords"`
}

type QueueMessage struct {
	ID              string    `json:"id"`
	Platform        string    `json:"platform"`
	Channel         string    `json:"channel"`
	ContentKind     string    `json:"content_kind"`
	Title           string    `json:"title"`
	Content         string    `json:"content"`
	URL             string    `json:"url"`
	Author          string    `json:"author"`
	PublishedAt     time.Time `json:"published_at"`
	MatchedKeywords []string  `json:"matched_keywords"`
	IngestedAt      time.Time `json:"ingested_at"`
}

type Insight struct {
	CorePainPoint       string `json:"core_pain_point"`
	CurrentWorkaround   string `json:"current_workaround"`
	CommercialPotential int    `json:"commercial_potential"`
	SaaSFeasibility     string `json:"saas_feasibility"`
	IsExplicitContent   bool   `json:"is_explicit_content"`
}

type DBInsight struct {
	ID                  int       `json:"id"`
	Platform            string    `json:"platform"`
	Channel             string    `json:"channel"`
	ContentKind         string    `json:"content_kind"`
	ClusterKey          string    `json:"cluster_key"`
	ClusterLabel        string    `json:"cluster_label"`
	SourcePostID        string    `json:"source_post_id"`
	Title               string    `json:"title"`
	SourceURL           string    `json:"source_url"`
	Author              string    `json:"author"`
	RawContent          string    `json:"raw_content"`
	CorePainPoint       string    `json:"core_pain_point"`
	CurrentWorkaround   string    `json:"current_workaround"`
	CommercialPotential int       `json:"commercial_potential"`
	SaaSFeasibility     string    `json:"saas_feasibility"`
	IsExplicitContent   bool      `json:"is_explicit_content"`
	MatchedKeywords     []string  `json:"matched_keywords"`
	AnalysisModel       string    `json:"analysis_model"`
	EmbeddingModel      string    `json:"embedding_model"`
	PublishedAt         time.Time `json:"published_at"`
	CreatedAt           time.Time `json:"created_at"`
	Similarity          float64   `json:"similarity,omitempty"`
}

type InsightStats struct {
	TotalInsights    int     `json:"total_insights"`
	LiveLast24h      int     `json:"live_last_24h"`
	AveragePotential float64 `json:"average_potential"`
	TopPlatform      string  `json:"top_platform"`
}

type TrendCluster struct {
	ClusterKey   string    `json:"cluster_key"`
	ClusterLabel string    `json:"cluster_label"`
	InsightCount int       `json:"insight_count"`
	AverageScore float64   `json:"average_score"`
	TopScore     int       `json:"top_score"`
	LatestSeenAt time.Time `json:"latest_seen_at"`
	Platforms    []string  `json:"platforms"`
	SampleTitles []string  `json:"sample_titles"`
	SamplePain   []string  `json:"sample_pain_points"`
}

type IngestionResult struct {
	Sources           int      `json:"sources"`
	Fetched           int      `json:"fetched"`
	Enqueued          int      `json:"enqueued"`
	SkippedDuplicates int      `json:"skipped_duplicates"`
	Errors            []string `json:"errors,omitempty"`
}

func (p ScrapedPost) ToQueueMessage() QueueMessage {
	return QueueMessage{
		ID:              ensurePostID(p.Platform, p.Channel, p.ID, p.URL, p.Title),
		Platform:        p.Platform,
		Channel:         p.Channel,
		ContentKind:     defaultString(p.ContentKind, "post"),
		Title:           p.Title,
		Content:         p.Content,
		URL:             p.URL,
		Author:          p.Author,
		PublishedAt:     p.PublishedAt.UTC(),
		MatchedKeywords: p.MatchedKeywords,
		IngestedAt:      time.Now().UTC(),
	}
}

func (m QueueMessage) ToScrapedPost() ScrapedPost {
	return ScrapedPost{
		ID:              m.ID,
		Platform:        m.Platform,
		Channel:         m.Channel,
		ContentKind:     m.ContentKind,
		Title:           m.Title,
		Content:         m.Content,
		URL:             m.URL,
		Author:          m.Author,
		PublishedAt:     m.PublishedAt,
		MatchedKeywords: m.MatchedKeywords,
	}
}

func ensurePostID(platform, channel, preferred, url, title string) string {
	preferred = strings.TrimSpace(preferred)
	if preferred != "" {
		return preferred
	}

	hasher := sha1.New()
	hasher.Write([]byte(strings.Join([]string{platform, channel, url, title}, "::")))
	return hex.EncodeToString(hasher.Sum(nil))
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
