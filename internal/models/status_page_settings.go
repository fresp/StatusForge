package models

import "time"

const (
	StatusPageSettingsKey = "status_page"
)

type StatusPageHeadSettings struct {
	Title       string            `bson:"title" json:"title"`
	Description string            `bson:"description" json:"description"`
	Keywords    string            `bson:"keywords" json:"keywords"`
	FaviconURL  string            `bson:"faviconUrl" json:"faviconUrl"`
	MetaTags    map[string]string `bson:"metaTags,omitempty" json:"metaTags,omitempty"`
}

type StatusPageBrandingSettings struct {
	SiteName           string `bson:"siteName" json:"siteName"`
	LogoURL            string `bson:"logoUrl" json:"logoUrl"`
	BackgroundImageURL string `bson:"backgroundImageUrl" json:"backgroundImageUrl"`
	HeroImageURL       string `bson:"heroImageUrl" json:"heroImageUrl"`
}

type StatusPageThemePalette struct {
	PrimaryColor    string `bson:"primaryColor" json:"primaryColor"`
	BackgroundColor string `bson:"backgroundColor" json:"backgroundColor"`
	TextColor       string `bson:"textColor" json:"textColor"`
	AccentColor     string `bson:"accentColor" json:"accentColor"`
}

type StatusPageThemeTypography struct {
	FontFamily string `bson:"fontFamily" json:"fontFamily"`
	FontScale  string `bson:"fontScale" json:"fontScale"`
}

type StatusPageThemeSettings struct {
	Preset     string                    `bson:"preset" json:"preset"`
	Mode       string                    `bson:"mode" json:"mode"`
	Light      StatusPageThemePalette    `bson:"light" json:"light"`
	Dark       StatusPageThemePalette    `bson:"dark" json:"dark"`
	Typography StatusPageThemeTypography `bson:"typography" json:"typography"`
}

type StatusPageLayoutSettings struct {
	Variant string `bson:"variant" json:"variant"`
}

type StatusPageFooterSettings struct {
	Text          string `bson:"text" json:"text"`
	ShowPoweredBy bool   `bson:"showPoweredBy" json:"showPoweredBy"`
}

type StatusPageSettings struct {
	Key       string                     `bson:"key" json:"-"`
	Head      StatusPageHeadSettings     `bson:"head" json:"head"`
	Branding  StatusPageBrandingSettings `bson:"branding" json:"branding"`
	Theme     StatusPageThemeSettings    `bson:"theme" json:"theme"`
	Layout    StatusPageLayoutSettings   `bson:"layout" json:"layout"`
	Footer    StatusPageFooterSettings   `bson:"footer" json:"footer"`
	CustomCSS string                     `bson:"customCss" json:"customCss"`
	UpdatedAt time.Time                  `bson:"updatedAt" json:"updatedAt"`
	CreatedAt time.Time                  `bson:"createdAt" json:"createdAt"`
}

func DefaultStatusPageSettings() StatusPageSettings {
	now := time.Now()
	return StatusPageSettings{
		Key: StatusPageSettingsKey,
		Head: StatusPageHeadSettings{
			Title:       "Statora",
			Description: "Live system status and incident updates.",
			Keywords:    "status, uptime, incidents, maintenance",
			FaviconURL:  "/vite.svg",
			MetaTags:    map[string]string{},
		},
		Branding: StatusPageBrandingSettings{
			SiteName:           "Statora",
			LogoURL:            "",
			BackgroundImageURL: "",
			HeroImageURL:       "",
		},
		Theme: StatusPageThemeSettings{
			Preset: "default",
			Mode:   "system",
			Light: StatusPageThemePalette{
				PrimaryColor:    "#16a34a",
				BackgroundColor: "#f9fafb",
				TextColor:       "#111827",
				AccentColor:     "#0ea5e9",
			},
			Dark: StatusPageThemePalette{
				PrimaryColor:    "#22c55e",
				BackgroundColor: "#0b1220",
				TextColor:       "#e5e7eb",
				AccentColor:     "#38bdf8",
			},
			Typography: StatusPageThemeTypography{
				FontFamily: "Inter, system-ui, -apple-system, BlinkMacSystemFont, \"Segoe UI\", sans-serif",
				FontScale:  "md",
			},
		},
		Layout: StatusPageLayoutSettings{
			Variant: "classic",
		},
		Footer: StatusPageFooterSettings{
			Text:          "",
			ShowPoweredBy: true,
		},
		CustomCSS: "",
		UpdatedAt: now,
		CreatedAt: now,
	}
}
