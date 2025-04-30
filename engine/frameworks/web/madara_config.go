package web

// MadaraConfig holds configuration for Madara-based sites
type MadaraConfig struct {
	ID          string            // Short identifier
	Name        string            // Display name
	SiteURL     string            // Base URL for the site
	Description string            // Site description
	Headers     map[string]string // Additional HTTP headers

	// Selectors for different content types
	MangaSelector   string // Selector for manga list items
	ChapterSelector string // Selector for chapter list items
	PageSelector    string // Selector for page images

	// Custom options
	UseLegacyAjax    bool   // Use old-style AJAX requests
	CustomLoadAction string // Custom AJAX action for loading more content
}

// DefaultMadaraConfig returns a default configuration for Madara sites
func DefaultMadaraConfig(id, name, siteURL, description string) MadaraConfig {
	return MadaraConfig{
		ID:              id,
		Name:            name,
		SiteURL:         siteURL,
		Description:     description,
		MangaSelector:   "div.post-title h3 a, div.post-title h5 a, div.page-item-detail.manga div.post-title h3 a",
		ChapterSelector: "li.wp-manga-chapter > a",
		PageSelector:    "div.page-break source, div.page-break img, .reading-content img",
		UseLegacyAjax:   false,
		Headers: map[string]string{
			"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
			"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9",
			"Accept-Language": "en-US,en;q=0.9",
			"Cache-Control":   "max-age=0",
			"Connection":      "keep-alive",
		},
	}
}
