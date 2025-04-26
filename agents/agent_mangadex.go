package agents

// FIXME: Implement this whole agent from the ground up with the new engine and using the helper functions
// 		base it on: https://github.com/manga-download/hakuneko/blob/master/src/web/mjs/connectors/MangaDex.mjs
import (
	"Luminary/engine"
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// MangaDex implements the Agent interface for MangaDex using the BaseAgent
type MangaDex struct {
	*BaseAgent
	serverNetwork []string
}

// NewMangaDex creates a new MangaDex agent
func NewMangaDex() *MangaDex {
	agent := &MangaDex{
		BaseAgent: NewBaseAgent(
			"mgd",
			"MangaDex",
			"World's largest manga community and scanlation site",
		),
		serverNetwork: []string{
			"https://uploads.mangadex.org/data/",
		},
	}

	// Configure site URLs
	agent.SetSiteURL("https://mangadex.org")
	agent.SetAPIURL("https://api.mangadex.org")

	// Configure API endpoints
	agent.configureAPIEndpoints()

	// Configure extractors
	agent.configureExtractors()

	// Configure pagination
	agent.configurePagination()

	return agent
}

// OnInitialize initializes the MangaDex agent
func (m *MangaDex) OnInitialize(ctx context.Context) error {
	// Add additional servers to the network
	m.serverNetwork = append(m.serverNetwork, "https://cache.ayaya.red/mdah/data/")
	m.Engine.Logger.Info("Added network seeds '[ %s ]' to %s", strings.Join(m.serverNetwork, ", "), m.Name())
	return nil
}

// configureAPIEndpoints sets up the API configuration
func (m *MangaDex) configureAPIEndpoints() {
	m.APIConfig = engine.APIConfig{
		BaseURL:      m.apiURL,
		RateLimitKey: "api.mangadex.org",
		RetryCount:   3,
		ThrottleTime: 2 * time.Second,
		DefaultHeaders: map[string]string{
			"User-Agent": "Luminary/1.0",
			"Referer":    "https://mangadex.org",
		},
		Endpoints: map[string]engine.APIEndpoint{
			// Manga details endpoint
			"manga": {
				Path:          "/manga/{id}",
				Method:        "GET",
				ResponseType:  &MangaDexMangaResponse{},
				PathFormatter: engine.DefaultPathFormatter,
			},

			// Chapter details endpoint
			"chapter": {
				Path:          "/chapter/{id}",
				Method:        "GET",
				ResponseType:  &MangaDexChapterResponse{},
				PathFormatter: engine.DefaultPathFormatter,
			},

			// Chapter pages endpoint
			"chapterPages": {
				Path:          "/at-home/server/{id}",
				Method:        "GET",
				ResponseType:  &MangaDexPagesResponse{},
				PathFormatter: engine.DefaultPathFormatter,
			},

			// Manga chapters endpoint
			"chapters": {
				Path:          "/manga/{id}/feed",
				Method:        "GET",
				ResponseType:  &MangaDexChapterListResponse{},
				PathFormatter: engine.DefaultPathFormatter,
				QueryFormatter: func(params interface{}) url.Values {
					queryParams := url.Values{}
					queryParams.Set("limit", "100")
					queryParams.Set("order[chapter]", "asc")
					// Add other parameters based on the input if needed
					return queryParams
				},
			},

			// Search endpoint
			"search": {
				Path:         "/manga",
				Method:       "GET",
				ResponseType: &MangaDexSearchResponse{},
				QueryFormatter: func(params interface{}) url.Values {
					queryParams := url.Values{}

					// Handle search options
					if opts, ok := params.(*engine.SearchOptions); ok {
						// Apply the title query
						if opts.Query != "" {
							queryParams.Set("title", opts.Query)
						}

						// Set limit
						if opts.Limit > 0 {
							queryParams.Set("limit", strconv.Itoa(opts.Limit))
						} else {
							queryParams.Set("limit", "10") // Default
						}

						// Apply sorting
						if opts.Sort != "" {
							switch strings.ToLower(opts.Sort) {
							case "relevance":
								queryParams.Set("order[relevance]", "desc")
							case "popularity":
								queryParams.Set("order[followedCount]", "desc")
							case "name":
								queryParams.Set("order[title]", "asc")
							}
						}

						// Apply filters
						if opts.Filters != nil {
							for field, value := range opts.Filters {
								switch field {
								case "author":
									queryParams.Add("authors[]", value)
								case "genre":
									queryParams.Add("includedTags[]", value)
								}
							}
						}
					}

					// Include all content ratings
					queryParams.Add("contentRating[]", "safe")
					queryParams.Add("contentRating[]", "suggestive")
					queryParams.Add("contentRating[]", "erotica")
					queryParams.Add("contentRating[]", "pornographic")

					return queryParams
				},
			},
		},
	}
}

// configureExtractors sets up the data extractors
func (m *MangaDex) configureExtractors() {
	// Manga extractor set
	m.ExtractorSets["manga"] = engine.ExtractorSet{
		Name:  "MangaDexManga",
		Model: &MangaInfo{},
		Extractors: []engine.Extractor{
			{
				Name:       "ID",
				SourcePath: []string{"data", "id"},
				TargetPath: "ID",
				Required:   true,
			},
			{
				Name:       "Title",
				SourcePath: []string{"data", "attributes", "title"},
				TargetPath: "Title",
				Transform: func(value interface{}) interface{} {
					if titleMap, ok := value.(map[string]string); ok {
						return m.extractBestTitle(titleMap)
					}
					return ""
				},
				Required: true,
			},
			{
				Name:       "Description",
				SourcePath: []string{"data", "attributes", "description"},
				TargetPath: "Description",
				Transform: func(value interface{}) interface{} {
					if descMap, ok := value.(map[string]string); ok {
						return m.extractBestDescription(descMap)
					}
					return ""
				},
			},
			{
				Name:       "AltTitles",
				SourcePath: []string{"data", "attributes", "altTitles"},
				TargetPath: "AltTitles",
				Transform: func(value interface{}) interface{} {
					if altTitles, ok := value.([]map[string]string); ok {
						titleMap := make(map[string]string)
						// Get the main title first
						mainTitle := m.extractBestTitle(titleMap)
						return m.extractAltTitles(titleMap, altTitles, mainTitle)
					}
					return []string{}
				},
			},
			{
				Name:       "Status",
				SourcePath: []string{"data", "attributes", "status"},
				TargetPath: "Status",
			},
			{
				Name:       "Tags",
				SourcePath: []string{"data", "attributes", "tags"},
				TargetPath: "Tags",
				Transform: func(value interface{}) interface{} {
					if tags, ok := value.([]struct {
						Attributes struct {
							Name map[string]string `json:"name"`
						} `json:"attributes"`
					}); ok {
						return m.extractTags(tags)
					}
					return []string{}
				},
			},
			{
				Name:       "Authors",
				SourcePath: []string{"data", "relationships"},
				TargetPath: "Authors",
				Transform: func(value interface{}) interface{} {
					if relationships, ok := value.([]struct {
						ID         string `json:"id"`
						Type       string `json:"type"`
						Attributes struct {
							Name string `json:"name"`
						} `json:"attributes"`
					}); ok {
						return m.extractAuthors(relationships)
					}
					return []string{}
				},
			},
		},
	}

	// Chapter info extractor set
	m.ExtractorSets["chapterInfo"] = engine.ExtractorSet{
		Name:  "MangaDexChapterInfo",
		Model: &ChapterInfo{},
		Extractors: []engine.Extractor{
			{
				Name:       "ID",
				SourcePath: []string{"id"},
				TargetPath: "ID",
				Required:   true,
			},
			{
				Name:       "Title",
				SourcePath: []string{"attributes", "title"},
				TargetPath: "Title",
				Transform: func(value interface{}) interface{} {
					title, _ := value.(string)
					if title == "" {
						// Try to build a title from chapter and volume
						chapterStr := ""
						volumeStr := ""

						// Get chapter number
						if ch, ok := m.getSourceValue([]string{"attributes", "chapter"}).(string); ok && ch != "" {
							chapterStr = ch
						}

						// Get volume number
						if vol, ok := m.getSourceValue([]string{"attributes", "volume"}).(string); ok && vol != "" {
							volumeStr = vol
						}

						if volumeStr != "" && chapterStr != "" {
							return fmt.Sprintf("Vol. %s Ch. %s", volumeStr, chapterStr)
						} else if chapterStr != "" {
							return fmt.Sprintf("Chapter %s", chapterStr)
						}
					}
					return title
				},
			},
			{
				Name:       "Number",
				SourcePath: []string{"attributes", "chapter"},
				TargetPath: "Number",
				Transform: func(value interface{}) interface{} {
					if chapterStr, ok := value.(string); ok && chapterStr != "" {
						if num, err := strconv.ParseFloat(chapterStr, 64); err == nil {
							return num
						}
					}
					return 0.0
				},
			},
			{
				Name:       "Date",
				SourcePath: []string{"attributes", "publishAt"},
				TargetPath: "Date",
				Transform: func(value interface{}) interface{} {
					if dateStr, ok := value.(string); ok && dateStr != "" {
						if date, err := time.Parse(time.RFC3339, dateStr); err == nil {
							return date
						}
					}
					return time.Now()
				},
			},
		},
	}

	// Chapter with pages extractor set
	m.ExtractorSets["chapter"] = engine.ExtractorSet{
		Name:  "MangaDexChapter",
		Model: &Chapter{},
		Extractors: []engine.Extractor{
			{
				Name:       "ChapterInfo",
				SourcePath: []string{"data"},
				TargetPath: "Info",
				Transform: func(value interface{}) interface{} {
					// Use the chapterInfo extractor to extract chapter metadata
					if data, ok := value.(map[string]interface{}); ok {
						result, err := m.Engine.Extractor.Extract(m.ExtractorSets["chapterInfo"], data)
						if err == nil {
							if chapterInfo, ok := result.(*ChapterInfo); ok {
								return *chapterInfo
							}
						}
					}
					return ChapterInfo{}
				},
				Required: true,
			},
			// Pages will be loaded separately using the chapterPages endpoint
		},
	}
}

// configurePagination sets up the pagination configuration
func (m *MangaDex) configurePagination() {
	m.PaginationConfig = engine.PaginationConfig{
		LimitParam:     "limit",
		OffsetParam:    "offset",
		TotalCountPath: []string{"total"},
		ItemsPath:      []string{"data"},
		DefaultLimit:   100,
		MaxLimit:       100,
	}
}

// getSourceValue is a helper to get a value from the source data
func (m *MangaDex) getSourceValue(path []string) interface{} {
	// This is a stub - in a real implementation, you'd need access to the source data
	return nil
}

// GetChapter overrides the BaseAgent's method to handle the special case of loading pages
func (m *MangaDex) GetChapter(ctx context.Context, chapterID string) (*Chapter, error) {
	// Initialize if needed
	if err := m.Initialize(ctx); err != nil {
		return nil, err
	}

	// Try cache first
	cacheKey := fmt.Sprintf("chapter:%s:%s", m.id, chapterID)
	var cachedChapter Chapter
	if m.Engine.Cache.Get(cacheKey, &cachedChapter) {
		m.Engine.Logger.Debug("[%s] Using cached chapter info for: %s", m.id, chapterID)
		return &cachedChapter, nil
	}

	// Log chapter retrieval
	m.Engine.Logger.Info("[%s] Fetching chapter details for: %s", m.id, chapterID)

	// Fetch chapter details using API service
	response, err := m.Engine.API.FetchFromAPI(
		ctx,
		m.APIConfig,
		"chapter",
		nil,
		chapterID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch chapter: %w", err)
	}

	// Create a new chapter object
	chapter := &Chapter{
		Info: ChapterInfo{
			ID: chapterID,
		},
	}

	// Process the response directly instead of using the extractor
	chapterResp, ok := response.(*MangaDexChapterResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected response type: %T", response)
	}

	// Extract chapter information
	chapter.Info.ID = chapterID
	chapter.Info.Title = chapterResp.Data.Attributes.Title

	// Convert chapter number if present
	if chapterResp.Data.Attributes.Chapter != "" {
		if num, err := strconv.ParseFloat(chapterResp.Data.Attributes.Chapter, 64); err == nil {
			chapter.Info.Number = num
		}
	}

	// Set publication date
	chapter.Info.Date = chapterResp.Data.Attributes.PublishAt

	// Extract manga ID from relationships
	for _, rel := range chapterResp.Data.Relationships {
		if rel.Type == "manga" {
			chapter.MangaID = rel.ID
			break
		}
	}

	// Now fetch the pages using the chapterPages endpoint
	pagesResponse, err := m.Engine.API.FetchFromAPI(
		ctx,
		m.APIConfig,
		"chapterPages",
		nil,
		chapterID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch chapter pages: %w", err)
	}

	// Extract pages from the response
	pages, err := m.extractPages(pagesResponse, chapterID)
	if err != nil {
		return nil, fmt.Errorf("failed to extract chapter pages: %w", err)
	}

	// Add pages to the chapter
	chapter.Pages = pages

	// Cache the result
	if err := m.Engine.Cache.Set(cacheKey, chapter); err != nil {
		m.Engine.Logger.Warn("[%s] Failed to cache chapter info: %v", m.id, err)
	}

	return chapter, nil
}

// extractPages extracts pages from the pages response
func (m *MangaDex) extractPages(response interface{}, chapterID string) ([]Page, error) {
	// Try to cast to the expected response type
	pagesResp, ok := response.(*MangaDexPagesResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected response type: %T", response)
	}

	// Extract pages
	pages := make([]Page, len(pagesResp.Chapter.Data))
	for i, file := range pagesResp.Chapter.Data {
		// Try using the first server in our network
		formattedUrl := fmt.Sprintf("%s%s/%s",
			m.serverNetwork[0],
			pagesResp.Chapter.Hash,
			file)

		pages[i] = Page{
			Index:    i,
			URL:      formattedUrl,
			Filename: file,
		}
	}

	return pages, nil
}

// extractMangaIDFromChapterResponse extracts the manga ID from a chapter response
func (m *MangaDex) extractMangaIDFromChapterResponse(response interface{}) (string, error) {
	// Try to cast to the expected response type
	chapterResp, ok := response.(*MangaDexChapterResponse)
	if !ok {
		return "", fmt.Errorf("unexpected response type: %T", response)
	}

	// Find the manga ID in relationships
	for _, rel := range chapterResp.Data.Relationships {
		if rel.Type == "manga" {
			return rel.ID, nil
		}
	}

	return "", fmt.Errorf("manga ID not found in chapter relationships")
}

// extractBestTitle extracts the best title from a title map
func (m *MangaDex) extractBestTitle(titleMap map[string]string) string {
	// Prefer English
	if enTitle, ok := titleMap["en"]; ok && enTitle != "" {
		return enTitle
	}

	// Then try Japanese
	if jaTitle, ok := titleMap["ja"]; ok && jaTitle != "" {
		return jaTitle
	}

	// Finally, take any title
	for _, title := range titleMap {
		if title != "" {
			return title
		}
	}

	return ""
}

// extractBestDescription extracts the best description from a description map
func (m *MangaDex) extractBestDescription(descMap map[string]string) string {
	// Prefer English
	if enDesc, ok := descMap["en"]; ok && enDesc != "" {
		return enDesc
	}

	// Then try Japanese
	if jaDesc, ok := descMap["ja"]; ok && jaDesc != "" {
		return jaDesc
	}

	// Finally, take any description
	for _, desc := range descMap {
		if desc != "" {
			return desc
		}
	}

	return ""
}

// extractAltTitles extracts alternative titles
func (m *MangaDex) extractAltTitles(titleMap map[string]string, altTitlesList []map[string]string, mainTitle string) []string {
	var altTitles []string

	// Process all language variants from the main title
	for _, t := range titleMap {
		if t != mainTitle && t != "" { // Don't include the main title or empty strings
			altTitles = append(altTitles, t)
		}
	}

	// Process all entries in altTitles array
	for _, titleMap := range altTitlesList {
		for _, t := range titleMap {
			if t != mainTitle && t != "" { // Don't include the main title or empty strings
				// Check if the title is already in our list
				duplicate := false
				for _, existingTitle := range altTitles {
					if existingTitle == t {
						duplicate = true
						break
					}
				}

				if !duplicate {
					altTitles = append(altTitles, t)
				}
			}
		}
	}

	return altTitles
}

// extractTags extracts tags from the manga
func (m *MangaDex) extractTags(tagsList []struct {
	Attributes struct {
		Name map[string]string `json:"name"`
	} `json:"attributes"`
}) []string {
	tags := make([]string, 0, len(tagsList))
	for _, tag := range tagsList {
		// Prefer English tag name
		if name, ok := tag.Attributes.Name["en"]; ok && name != "" {
			tags = append(tags, name)
			continue
		}

		// Fall back to any tag name
		for _, name := range tag.Attributes.Name {
			if name != "" {
				tags = append(tags, name)
				break
			}
		}
	}
	return tags
}

// extractAuthors extracts authors from relationships
func (m *MangaDex) extractAuthors(relationships []struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Name string `json:"name"`
	} `json:"attributes"`
}) []string {
	var authors []string
	for _, rel := range relationships {
		if (rel.Type == "author" || rel.Type == "artist") && rel.Attributes.Name != "" {
			authors = append(authors, rel.Attributes.Name)
		}
	}
	return authors
}

