package mangadex

import (
	"Luminary/engine"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// FIXME: Try re-implementing this from scratch with claude
// 	use this as template: https://github.com/manga-download/hakuneko/blob/master/src/web/mjs/connectors/MangaDex.mjs
import (
	"Luminary/engine"
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// MangaDex implements the engine.Agent interface
type MangaDex struct {
	engine        *engine.Engine
	id            string
	name          string
	description   string
	siteURL       string
	apiConfig     engine.APIConfig
	extractorSets map[string]engine.ExtractorSet
	serverNetwork []string
}

// NewMangaDex creates a new MangaDex agent
func NewMangaDex(e *engine.Engine) engine.Agent {
	agent := &MangaDex{
		engine:        e,
		id:            "mgd",
		name:          "MangaDex",
		description:   "World's largest manga community and scanlation site",
		siteURL:       "https://mangadex.org",
		extractorSets: make(map[string]engine.ExtractorSet),
		serverNetwork: []string{
			"https://uploads.mangadex.org/data/",
		},
	}

	// Configure API endpoints
	agent.configureAPIEndpoints()

	// Configure extractors
	agent.configureExtractors()

	return agent
}

// ID returns the agent's identifier
func (m *MangaDex) ID() string {
	return m.id
}

// Name returns the agent's display name
func (m *MangaDex) Name() string {
	return m.name
}

// Description returns the agent's description
func (m *MangaDex) Description() string {
	return m.description
}

// SiteURL returns the agent's website URL
func (m *MangaDex) SiteURL() string {
	return m.siteURL
}

// Initialize initializes the MangaDex agent
func (m *MangaDex) Initialize(ctx context.Context) error {
	return engine.ExecuteInitialize(ctx, m.engine, m.id, m.name, m.onInitialize)
}

// onInitialize performs MangaDex-specific initialization
func (m *MangaDex) onInitialize(ctx context.Context) error {
	// Add additional servers to the network
	m.serverNetwork = append(m.serverNetwork, "https://cache.ayaya.red/mdah/data/")
	m.engine.Logger.Info("Added network seeds '[ %s ]' to %s", strings.Join(m.serverNetwork, ", "), m.Name())
	return nil
}

// configureAPIEndpoints sets up the API configuration
func (m *MangaDex) configureAPIEndpoints() {
	m.apiConfig = engine.APIConfig{
		BaseURL:      "https://api.mangadex.org",
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
				ResponseType:  &MangaResponse{},
				PathFormatter: engine.DefaultPathFormatter,
			},

			// Chapter details endpoint
			"chapter": {
				Path:          "/chapter/{id}",
				Method:        "GET",
				ResponseType:  &ChapterResponse{},
				PathFormatter: engine.DefaultPathFormatter,
			},

			// Chapter pages endpoint
			"chapterPages": {
				Path:          "/at-home/server/{id}",
				Method:        "GET",
				ResponseType:  &PagesResponse{},
				PathFormatter: engine.DefaultPathFormatter,
			},

			// Manga chapters endpoint
			"chapters": {
				Path:          "/manga/{id}/feed",
				Method:        "GET",
				ResponseType:  &ChapterListResponse{},
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
				ResponseType: &SearchResponse{},
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
	m.extractorSets["manga"] = engine.ExtractorSet{
		Name:  "MangaDexManga",
		Model: &engine.MangaInfo{},
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
			// Other extractors...
		},
	}

	// Other extractor sets...
}

// Search implements the engine.Agent interface for searching
func (m *MangaDex) Search(ctx context.Context, query string, options engine.SearchOptions) ([]engine.Manga, error) {
	// Use the engine helper with appropriate configuration
	return engine.ExecuteSearch(
		ctx,
		m.engine,
		m.id,
		query,
		options,
		m.apiConfig,
		engine.PaginationConfig{
			LimitParam:     "limit",
			OffsetParam:    "offset",
			TotalCountPath: []string{"total"},
			ItemsPath:      []string{"data"},
			DefaultLimit:   100,
			MaxLimit:       100,
		},
		m.extractorSets["manga"],
	)
}

// GetManga implements the engine.Agent interface for retrieving manga details
func (m *MangaDex) GetManga(ctx context.Context, id string) (*engine.MangaInfo, error) {
	// Use the engine helper with appropriate configuration
	return engine.ExecuteGetManga(
		ctx,
		m.engine,
		m.id,
		id,
		m.apiConfig,
		m.extractorSets["manga"],
		func(ctx context.Context, mangaID string) ([]engine.ChapterInfo, error) {
			return m.fetchChaptersForManga(ctx, mangaID)
		},
	)
}

// GetChapter implements the engine.Agent interface for retrieving chapter details
func (m *MangaDex) GetChapter(ctx context.Context, chapterID string) (*engine.Chapter, error) {
	// Use the engine helper with customized processing function
	return engine.ExecuteGetChapter(
		ctx,
		m.engine,
		m.id,
		chapterID,
		m.apiConfig,
		m.extractorSets["chapter"],
		m.processChapterResponse,
	)
}

// TryGetMangaForChapter attempts to get manga info for a chapter
func (m *MangaDex) TryGetMangaForChapter(ctx context.Context, chapterID string) (*engine.Manga, error) {
	// Fetch chapter details first to get manga ID
	chapter, err := m.GetChapter(ctx, chapterID)
	if err != nil {
		return nil, err
	}

	// If manga ID is available in chapter
	if chapter.MangaID != "" {
		// Get manga details
		mangaInfo, err := m.GetManga(ctx, chapter.MangaID)
		if err != nil {
			return nil, err
		}
		return &mangaInfo.Manga, nil
	}

	return nil, fmt.Errorf("couldn't determine manga for chapter %s", chapterID)
}

// DownloadChapter implements the engine.Agent interface for downloading a chapter
func (m *MangaDex) DownloadChapter(ctx context.Context, chapterID, destDir string) error {
	// Use the engine helper for downloading
	return engine.ExecuteDownloadChapter(
		ctx,
		m.engine,
		m.id,
		m.name,
		chapterID,
		destDir,
		m.GetChapter,
		m.TryGetMangaForChapter,
	)
}

// processChapterResponse is a custom handler for chapter API responses
func (m *MangaDex) processChapterResponse(response interface{}, chapterID string) (*engine.Chapter, error) {
	// Cast to expected response type
	chapterResp, ok := response.(*ChapterResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected response type: %T", response)
	}

	// Create a new chapter object
	chapter := &engine.Chapter{
		Info: engine.ChapterInfo{
			ID: chapterID,
		},
	}

	// Extract chapter information
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

	// Fetch pages using a separate API call
	pagesResponse, err := m.engine.API.FetchFromAPI(
		context.Background(),
		m.apiConfig,
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

	return chapter, nil
}

// extractPages extracts pages from the pages response
func (m *MangaDex) extractPages(response interface{}, chapterID string) ([]engine.Page, error) {
	// Try to cast to the expected response type
	pagesResp, ok := response.(*PagesResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected response type: %T", response)
	}

	// Extract pages
	pages := make([]engine.Page, len(pagesResp.Chapter.Data))
	for i, file := range pagesResp.Chapter.Data {
		// Try using the first server in our network
		formattedUrl := fmt.Sprintf("%s%s/%s",
			m.serverNetwork[0],
			pagesResp.Chapter.Hash,
			file)

		pages[i] = engine.Page{
			Index:    i,
			URL:      formattedUrl,
			Filename: file,
		}
	}

	return pages, nil
}

// fetchChaptersForManga fetches all chapters for a manga
func (m *MangaDex) fetchChaptersForManga(ctx context.Context, mangaID string) ([]engine.ChapterInfo, error) {
	// Implementation omitted for brevity
	return []engine.ChapterInfo{}, nil
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

// Response types

type MangaResponse struct {
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

type ChapterResponse struct {
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

type PagesResponse struct {
	BaseURL string `json:"baseUrl"`
	Chapter struct {
		Hash string   `json:"hash"`
		Data []string `json:"data"`
	} `json:"chapter"`
}

type ChapterListResponse struct {
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

type SearchResponse struct {
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

// NewAgent is the exported constructor for use during application initialization
func NewAgent(e *engine.Engine) engine.Agent {
	return NewMangaDex(e)
}
