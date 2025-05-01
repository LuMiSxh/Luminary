package providers

import (
	"Luminary/pkg/engine"
	"Luminary/pkg/provider"
	"Luminary/pkg/provider/madara"
)

// NewMadaraProvider creates a new KissManga agent using the Madara framework
func NewMadaraProvider(e *engine.Engine) provider.Provider {
	// Configure KissManga-specific settings using the Madara framework
	config := madara.Config{
		ID:              "kmg",
		Name:            "KissManga",
		SiteURL:         "https://kissmanga.in",
		Description:     "Read manga online for free at KissManga with daily updates",
		MangaSelector:   "div.post-title h3 a, div.post-title h5 a",
		ChapterSelector: "li.wp-manga-chapter > a, .chapter-link, div.listing-chapters_wrap a, .wp-manga-chapter a",
		PageSelector:    "div.page-break source, div.page-break img, .reading-content img",
		Headers: map[string]string{
			"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.102 Safari/537.36",
			"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9",
			"Accept-Language": "en-US,en;q=0.9",
			"Cache-Control":   "no-cache",
			"Connection":      "keep-alive",
			"Pragma":          "no-cache",
			"Referer":         "https://kissmanga.in/",
		},
	}

	// Create the Madara agent with our custom configuration
	return madara.NewProvider(e, config)
}