// MangaDexMangaResponse represents the API response for manga details
type MangaDexMangaResponse struct {
	Data struct {
		ID         string `json:"id"`
		Attributes struct {
			Title       map[string]string   `json:"title"`
			Description map[string]string   `json:"description"`
			Status      string              `json:"status"`
			AltTitles   []map[string]string `json:"altTitles"`
			Tags        []struct {
				Attributes struct {
					Name map[string]string `json:"name"`
				} `json:"attributes"`
			} `json:"tags"`
		} `json:"attributes"`
		Relationships []struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Name string `json:"name"`
			} `json:"attributes"`
		} `json:"relationships"`
	} `json:"data"`
}

// MangaDexChapterResponse represents the API response for chapter details
type MangaDexChapterResponse struct {
	Data struct {
		ID         string `json:"id"`
		Attributes struct {
			Title     string    `json:"title"`
			Chapter   string    `json:"chapter"`
			Volume    string    `json:"volume"`
			PublishAt time.Time `json:"publishAt"`
		} `json:"attributes"`
		Relationships []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"relationships"`
	} `json:"data"`
}

// MangaDexPagesResponse represents the API response for chapter pages
type MangaDexPagesResponse struct {
	BaseURL string `json:"baseUrl"`
	Chapter struct {
		Hash string   `json:"hash"`
		Data []string `json:"data"`
	} `json:"chapter"`
}

// MangaDexChapterListResponse represents the API response for manga chapters
type MangaDexChapterListResponse struct {
	Data []struct {
		ID         string `json:"id"`
		Attributes struct {
			Title     string    `json:"title"`
			Chapter   string    `json:"chapter"`
			Volume    string    `json:"volume"`
			PublishAt time.Time `json:"publishAt"`
		} `json:"attributes"`
	} `json:"data"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// MangaDexSearchResponse represents the API response for manga search
type MangaDexSearchResponse struct {
	Data []struct {
		ID         string `json:"id"`
		Attributes struct {
			Title       map[string]string   `json:"title"`
			AltTitles   []map[string]string `json:"altTitles"`
			Description map[string]string   `json:"description"`
			Tags        []struct {
				Attributes struct {
					Name map[string]string `json:"name"`
				} `json:"attributes"`
			} `json:"tags"`
			Status string `json:"status"`
		} `json:"attributes"`
		Relationships []struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				Name string `json:"name"`
			} `json:"attributes"`
		} `json:"relationships"`
	} `json:"data"`
}

func init() {
	Register(NewMangaDex())
}
