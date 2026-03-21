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
	SiteName string `bson:"siteName" json:"siteName"`
	LogoURL  string `bson:"logoUrl" json:"logoUrl"`
}

type StatusPageThemeSettings struct {
	PrimaryColor    string `bson:"primaryColor" json:"primaryColor"`
	BackgroundColor string `bson:"backgroundColor" json:"backgroundColor"`
	TextColor       string `bson:"textColor" json:"textColor"`
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
			Title:       "Status Platform",
			Description: "Live system status and incident updates.",
			Keywords:    "status, uptime, incidents, maintenance",
			FaviconURL:  "/vite.svg",
			MetaTags:    map[string]string{},
		},
		Branding: StatusPageBrandingSettings{
			SiteName: "System Status",
			LogoURL:  "",
		},
		Theme: StatusPageThemeSettings{
			PrimaryColor:    "#16a34a",
			BackgroundColor: "#f9fafb",
			TextColor:       "#111827",
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
