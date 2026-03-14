package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func StartServer(db *DB, port string) {
	r := gin.Default()

	// Simple CORS middleware for local frontend development
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	r.GET("/api/insights", func(c *gin.Context) {
		insights, err := db.GetLatestInsights(c.Request.Context(), 50)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, insights)
	})

	r.Run(":" + port)
}
