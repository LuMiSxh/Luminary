package providers

import (
	"Luminary/pkg/engine"
	"Luminary/pkg/provider/base"
	"Luminary/pkg/provider/registry"
	"time"
)

func init() {
	registry.Register(NewKissMangaProvider)
}

// NewKissMangaProvider creates a new KissManga provider using the simplified framework
func NewKissMangaProvider(e *engine.Engine) engine.Provider {
	return base.New(e, base.Config{
		ID:          "kmg",
		Name:        "KissManga",
		Description: "Read manga online for free at KissManga with daily updates",
		SiteURL:     "https://kissmanga.in",
		Type:        base.TypeMadara,

		Madara: &base.MadaraConfig{
			Selectors: map[string]string{
				"search":      "div.post-title h3 a, div.post-title h5 a",
				"title":       "h1.post-title, .post-title-font",
				"description": ".description-summary, .summary__content",
				"chapters":    "li.wp-manga-chapter > a, .chapter-link",
				"pages":       "div.page-break img, .reading-content img",
				"author":      ".author-content a, .manga-authors a",
				"status":      ".post-status .summary-content",
				"genres":      ".genres-content a",
			},
			AjaxSearch:       true,
			CustomLoadAction: "madara_load_more",
		},

		// Web configuration for search
		Web: &base.WebConfig{
			SearchPath: "/?s={query}&post_type=wp-manga",
		},

		Headers: map[string]string{
			"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
			"Accept-Language": "en-US,en;q=0.9",
			"Cache-Control":   "no-cache",
			"Referer":         "https://kissmanga.in/",
		},

		RateLimit: 2 * time.Second,
	}).Build()
}
