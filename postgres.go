package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/lib/pq"
)

type DB struct {
	conn *sql.DB
}

func NewDB(dsn string) (*DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	instance := &DB{conn: db}
	if err := instance.migrate(); err != nil {
		return nil, err
	}

	return instance, nil
}

func (db *DB) migrate() error {
	if _, err := db.conn.Exec(`CREATE EXTENSION IF NOT EXISTS vector;`); err != nil {
		log.Printf("pgvector extension not available yet: %v", err)
	}

	statements := []string{
		`CREATE TABLE IF NOT EXISTS insights (
			id SERIAL PRIMARY KEY,
			platform TEXT NOT NULL,
			channel TEXT,
			content_kind TEXT NOT NULL DEFAULT 'post',
			cluster_key TEXT,
			cluster_label TEXT,
			source_post_id TEXT,
			title TEXT,
			source_url TEXT NOT NULL,
			author TEXT,
			raw_content TEXT,
			core_pain_point TEXT,
			current_workaround TEXT,
			commercial_potential INT,
			saas_feasibility TEXT,
			is_explicit_content BOOLEAN DEFAULT FALSE,
			matched_keywords TEXT[] DEFAULT ARRAY[]::TEXT[],
			analysis_model TEXT,
			published_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		);`,
		`ALTER TABLE insights ADD COLUMN IF NOT EXISTS channel TEXT;`,
		`ALTER TABLE insights ADD COLUMN IF NOT EXISTS content_kind TEXT NOT NULL DEFAULT 'post';`,
		`ALTER TABLE insights ADD COLUMN IF NOT EXISTS cluster_key TEXT;`,
		`ALTER TABLE insights ADD COLUMN IF NOT EXISTS cluster_label TEXT;`,
		`ALTER TABLE insights ADD COLUMN IF NOT EXISTS source_post_id TEXT;`,
		`ALTER TABLE insights ADD COLUMN IF NOT EXISTS title TEXT;`,
		`ALTER TABLE insights ADD COLUMN IF NOT EXISTS author TEXT;`,
		`ALTER TABLE insights ADD COLUMN IF NOT EXISTS raw_content TEXT;`,
		`ALTER TABLE insights ADD COLUMN IF NOT EXISTS matched_keywords TEXT[] DEFAULT ARRAY[]::TEXT[];`,
		`ALTER TABLE insights ADD COLUMN IF NOT EXISTS analysis_model TEXT;`,
		`ALTER TABLE insights ADD COLUMN IF NOT EXISTS published_at TIMESTAMPTZ;`,
		`CREATE UNIQUE INDEX IF NOT EXISTS insights_source_unique_idx ON insights (platform, source_post_id);`,
		`CREATE INDEX IF NOT EXISTS insights_created_at_idx ON insights (created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS insights_explicit_idx ON insights (is_explicit_content);`,
		`CREATE INDEX IF NOT EXISTS insights_cluster_idx ON insights (cluster_key, created_at DESC);`,
	}

	for _, statement := range statements {
		if _, err := db.conn.Exec(statement); err != nil {
			return err
		}
	}

	return nil
}

