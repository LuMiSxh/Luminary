package providers

import (
	"Luminary/pkg/engine"
	"Luminary/pkg/engine/core"
	"Luminary/pkg/engine/network"
	"Luminary/pkg/engine/parser"
	"Luminary/pkg/engine/search"
	"Luminary/pkg/provider"
	"Luminary/pkg/provider/api"
	"Luminary/pkg/provider/common"
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type MgdMangaResp struct {
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
			LastChapter string `json:"lastChapter"`
			LastVolume  string `json:"lastVolume"`
			UpdatedAt   string `json:"updatedAt"` // Last updated timestamp
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

type MgdChapterResp struct {
	Data struct {
		ID         string `json:"id"`
		Attributes struct {
			Title              string    `json:"title"`
			Chapter            string    `json:"chapter"`
			Volume             string    `json:"volume"`
			PublishAt          time.Time `json:"publishAt"`
			TranslatedLanguage string    `json:"translatedLanguage"` // Language code (e.g., "en", "ja")
		} `json:"attributes"`
		Relationships []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"relationships"`
	} `json:"data"`
}

type MgdPagesResp struct {
	BaseURL string `json:"baseUrl"`
	Chapter struct {
		Hash string   `json:"hash"`
		Data []string `json:"data"`
	} `json:"chapter"`
}

type MgdChapterListResp struct {
	Data []struct {
		ID         string `json:"id"`
		Attributes struct {
			Title              string    `json:"title"`
			Chapter            string    `json:"chapter"`
			Volume             string    `json:"volume"`
			PublishAt          time.Time `json:"publishAt"`
			TranslatedLanguage string    `json:"translatedLanguage"` // Language code
		} `json:"attributes"`
	} `json:"data"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type MgdSearchResp struct {
	Result   string `json:"result"`
	Response string `json:"response"`
	Data     []struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
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
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

// MangaDexState holds MangaDex-specific data
type MangaDexState struct {
	ServerNetwork []string
}

// MangadexProvider is the main struct for the MangaDex provider
type MangadexProvider struct {
	*api.Provider
	state  *MangaDexState
	engine *engine.Engine
}

// languageCodeToName maps language codes to readable names
var languageCodeToName = map[string]string{
	"en": "English",
	"ja": "Japanese",
	"es": "Spanish",
	"fr": "French",
	"de": "German",
	"pt": "Portuguese",
	"ru": "Russian",
	"ko": "Korean",
	"zh": "Chinese",
	"it": "Italian",
	"ar": "Arabic",
	"tr": "Turkish",
	"th": "Thai",
	"vi": "Vietnamese",
	"id": "Indonesian",
	"pl": "Polish",
	"nl": "Dutch",
	"sv": "Swedish",
	"da": "Danish",
	"no": "Norwegian",
	"fi": "Finnish",
	"hu": "Hungarian",
	"cs": "Czech",
	"sk": "Slovak",
	"bg": "Bulgarian",
	"hr": "Croatian",
	"sr": "Serbian",
	"sl": "Slovenian",
	"et": "Estonian",
	"lv": "Latvian",
	"lt": "Lithuanian",
	"ro": "Romanian",
	"el": "Greek",
	"he": "Hebrew",
	"fa": "Persian",
	"hi": "Hindi",
	"bn": "Bengali",
	"ta": "Tamil",
	"te": "Telugu",
	"ml": "Malayalam",
	"kn": "Kannada",
	"gu": "Gujarati",
	"pa": "Punjabi",
	"ur": "Urdu",
}

// convertLanguageCode converts a language code to a standardized format
func convertLanguageCode(code string) *string {
	if code == "" {
		return nil
	}

	return &code
}

// parseNullableTime safely parses a time field, returning nil if empty or invalid
func parseNullableTime(timeStr string) *time.Time {
	if timeStr == "" {
		return nil
	}

	// Try parsing RFC3339 format first (common for APIs)
	if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
		return &t
	}

	// Try other common formats
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return &t
		}
	}

	return nil
}

// Initialize overrides the base provider's Initialize method
func (m *MangadexProvider) Initialize(ctx context.Context) error {
	return common.ExecuteInitialize(ctx, m.engine, m.ID(), m.Name(), func(ctx context.Context) error {
		// Add our custom network seed
		m.state.ServerNetwork = append(m.state.ServerNetwork, "https://cache.ayaya.red/mdah/data/")
		m.engine.Logger.Info("Added network seeds '[ %s ]' to %s", strings.Join(m.state.ServerNetwork, ", "), m.Name())
		return nil
	})
}

// NewMangadexProvider creates a new MangaDex provider
func NewMangadexProvider(e *engine.Engine) provider.Provider {
	// Create state with initial server network
	state := &MangaDexState{
		ServerNetwork: []string{
			"https://uploads.MangaDex.org/data/",
		},
	}

	// Create API provider configuration
	config := api.Config{
		// Basic identity
		ID:          "mgd",
		Name:        "MangaDex",
		Description: "World's largest manga community and scanlation site",
		SiteURL:     "https://mangadex.org",

		// API configuration
		BaseURL:      "https://api.mangadex.org",
		RateLimitKey: "api.mangadex.org",
		RetryCount:   3,
		ThrottleTime: 2 * time.Second,

		DefaultHeaders: map[string]string{
			"User-MangadexProvider": "Luminary/1.0",
			"Referer":               "https://mangadex.org",
		},

		// Endpoints configuration
		Endpoints: map[string]api.EndpointConfig{
			"manga": {
				Path:         "/manga/{id}",
				Method:       "GET",
				ResponseType: &MgdMangaResp{},
			},
			"chapter": {
				Path:         "/chapter/{id}",
				Method:       "GET",
				ResponseType: &MgdChapterResp{},
			},
			"chapterPages": {
				Path:         "/at-home/server/{id}",
				Method:       "GET",
				ResponseType: &MgdPagesResp{},
			},
			"chapters": {
				Path:         "/manga/{id}/feed",
				Method:       "GET",
				ResponseType: &MgdChapterListResp{},
			},
			"search": {
				Path:         "/manga",
				Method:       "GET",
				ResponseType: &MgdSearchResp{},
			},
		},

		// Custom query formatters
		QueryFormatters: map[string]api.QueryFormatter{
			"search":   formatSearchQuery,
			"chapters": formatChaptersQuery,
		},

		// Custom response processors
		ResponseProcessors: map[string]api.ResponseProcessor{
			"chapter": createChapterProcessor(e, state),
		},

		// Pagination configuration
		PaginationConfig: &search.PaginationConfig{
			LimitParam:     "limit",
			OffsetParam:    "offset",
			TotalCountPath: []string{"Total"},
			ItemsPath:      []string{"Data"},
			DefaultLimit:   100,
			MaxLimit:       100,
		},

		// Chapter configuration - this is the key enhancement
		ChapterConfig: api.ChapterFetchConfig{
			EndpointName:      "chapters",
			ResponseItemsPath: []string{"Data"},
			TotalCountPath:    []string{"Total"},
			LimitParamName:    "limit",
			OffsetParamName:   "offset",
			DefaultPageSize:   100,
			MaxPageSize:       100,
			ProcessChapters:   processChapters(state),
		},
	}

	// Configure extractors
	config.ExtractorSets = configureExtractors()

	// Create the API provider
	baseMangadexProvider := api.NewProvider(e, config)

	// Create our custom provider
	prov := &MangadexProvider{
		Provider: baseMangadexProvider.(*api.Provider),
		state:    state,
		engine:   e,
	}

	return prov
}

// processChapters creates a function to process chapter responses
func processChapters(state *MangaDexState) api.ProcessChaptersFunc {
	return func(ctx context.Context, provider *api.Provider, response interface{}, mangaID string) ([]core.ChapterInfo, bool, error) {
		// Cast to the expected response type
		chaptersResp, ok := response.(*MgdChapterListResp)
		if !ok {
			return nil, false, fmt.Errorf("unexpected response type for chapters: %T", response)
		}

		// If we didn't get any data, return empty with no more pages
		if len(chaptersResp.Data) == 0 {
			return []core.ChapterInfo{}, false, nil
		}

		// Process the chapters
		var chapters []core.ChapterInfo
		for _, item := range chaptersResp.Data {
			// Convert string chapter number to float
			var chapterNum float64
			if item.Attributes.Chapter != "" {
				if num, err := strconv.ParseFloat(item.Attributes.Chapter, 64); err == nil {
					chapterNum = num
				}
			}

			// Extract title
			title := item.Attributes.Title
			if title == "" {
				// Build a title from volume and chapter info like HakuNeko does
				if item.Attributes.Volume != "" {
					title += "Vol." + item.Attributes.Volume
				}
				if item.Attributes.Chapter != "" {
					if title != "" {
						title += " "
					}
					title += "Ch." + item.Attributes.Chapter
				}
				if title == "" {
					title = "Untitled"
				}
			}

			// Handle nullable date
			var publishDate *time.Time
			if !item.Attributes.PublishAt.IsZero() {
				publishDate = &item.Attributes.PublishAt
			}

			// Handle language
			language := convertLanguageCode(item.Attributes.TranslatedLanguage)

			// Create the chapter info
			chapterInfo := core.ChapterInfo{
				ID:       item.ID,
				Title:    title,
				Number:   chapterNum,
				Date:     publishDate,
				Language: language,
			}

			chapters = append(chapters, chapterInfo)
		}

		// Determine if there are more pages
		hasMore := len(chaptersResp.Data) >= chaptersResp.Limit && chaptersResp.Offset+len(chaptersResp.Data) < chaptersResp.Total

		return chapters, hasMore, nil
	}
}

// createChapterProcessor creates a processor for chapter responses
func createChapterProcessor(e *engine.Engine, state *MangaDexState) api.ResponseProcessor {
	return func(response interface{}, chapterID string) (interface{}, error) {
		// Cast to expected response type
		chapterResp, ok := response.(*MgdChapterResp)
		if !ok {
			return nil, fmt.Errorf("unexpected response type: %T", response)
		}

		// Create a new chapter object
		chapter := &core.Chapter{
			Info: core.ChapterInfo{
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

		// Handle nullable publication date
		if !chapterResp.Data.Attributes.PublishAt.IsZero() {
			chapter.Info.Date = &chapterResp.Data.Attributes.PublishAt
		}

		// Handle language
		chapter.Info.Language = convertLanguageCode(chapterResp.Data.Attributes.TranslatedLanguage)

		// Extract manga ID from relationships
		for _, rel := range chapterResp.Data.Relationships {
			if rel.Type == "manga" {
				chapter.MangaID = rel.ID
				break
			}
		}

		// Fetch pages using a separate API call
		apiConfig := network.APIConfig{
			BaseURL:      "https://api.mangadex.org",
			RateLimitKey: "api.mangadex.org",
			RetryCount:   3,
			ThrottleTime: 3 * time.Second,
			DefaultHeaders: map[string]string{
				"User-MangadexProvider": "Luminary/1.0",
				"Referer":               "https://mangadex.org",
			},
			Endpoints: map[string]network.APIEndpoint{
				"chapterPages": {
					Path:          "/at-home/server/{id}",
					Method:        "GET",
					ResponseType:  &MgdPagesResp{},
					PathFormatter: network.DefaultPathFormatter,
				},
			},
		}

		pagesResponse, err := e.API.FetchFromAPI(
			context.Background(),
			apiConfig,
			"chapterPages",
			nil,
			chapterID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch chapter pages: %w", err)
		}

		// Extract pages from the response
		pagesResp, ok := pagesResponse.(*MgdPagesResp)
		if !ok {
			return nil, fmt.Errorf("unexpected response type for pages: %T", pagesResponse)
		}

		// Build pages
		pages := make([]core.Page, len(pagesResp.Chapter.Data))
		for i, file := range pagesResp.Chapter.Data {
			formattedUrl := fmt.Sprintf("%s%s/%s",
				state.ServerNetwork[0],
				pagesResp.Chapter.Hash,
				file)

			pages[i] = core.Page{
				Index:    i,
				URL:      formattedUrl,
				Filename: file,
			}
		}

		// Add pages to the chapter
		chapter.Pages = pages

		return chapter, nil
	}
}

// formatSearchQuery formats search parameters for MangaDex
func formatSearchQuery(params interface{}) url.Values {
	queryParams := url.Values{}

	// Handle search options
	if opts, ok := params.(*core.SearchOptions); ok {
		// Apply the title query
		if opts.Query != "" {
			queryParams.Set("title", opts.Query)
		}

		// Set limit - critical fix: Use the ACTUAL limit from pagination
		if opts.Limit > 0 {
			queryParams.Set("limit", strconv.Itoa(opts.Limit))
		} else {
			// For unlimited results, use maximum allowed by MangaDex (100)
			queryParams.Set("limit", "100")
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
		} else {
			// Default sort by relevance
			queryParams.Set("order[relevance]", "desc")
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

	// Include all content ratings (MangaDex-specific)
	queryParams.Add("contentRating[]", "safe")
	queryParams.Add("contentRating[]", "suggestive")
	queryParams.Add("contentRating[]", "erotica")
	queryParams.Add("contentRating[]", "pornographic")

	return queryParams
}

// formatChaptersQuery formats chapter list parameters for MangaDex
func formatChaptersQuery(params interface{}) url.Values {
	queryParams := url.Values{}
	queryParams.Set("limit", "100")
	queryParams.Set("order[chapter]", "asc")

	// Include all content ratings (MangaDex-specific)
	queryParams.Add("contentRating[]", "safe")
	queryParams.Add("contentRating[]", "suggestive")
	queryParams.Add("contentRating[]", "erotica")
	queryParams.Add("contentRating[]", "pornographic")

	// Add offset if provided in params
	if params != nil {
		if p, ok := params.(struct {
			Offset int
			Limit  int
		}); ok {
			queryParams.Set("offset", strconv.Itoa(p.Offset))
			if p.Limit > 0 {
				queryParams.Set("limit", strconv.Itoa(p.Limit))
			}
		}
	}

	return queryParams
}

// extractBestTitle picks the best title from a map of titles
func extractBestTitle(titleMap map[string]string) string {
	// Prefer English like HakuNeko does
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

// configureExtractors creates the extractors for MangaDex responses
func configureExtractors() map[string]parser.ExtractorSet {
	extractorSets := make(map[string]parser.ExtractorSet)

	// Manga extractor set
	extractorSets["manga"] = parser.ExtractorSet{
		Name:  "MangaDexManga",
		Model: &core.MangaInfo{},
		Extractors: []parser.Extractor{
			{
				Name:       "ID",
				SourcePath: []string{"Data", "ID"},
				TargetPath: "ID",
				Required:   true,
			},
			{
				Name:       "Title",
				SourcePath: []string{"Data", "Attributes", "Title"},
				TargetPath: "Title",
				Transform: func(value interface{}) interface{} {
					if titleMap, ok := value.(map[string]string); ok {
						return extractBestTitle(titleMap)
					}
					return ""
				},
				Required: true,
			},
			{
				Name:       "Description",
				SourcePath: []string{"Data", "Attributes", "Description"},
				TargetPath: "Description",
				Transform: func(value interface{}) interface{} {
					if descMap, ok := value.(map[string]string); ok {
						if enDesc, exists := descMap["en"]; exists && enDesc != "" {
							return enDesc
						}
						for _, desc := range descMap {
							if desc != "" {
								return desc
							}
						}
					}
					return ""
				},
				Required: false,
			},
			{
				Name:       "Status",
				SourcePath: []string{"Data", "Attributes", "Status"},
				TargetPath: "Status",
				Required:   false,
			},
			{
				Name:       "LastUpdated",
				SourcePath: []string{"Data", "Attributes", "UpdatedAt"},
				TargetPath: "LastUpdated",
				Transform: func(value interface{}) interface{} {
					if timeStr, ok := value.(string); ok {
						return parseNullableTime(timeStr)
					}
					return nil
				},
				Required: false,
			},
			{
				Name:       "Tags",
				SourcePath: []string{"Data", "Attributes", "Tags"},
				TargetPath: "Tags",
				Transform: func(value interface{}) interface{} {
					if tags, ok := value.([]interface{}); ok {
						result := make([]string, 0, len(tags))
						for _, tag := range tags {
							if tagMap, ok := tag.(map[string]interface{}); ok {
								if attrs, ok := tagMap["attributes"].(map[string]interface{}); ok {
									if names, ok := attrs["name"].(map[string]string); ok {
										if enName, exists := names["en"]; exists && enName != "" {
											result = append(result, enName)
										}
									}
								}
							}
						}
						return result
					}
					return []string{}
				},
				Required: false,
			},
			{
				Name:       "AltTitles",
				SourcePath: []string{"Data", "Attributes", "AltTitles"},
				TargetPath: "AltTitles",
				Transform: func(value interface{}) interface{} {
					if altTitles, ok := value.([]map[string]string); ok {
						result := make([]string, 0, len(altTitles))
						for _, titleMap := range altTitles {
							for _, title := range titleMap {
								if title != "" {
									result = append(result, title)
									break
								}
							}
						}
						return result
					}
					return []string{}
				},
				Required: false,
			},
			{
				Name:       "Authors",
				SourcePath: []string{"Data", "Relationships"},
				TargetPath: "Authors",
				Transform: func(value interface{}) interface{} {
					if relationships, ok := value.([]interface{}); ok {
						authors := make([]string, 0)
						for _, rel := range relationships {
							if relMap, ok := rel.(map[string]interface{}); ok {
								if relType, ok := relMap["type"].(string); ok && (relType == "author" || relType == "artist") {
									if attrs, ok := relMap["attributes"].(map[string]interface{}); ok {
										if name, ok := attrs["name"].(string); ok && name != "" {
											authors = append(authors, name)
										}
									}
								}
							}
						}
						return authors
					}
					return []string{}
				},
				Required: false,
			},
		},
	}

	// Search results extractor
	extractorSets["search"] = parser.ExtractorSet{
		Name:  "MangaDexSearch",
		Model: &core.Manga{},
		Extractors: []parser.Extractor{
			{
				Name:       "ID",
				SourcePath: []string{"ID"},
				TargetPath: "ID",
				Required:   true,
			},
			{
				Name:       "Title",
				SourcePath: []string{"Attributes", "Title"},
				TargetPath: "Title",
				Transform: func(value interface{}) interface{} {
					if titleMap, ok := value.(map[string]string); ok {
						return extractBestTitle(titleMap)
					}
					return ""
				},
				Required: true,
			},
			// Other extractors for search (description, authors, etc.)
		},
	}

	// Chapter extractor
	extractorSets["chapter"] = parser.ExtractorSet{
		Name:  "MangaDexChapter",
		Model: &core.Chapter{},
		Extractors: []parser.Extractor{
			{
				Name:       "ID",
				SourcePath: []string{"Data", "ID"},
				TargetPath: "Info.ID",
				Required:   true,
			},
			{
				Name:       "Title",
				SourcePath: []string{"Data", "Attributes", "Title"},
				TargetPath: "Info.Title",
				Required:   false,
			},
			{
				Name:       "Date",
				SourcePath: []string{"Data", "Attributes", "PublishAt"},
				TargetPath: "Info.Date",
				Transform: func(value interface{}) interface{} {
					if timeVal, ok := value.(time.Time); ok && !timeVal.IsZero() {
						return &timeVal
					}
					return nil
				},
				Required: false,
			},
			{
				Name:       "Language",
				SourcePath: []string{"Data", "Attributes", "TranslatedLanguage"},
				TargetPath: "Info.Language",
				Transform: func(value interface{}) interface{} {
					if langCode, ok := value.(string); ok {
						return convertLanguageCode(langCode)
					}
					return nil
				},
				Required: false,
			},
		},
	}

	return extractorSets
}
