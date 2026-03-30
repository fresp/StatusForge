package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	shared "github.com/fresp/StatusForge/internal/domain/shared"
	"github.com/fresp/StatusForge/internal/models"
	"github.com/fresp/StatusForge/internal/repository"
	monitorservice "github.com/fresp/StatusForge/internal/services/monitor"
	"github.com/fresp/StatusForge/internal/utils"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func GetMonitors(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, limit, err := parsePaginationParams(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		service := monitorservice.NewService(repository.NewMongoMonitorRepository(db))
		monitors, total, err := service.List(ctx, page, limit)
		if err != nil {
			writeDomainError(c, err)
			return
		}

		if monitors == nil {
			monitors = []models.Monitor{}
		}

		writePaginatedResponse(c, monitors, int(total), page, limit)
	}
}

func CreateMonitor(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name            string               `json:"name" binding:"required"`
			Type            models.MonitorType   `json:"type" binding:"required"`
			Target          string               `json:"target" binding:"required"`
			Monitoring      models.MonitorConfig `json:"monitoring"`
			SSLThresholds   []int                `json:"sslThresholds"`
			IntervalSeconds int                  `json:"intervalSeconds"`
			TimeoutSeconds  int                  `json:"timeoutSeconds"`
			ComponentID     string               `json:"componentId"`
			SubComponentID  string               `json:"subComponentId"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		service := monitorservice.NewService(repository.NewMongoMonitorRepository(db))
		monitor, err := service.Create(ctx, monitorservice.MonitorUpsertInput{
			Name:            req.Name,
			Type:            req.Type,
			Target:          req.Target,
			Monitoring:      req.Monitoring,
			SSLThresholds:   req.SSLThresholds,
			IntervalSeconds: req.IntervalSeconds,
			TimeoutSeconds:  req.TimeoutSeconds,
			ComponentID:     req.ComponentID,
			SubComponentID:  req.SubComponentID,
		})
		if err != nil {
			if monitorservice.IsValidationError(err) {
				writeDomainError(c, errorsWrap(err, shared.ErrInvalidInput))
				return
			}
			writeDomainError(c, err)
			return
		}

		c.JSON(http.StatusCreated, monitor)
	}
}

func UpdateMonitor(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}

		var req struct {
			Name            string               `json:"name" binding:"required"`
			Type            models.MonitorType   `json:"type" binding:"required"`
			Target          string               `json:"target" binding:"required"`
			Monitoring      models.MonitorConfig `json:"monitoring"`
			SSLThresholds   []int                `json:"sslThresholds"`
			IntervalSeconds int                  `json:"intervalSeconds"`
			TimeoutSeconds  int                  `json:"timeoutSeconds"`
			ComponentID     string               `json:"componentId"`
			SubComponentID  string               `json:"subComponentId"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		service := monitorservice.NewService(repository.NewMongoMonitorRepository(db))
		matched, err := service.Update(ctx, id, monitorservice.MonitorUpsertInput{
			Name:            req.Name,
			Type:            req.Type,
			Target:          req.Target,
			Monitoring:      req.Monitoring,
			SSLThresholds:   req.SSLThresholds,
			IntervalSeconds: req.IntervalSeconds,
			TimeoutSeconds:  req.TimeoutSeconds,
			ComponentID:     req.ComponentID,
			SubComponentID:  req.SubComponentID,
		})
		if err != nil {
			if monitorservice.IsValidationError(err) {
				writeDomainError(c, errorsWrap(err, shared.ErrInvalidInput))
				return
			}
			writeDomainError(c, err)
			return
		}
		if !matched {
			writeDomainError(c, errorsWrap(shared.ErrNotFound, errMonitorNotFound))
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "updated"})
	}
}