func (db *DB) SaveInsight(ctx context.Context, post ScrapedPost, insight *Insight, model string) (DBInsight, error) {
	clusterKey, clusterLabel := BuildPainCluster(insight.CorePainPoint, post.MatchedKeywords)

	query := `
		INSERT INTO insights (
			platform,
			channel,
			content_kind,
			cluster_key,
			cluster_label,
			source_post_id,
			title,
			source_url,
			author,
			raw_content,
			core_pain_point,
			current_workaround,
			commercial_potential,
			saas_feasibility,
			is_explicit_content,
			matched_keywords,
			analysis_model,
			published_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		ON CONFLICT (platform, source_post_id)
		DO UPDATE SET
			channel = EXCLUDED.channel,
			content_kind = EXCLUDED.content_kind,
			cluster_key = EXCLUDED.cluster_key,
			cluster_label = EXCLUDED.cluster_label,
			title = EXCLUDED.title,
			source_url = EXCLUDED.source_url,
			author = EXCLUDED.author,
			raw_content = EXCLUDED.raw_content,
			core_pain_point = EXCLUDED.core_pain_point,
			current_workaround = EXCLUDED.current_workaround,
			commercial_potential = EXCLUDED.commercial_potential,
			saas_feasibility = EXCLUDED.saas_feasibility,
			is_explicit_content = EXCLUDED.is_explicit_content,
			matched_keywords = EXCLUDED.matched_keywords,
			analysis_model = EXCLUDED.analysis_model,
			published_at = EXCLUDED.published_at
		RETURNING
			id,
			platform,
			channel,
			content_kind,
			COALESCE(cluster_key, ''),
			COALESCE(cluster_label, ''),
			source_post_id,
			title,
			source_url,
			author,
			raw_content,
			core_pain_point,
			current_workaround,
			commercial_potential,
			saas_feasibility,
			is_explicit_content,
			matched_keywords,
			analysis_model,
			COALESCE(published_at, CURRENT_TIMESTAMP),
			created_at
	`

	var record DBInsight
	err := db.conn.QueryRowContext(
		ctx,
		query,
		post.Platform,
		post.Channel,
		defaultString(post.ContentKind, "post"),
		clusterKey,
		clusterLabel,
		ensurePostID(post.Platform, post.Channel, post.ID, post.URL, post.Title),
		post.Title,
		post.URL,
		post.Author,
		post.Content,
		insight.CorePainPoint,
		insight.CurrentWorkaround,
		insight.CommercialPotential,
		insight.SaaSFeasibility,
		insight.IsExplicitContent,
		pq.Array(post.MatchedKeywords),
		model,
		post.PublishedAt.UTC(),
	).Scan(
		&record.ID,
		&record.Platform,
		&record.Channel,
		&record.ContentKind,
		&record.ClusterKey,
		&record.ClusterLabel,
		&record.SourcePostID,
		&record.Title,
		&record.SourceURL,
		&record.Author,
		&record.RawContent,
		&record.CorePainPoint,
		&record.CurrentWorkaround,
		&record.CommercialPotential,
		&record.SaaSFeasibility,
		&record.IsExplicitContent,
		pq.Array(&record.MatchedKeywords),
		&record.AnalysisModel,
		&record.PublishedAt,
		&record.CreatedAt,
	)
	return record, err
}

func (db *DB) GetLatestInsights(ctx context.Context, limit int, includeExplicit bool) ([]DBInsight, error) {
	limit = clampLimit(limit)

	query := `
		SELECT
			id,
			platform,
			channel,
			content_kind,
			COALESCE(cluster_key, ''),
			COALESCE(cluster_label, ''),
			COALESCE(source_post_id, ''),
			COALESCE(title, ''),
			source_url,
			COALESCE(author, ''),
			COALESCE(raw_content, ''),
			COALESCE(core_pain_point, ''),
			COALESCE(current_workaround, ''),
			COALESCE(commercial_potential, 0),
			COALESCE(saas_feasibility, ''),
			COALESCE(is_explicit_content, FALSE),
			COALESCE(matched_keywords, ARRAY[]::TEXT[]),
			COALESCE(analysis_model, ''),
			COALESCE(published_at, created_at),
			created_at
		FROM insights
	`
	if !includeExplicit {
		query += ` WHERE COALESCE(is_explicit_content, FALSE) = FALSE `
	}
	query += ` ORDER BY created_at DESC LIMIT $1`

	rows, err := db.conn.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var insights []DBInsight
	for rows.Next() {
		var insight DBInsight
		if err := rows.Scan(
			&insight.ID,
			&insight.Platform,
			&insight.Channel,
			&insight.ContentKind,
			&insight.ClusterKey,
			&insight.ClusterLabel,
			&insight.SourcePostID,
			&insight.Title,
			&insight.SourceURL,
			&insight.Author,
			&insight.RawContent,
			&insight.CorePainPoint,
			&insight.CurrentWorkaround,
			&insight.CommercialPotential,
			&insight.SaaSFeasibility,
			&insight.IsExplicitContent,
			pq.Array(&insight.MatchedKeywords),
			&insight.AnalysisModel,
			&insight.PublishedAt,
			&insight.CreatedAt,
		); err != nil {
			return nil, err
		}
		insights = append(insights, insight)
	}

	return insights, rows.Err()
}

