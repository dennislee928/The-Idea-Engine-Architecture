package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func NewRouter(
	db *DB,
	broadcaster *Broadcaster,
	internalToken string,
	triggerIngestion func(context.Context) (IngestionResult, error),
	embedder Embedder,
) *gin.Engine {
	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Internal-Token")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().UTC()})
	})

	r.GET("/api/insights", func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
		includeExplicit := strings.EqualFold(c.DefaultQuery("include_explicit", "false"), "true")

		insights, err := db.GetLatestInsights(c.Request.Context(), limit, includeExplicit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, insights)
	})

	r.GET("/api/insights/:id/similar", func(c *gin.Context) {
		insightID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid insight id"})
			return
		}

		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "8"))
		results, err := db.GetSimilarInsights(c.Request.Context(), insightID, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, results)
	})

	r.GET("/api/stats", func(c *gin.Context) {
		stats, err := db.GetStats(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, stats)
	})

	r.GET("/api/trends", func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "12"))
		windowHours, _ := strconv.Atoi(c.DefaultQuery("window_hours", "168"))

		trends, err := db.GetTrendClusters(c.Request.Context(), windowHours, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, trends)
	})

	r.GET("/api/search", func(c *gin.Context) {
		query := strings.TrimSpace(c.Query("q"))
		if query == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing q parameter"})
			return
		}

		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "8"))
		embedding, err := embedder.EmbedQuery(c.Request.Context(), query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		results, err := db.SemanticSearch(c.Request.Context(), embedding, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, results)
	})

	r.GET("/api/stream", func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")

		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
			return
		}

		id, subscription := broadcaster.Subscribe()
		defer broadcaster.Unsubscribe(id)

		heartbeat := time.NewTicker(20 * time.Second)
		defer heartbeat.Stop()

		c.Stream(func(w io.Writer) bool {
			select {
			case <-c.Request.Context().Done():
				return false
			case insight, ok := <-subscription:
				if !ok {
					return false
				}
				c.SSEvent("insight", insight)
				flusher.Flush()
				return true
			case tick := <-heartbeat.C:
				c.SSEvent("ping", gin.H{"time": tick.UTC().Format(time.RFC3339)})
				flusher.Flush()
				return true
			}
		})
	})

	r.POST("/internal/ingest/run", func(c *gin.Context) {
		if internalToken != "" && c.GetHeader("X-Internal-Token") != internalToken {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid internal token"})
			return
		}

		result, err := triggerIngestion(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusAccepted, gin.H{
				"message": fmt.Sprintf("ingestion completed with warnings: %v", err),
				"result":  result,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "ingestion completed",
			"result":  result,
		})
	})

	return r
}