func TestMonitor() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Type           models.MonitorType   `json:"type" binding:"required"`
			Target         string               `json:"target" binding:"required"`
			Monitoring     models.MonitorConfig `json:"monitoring"`
			SSLThresholds  []int                `json:"sslThresholds"`
			TimeoutSeconds int                  `json:"timeoutSeconds"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		timeout := time.Duration(req.TimeoutSeconds) * time.Second
		if timeout == 0 {
			timeout = 30 * time.Second
		}

		start := time.Now()
		status := models.MonitorUp
		statusCode := 0
		sslWarning := false
		sslDaysRemaining := 0
		sslTriggeredThreshold := 0
		domainWarning := false
		domainDaysRemaining := 0
		domainTriggeredThreshold := 0
		thresholds := monitorservice.SanitizeSSLThresholds(req.SSLThresholds)

		if err := monitorservice.ValidateAdvancedOptions(req.Type, req.Target, req.Monitoring.Advanced); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		switch req.Type {
		case models.MonitorHTTP:
			code, err := utils.CheckHTTP(req.Target, timeout, req.Monitoring.Advanced.IgnoreTLSError)
			statusCode = code
			if err != nil || code >= 500 || code == 0 {
				status = models.MonitorDown
			}
			if req.Monitoring.Advanced.CertExpiry {
				result, sslErr := utils.CheckHTTPSSLCertificate(req.Target, timeout, thresholds)
				if sslErr != nil {
					status = models.MonitorDown
				} else {
					sslWarning = result.Warning
					sslDaysRemaining = result.DaysRemaining
					sslTriggeredThreshold = result.TriggeredThreshold
				}
			}
			if req.Monitoring.Advanced.DomainExpiry {
				result, domainErr := utils.CheckDomain(req.Target, string(req.Type), thresholds)
				if domainErr != nil {
					status = models.MonitorDown
				} else {
					domainWarning = result.Warning
					domainDaysRemaining = result.DaysRemaining
					domainTriggeredThreshold = result.TriggeredThreshold
				}
			}
		case models.MonitorTCP:
			if err := utils.CheckTCP(req.Target, timeout); err != nil {
				status = models.MonitorDown
			}
		case models.MonitorDNS:
			if err := utils.CheckDNS(req.Target, timeout); err != nil {
				status = models.MonitorDown
			}
		case models.MonitorPing:
			if err := utils.CheckPing(req.Target, timeout); err != nil {
				status = models.MonitorDown
			}
		case models.MonitorSSL:
			result, err := utils.CheckSSL(req.Target, timeout, thresholds)
			if err != nil {
				status = models.MonitorDown
			} else {
				sslWarning = result.Warning
				sslDaysRemaining = result.DaysRemaining
				sslTriggeredThreshold = result.TriggeredThreshold
			}
			if req.Monitoring.Advanced.DomainExpiry {
				result, domainErr := utils.CheckDomain(req.Target, string(req.Type), thresholds)
				if domainErr != nil {
					status = models.MonitorDown
				} else {
					domainWarning = result.Warning
					domainDaysRemaining = result.DaysRemaining
					domainTriggeredThreshold = result.TriggeredThreshold
				}
			}
		}

		responseTime := time.Since(start).Milliseconds()

		c.JSON(http.StatusOK, gin.H{
			"status":                   status,
			"statusCode":               statusCode,
			"responseTime":             responseTime,
			"sslWarning":               sslWarning,
			"sslDaysRemaining":         sslDaysRemaining,
			"sslTriggeredThreshold":    sslTriggeredThreshold,
			"domainWarning":            domainWarning,
			"domainDaysRemaining":      domainDaysRemaining,
			"domainTriggeredThreshold": domainTriggeredThreshold,
		})
	}
}

func GetMonitorLogs(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		monitorID, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid monitor id"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		service := monitorservice.NewService(repository.NewMongoMonitorRepository(db))
		logs, err := service.Logs(ctx, monitorID, 100)
		if err != nil {
			writeDomainError(c, err)
			return
		}

		c.JSON(http.StatusOK, logs)
	}
}

func GetMonitorUptime(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		monitorID, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid monitor id"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		service := monitorservice.NewService(repository.NewMongoMonitorRepository(db))
		uptime, err := service.Uptime(ctx, monitorID, time.Now().AddDate(0, 0, -90))
		if err != nil {
			writeDomainError(c, err)
			return
		}

		c.JSON(http.StatusOK, uptime)
	}
}

func DeleteMonitor(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		service := monitorservice.NewService(repository.NewMongoMonitorRepository(db))
		err = service.Delete(ctx, id)
		if err != nil {
			writeDomainError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "deleted"})
	}
}

func GetMonitorOutages(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		service := monitorservice.NewService(repository.NewMongoMonitorRepository(db))
		outages, err := service.Outages(ctx)
		if err != nil {
			writeDomainError(c, err)
			return
		}

		c.JSON(http.StatusOK, outages)
	}
}

func GetMonitorHistory(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		monitorID, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid monitor id"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		service := monitorservice.NewService(repository.NewMongoMonitorRepository(db))
		logs, err := service.History(ctx, monitorID, 100)
		if err != nil {
			writeDomainError(c, err)
			return
		}

		c.JSON(http.StatusOK, logs)
	}
}

var errMonitorNotFound = errorsNew("monitor not found")

func errorsWrap(err error, sentinel error) error {
	if err == nil {
		return sentinel
	}
	return fmt.Errorf("%w: %v", sentinel, err)
}

func errorsNew(msg string) error { return errors.New(msg) }
