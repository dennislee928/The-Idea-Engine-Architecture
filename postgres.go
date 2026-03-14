package storage

import (
	"context"
	"database/sql"
	"log"

	"github.com/dennis_lee/idea-engine/backend/internal/analyzer" // Adjust import path as needed
	_ "github.com/lib/pq"
)

type DB struct {
	conn *sql.DB
}

type DBInsight struct {
	ID                  int    `json:"id"`
	Platform            string `json:"platform"`
	SourceURL           string `json:"source_url"`
	CorePainPoint       string `json:"core_pain_point"`
	CurrentWorkaround   string `json:"current_workaround"`
	CommercialPotential int    `json:"commercial_potential"`
	SaaSFeasibility     string `json:"saas_feasibility"`
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
	instance.migrate()

	return instance, nil
}

func (db *DB) migrate() {
	// Basic migration; in production, use a tool like golang-migrate
	query := `
	CREATE EXTENSION IF NOT EXISTS vector;
	CREATE TABLE IF NOT EXISTS insights (
		id SERIAL PRIMARY KEY,
		platform VARCHAR(50),
		source_url TEXT,
		core_pain_point TEXT,
		current_workaround TEXT,
		commercial_potential INT,
		saas_feasibility TEXT,
		is_explicit_content BOOLEAN,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := db.conn.Exec(query)
	if err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
}

func (db *DB) SaveInsight(ctx context.Context, platform, url string, insight *analyzer.Insight) error {
	query := `
		INSERT INTO insights (platform, source_url, core_pain_point, current_workaround, commercial_potential, saas_feasibility, is_explicit_content)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := db.conn.ExecContext(ctx, query,
		platform, url,
		insight.CorePainPoint, insight.CurrentWorkaround,
		insight.CommercialPotential, insight.SaaSFeasibility, insight.IsExplicitContent,
	)
	return err
}

func (db *DB) GetLatestInsights(ctx context.Context, limit int) ([]DBInsight, error) {
	query := `
		SELECT id, platform, source_url, core_pain_point, current_workaround, commercial_potential, saas_feasibility
		FROM insights
		ORDER BY created_at DESC
		LIMIT $1
	`
	rows, err := db.conn.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var insights []DBInsight
	for rows.Next() {
		var i DBInsight
		if err := rows.Scan(&i.ID, &i.Platform, &i.SourceURL, &i.CorePainPoint, &i.CurrentWorkaround, &i.CommercialPotential, &i.SaaSFeasibility); err != nil {
			return nil, err
		}
		insights = append(insights, i)
	}
	return insights, nil
}
