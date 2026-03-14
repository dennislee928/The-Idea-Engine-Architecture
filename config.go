package main

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ServerPort          string
	DatabaseURL         string
	RedisURL            string
	KafkaBroker         string
	KafkaTopic          string
	KafkaGroupID        string
	DedupTTL            time.Duration
	IngestionInterval   time.Duration
	IngestionBatchSize  int
	IntelligenceWorkers int
	InternalAPIToken    string

	LLMProvider string

	EmbeddingProvider   string
	EmbeddingDimensions int

	GeminiAPIKey         string
	GeminiModel          string
	GeminiEmbeddingModel string

	GroqAPIKey  string
	GroqModel   string
	GroqBaseURL string

	DcardForums   []string
	DcardKeywords []string

	RedditClientID     string
	RedditClientSecret string
	RedditUserAgent    string
	RedditSubreddits   []string
	RedditKeywords     []string

	AppStoreFeeds    []string
	AppStoreKeywords []string

	TranscriptFeedURLs []string
	TranscriptKeywords []string
}

func LoadConfig() Config {
	return Config{
		ServerPort:           envString("PORT", "8080"),
		DatabaseURL:          envString("DATABASE_URL", "postgres://idea_admin:idea_password@localhost:5432/idea_engine?sslmode=disable"),
		RedisURL:             envString("REDIS_URL", "redis://localhost:6379/0"),
		KafkaBroker:          envString("KAFKA_BROKER", "localhost:9092"),
		KafkaTopic:           envString("KAFKA_TOPIC", "raw-posts"),
		KafkaGroupID:         envString("KAFKA_GROUP_ID", "idea-engine-llm-group"),
		DedupTTL:             envDuration("DEDUP_TTL", 7*24*time.Hour),
		IngestionInterval:    envDuration("INGESTION_INTERVAL", 15*time.Minute),
		IngestionBatchSize:   envInt("INGESTION_BATCH_SIZE", 20),
		IntelligenceWorkers:  envInt("INTELLIGENCE_WORKERS", 3),
		InternalAPIToken:     envString("INTERNAL_API_TOKEN", ""),
		LLMProvider:          strings.ToLower(envString("LLM_PROVIDER", "mock")),
		EmbeddingProvider:    strings.ToLower(envString("EMBEDDING_PROVIDER", "mock")),
		EmbeddingDimensions:  envInt("EMBEDDING_DIMENSIONS", 256),
		GeminiAPIKey:         envString("GEMINI_API_KEY", ""),
		GeminiModel:          envString("GEMINI_MODEL", "gemini-1.5-flash"),
		GeminiEmbeddingModel: envString("GEMINI_EMBEDDING_MODEL", "gemini-embedding-001"),
		GroqAPIKey:           envString("GROQ_API_KEY", ""),
		GroqModel:            envString("GROQ_MODEL", "llama-3.1-8b-instant"),
		GroqBaseURL:          envString("GROQ_BASE_URL", "https://api.groq.com/openai/v1/chat/completions"),
		DcardForums:          envList("DCARD_FORUMS", []string{"softwareengineer", "job"}),
		DcardKeywords: envList("DCARD_KEYWORDS", []string{
			"求救", "有人也這樣嗎", "手動", "好累", "卡住", "崩潰",
		}),
		RedditClientID:     envString("REDDIT_CLIENT_ID", ""),
		RedditClientSecret: envString("REDDIT_CLIENT_SECRET", ""),
		RedditUserAgent:    envString("REDDIT_USER_AGENT", "idea-engine/0.1"),
		RedditSubreddits:   envList("REDDIT_SUBREDDITS", []string{"SmallBusiness", "Excel", "RemoteWork"}),
		RedditKeywords: envList("REDDIT_KEYWORDS", []string{
			"manual", "spreadsheet", "frustrated", "pain", "workaround", "too slow",
		}),
		AppStoreFeeds:      envList("APP_STORE_FEEDS", nil),
		AppStoreKeywords:   envList("APP_STORE_KEYWORDS", []string{"slow", "expensive", "subscription", "too many features"}),
		TranscriptFeedURLs: envList("TRANSCRIPT_FEED_URLS", nil),
		TranscriptKeywords: envList("TRANSCRIPT_KEYWORDS", []string{
			"copy paste", "manual", "workflow", "hack", "spreadsheet",
		}),
	}
}

func envString(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envList(key string, fallback []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}

	if len(result) == 0 {
		return fallback
	}

	return result
}
