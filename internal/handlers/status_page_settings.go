package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/fresp/StatusForge/internal/models"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type statusPageSettingsPatchRequest struct {
	Head *struct {
		Title       *string            `json:"title"`
		Description *string            `json:"description"`
		Keywords    *string            `json:"keywords"`
		FaviconURL  *string            `json:"faviconUrl"`
		MetaTags    *map[string]string `json:"metaTags"`
	} `json:"head"`
	Branding *struct {
		SiteName *string `json:"siteName"`
		LogoURL  *string `json:"logoUrl"`
	} `json:"branding"`
	Theme *struct {
		PrimaryColor    *string `json:"primaryColor"`
		BackgroundColor *string `json:"backgroundColor"`
		TextColor       *string `json:"textColor"`
	} `json:"theme"`
	Layout *struct {
		Variant *string `json:"variant"`
	} `json:"layout"`
	Footer *struct {
		Text          *string `json:"text"`
		ShowPoweredBy *bool   `json:"showPoweredBy"`
	} `json:"footer"`
	CustomCSS *string `json:"customCss"`
}

func settingsCollection(db *mongo.Database) *mongo.Collection {
	return db.Collection("settings")
}

func fetchOrCreateStatusPageSettings(ctx context.Context, db *mongo.Database) (models.StatusPageSettings, error) {
	var settings models.StatusPageSettings
	err := settingsCollection(db).FindOne(ctx, bson.M{"key": models.StatusPageSettingsKey}).Decode(&settings)
	if err == nil {
		return settings, nil
	}
	if err != mongo.ErrNoDocuments {
		return models.StatusPageSettings{}, err
	}

	defaultSettings := models.DefaultStatusPageSettings()
	if _, insertErr := settingsCollection(db).InsertOne(ctx, defaultSettings); insertErr != nil {
		return models.StatusPageSettings{}, insertErr
	}
	return defaultSettings, nil
}

func GetPublicStatusPageSettings(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		settings, err := fetchOrCreateStatusPageSettings(ctx, db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, settings)
	}
}

func GetAdminStatusPageSettings(db *mongo.Database) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		settings, err := fetchOrCreateStatusPageSettings(ctx, db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, settings)
	}
}

func UpdateStatusPageSettings(db *mongo.Database, hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req statusPageSettingsPatchRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		current, err := fetchOrCreateStatusPageSettings(ctx, db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		set := bson.M{"updatedAt": time.Now()}

		if req.Head != nil {
			if req.Head.Title != nil {
				set["head.title"] = *req.Head.Title
			}
			if req.Head.Description != nil {
				set["head.description"] = *req.Head.Description
			}
			if req.Head.Keywords != nil {
				set["head.keywords"] = *req.Head.Keywords
			}
			if req.Head.FaviconURL != nil {
				set["head.faviconUrl"] = *req.Head.FaviconURL
			}
			if req.Head.MetaTags != nil {
				set["head.metaTags"] = *req.Head.MetaTags
			}
		}

		if req.Branding != nil {
			if req.Branding.SiteName != nil {
				set["branding.siteName"] = *req.Branding.SiteName
			}
			if req.Branding.LogoURL != nil {
				set["branding.logoUrl"] = *req.Branding.LogoURL
			}
		}

		if req.Theme != nil {
			if req.Theme.PrimaryColor != nil {
				set["theme.primaryColor"] = *req.Theme.PrimaryColor
			}
			if req.Theme.BackgroundColor != nil {
				set["theme.backgroundColor"] = *req.Theme.BackgroundColor
			}
			if req.Theme.TextColor != nil {
				set["theme.textColor"] = *req.Theme.TextColor
			}
		}

		if req.Layout != nil && req.Layout.Variant != nil {
			if *req.Layout.Variant != "classic" && *req.Layout.Variant != "compact" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "layout.variant must be one of: classic, compact"})
				return
			}
			set["layout.variant"] = *req.Layout.Variant
		}

		if req.Footer != nil {
			if req.Footer.Text != nil {
				set["footer.text"] = *req.Footer.Text
			}
			if req.Footer.ShowPoweredBy != nil {
				set["footer.showPoweredBy"] = *req.Footer.ShowPoweredBy
			}
		}

		if req.CustomCSS != nil {
			set["customCss"] = *req.CustomCSS
		}

		if len(set) == 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no updatable fields provided"})
			return
		}

		var updated models.StatusPageSettings
		err = settingsCollection(db).FindOneAndUpdate(
			ctx,
			bson.M{"key": models.StatusPageSettingsKey},
			bson.M{"$set": set, "$setOnInsert": bson.M{"createdAt": current.CreatedAt, "key": models.StatusPageSettingsKey}},
			options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After),
		).Decode(&updated)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		BroadcastEvent(hub, "status_page_settings_updated", updated)
		c.JSON(http.StatusOK, updated)
	}
}