func (db *DB) GetStats(ctx context.Context) (InsightStats, error) {
	var stats InsightStats

	err := db.conn.QueryRowContext(ctx, `
		SELECT
			COUNT(*)::INT,
			COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '24 hours')::INT,
			COALESCE(ROUND(AVG(commercial_potential)::numeric, 1), 0)
		FROM insights
		WHERE COALESCE(is_explicit_content, FALSE) = FALSE
	`).Scan(&stats.TotalInsights, &stats.LiveLast24h, &stats.AveragePotential)
	if err != nil {
		return stats, err
	}

	err = db.conn.QueryRowContext(ctx, `
		SELECT COALESCE(platform, '')
		FROM insights
		WHERE COALESCE(is_explicit_content, FALSE) = FALSE
		GROUP BY platform
		ORDER BY COUNT(*) DESC, platform ASC
		LIMIT 1
	`).Scan(&stats.TopPlatform)
	if err != nil && err != sql.ErrNoRows {
		return stats, err
	}

	return stats, nil
}

func (db *DB) GetTrendClusters(ctx context.Context, windowHours, limit int) ([]TrendCluster, error) {
	if windowHours < 1 {
		windowHours = 24
	}
	limit = clampLimit(limit)

	rows, err := db.conn.QueryContext(ctx, `
		SELECT
			cluster_key,
			COALESCE(MAX(NULLIF(cluster_label, '')), cluster_key) AS cluster_label,
			COUNT(*)::INT AS insight_count,
			COALESCE(ROUND(AVG(commercial_potential)::numeric, 1), 0) AS average_score,
			COALESCE(MAX(commercial_potential), 0)::INT AS top_score,
			MAX(created_at) AS latest_seen_at,
			COALESCE(ARRAY_AGG(DISTINCT platform), ARRAY[]::TEXT[]) AS platforms,
			COALESCE(ARRAY_REMOVE(ARRAY_AGG(DISTINCT NULLIF(title, '')), NULL), ARRAY[]::TEXT[]) AS sample_titles,
			COALESCE(ARRAY_REMOVE(ARRAY_AGG(DISTINCT NULLIF(core_pain_point, '')), NULL), ARRAY[]::TEXT[]) AS sample_pain_points
		FROM insights
		WHERE COALESCE(is_explicit_content, FALSE) = FALSE
		  AND COALESCE(cluster_key, '') <> ''
		  AND created_at >= NOW() - ($1::INT * INTERVAL '1 hour')
		GROUP BY cluster_key
		HAVING COUNT(*) >= 1
		ORDER BY insight_count DESC, average_score DESC, latest_seen_at DESC
		LIMIT $2
	`, windowHours, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trends []TrendCluster
	for rows.Next() {
		var trend TrendCluster
		if err := rows.Scan(
			&trend.ClusterKey,
			&trend.ClusterLabel,
			&trend.InsightCount,
			&trend.AverageScore,
			&trend.TopScore,
			&trend.LatestSeenAt,
			pq.Array(&trend.Platforms),
			pq.Array(&trend.SampleTitles),
			pq.Array(&trend.SamplePain),
		); err != nil {
			return nil, err
		}

		if len(trend.SampleTitles) > 3 {
			trend.SampleTitles = trend.SampleTitles[:3]
		}
		if len(trend.SamplePain) > 3 {
			trend.SamplePain = trend.SamplePain[:3]
		}
		trends = append(trends, trend)
	}

	return trends, rows.Err()
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func clampLimit(limit int) int {
	switch {
	case limit < 1:
		return 20
	case limit > 200:
		return 200
	default:
		return limit
	}
}

func formatMatchedKeywords(keywords []string) string {
	return strings.Join(keywords, ", ")
}

func (db *DB) String() string {
	return fmt.Sprintf("DB<connected=%t>", db != nil && db.conn != nil)
}
