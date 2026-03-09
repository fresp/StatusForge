package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"status-platform/internal/database"
)

type HealthResponse struct {
	Status  string `json:"status"`
	MongoDB string `json:"mongodb"`
	Redis   string `json:"redis"`
}

func HealthCheckHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		mongoStatus := "connected"
		redisStatus := "connected"

		// Ping MongoDB
		db := database.GetDB()
		if db != nil {
			if err := db.Client().Ping(ctx, nil); err != nil {
				mongoStatus = "error"
				log.Printf("[HTTP] MongoDB ping failed: %v", err)
			}
		} else {
			mongoStatus = "disconnected"
			log.Printf("[HTTP] MongoDB is not connected")
		}

		// Ping Redis
		rdb := database.GetRedis()
		if rdb != nil {
			if _, err := rdb.Ping(ctx).Result(); err != nil {
				redisStatus = "error"
				log.Printf("[HTTP] Redis ping failed: %v", err)
			}
		} else {
			redisStatus = "disconnected"
			log.Printf("[HTTP] Redis is not connected")
		}

		response := HealthResponse{
			Status:  "healthy",
			MongoDB: mongoStatus,
			Redis:   redisStatus,
		}

		if mongoStatus != "connected" || redisStatus != "connected" {
			response.Status = "unhealthy"
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, response)
			return
		}

		c.JSON(http.StatusOK, response)
	}
}
