package agents

import (
	"Luminary/engine"
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// MangaDex implements the Agent interface for MangaDex
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

	// Configure specific settings
	agent.SetSiteURL("https://mangadex.org")
	agent.SetAPIURL("https://api.mangadex.org")

	return agent
}

// OnInitialize initializes the MangaDex agent
func (m *MangaDex) OnInitialize(ctx context.Context) error {
	// Add additional servers to the network
	m.serverNetwork = append(m.serverNetwork, "https://cache.ayaya.red/mdah/data/")
	m.Engine.Logger.Info("Added network seeds '[ %s ]' to %s", strings.Join(m.serverNetwork, ", "), m.Name())
	return nil
}

// Search implements manga search for MangaDex using helper
func (m *MangaDex) Search(ctx context.Context, query string, options engine.SearchOptions) ([]Manga, error) {
	// Use the helper function with our search implementation
	results, err := engine.PerformSearch[[]Manga](
		ctx,
		m,
		query,
		options,
		func(ctx context.Context, query string, options engine.SearchOptions) ([]Manga, error) {
			// Define the response structure
			var response struct {
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

			// Build the query parameters
			params := url.Values{}
			params.Set("title", query)

			// Set limit
			if options.Limit > 0 {
				params.Set("limit", strconv.Itoa(options.Limit))
			} else {
				params.Set("limit", "10") // Default
			}

			// Apply sorting
			if options.Sort != "" {
				switch strings.ToLower(options.Sort) {
				case "relevance":
					params.Set("order[relevance]", "desc")
				case "popularity":
					params.Set("order[followedCount]", "desc")
				case "name":
					params.Set("order[title]", "asc")
				}
			}

			// Apply filters
			if options.Filters != nil {
				for field, value := range options.Filters {
					switch field {
					case "author":
						params.Add("authors[]", value)
					case "genre":
						params.Add("includedTags[]", value)
					}
				}
			}

			// Include all content ratings
			params.Add("contentRating[]", "safe")
			params.Add("contentRating[]", "suggestive")
			params.Add("contentRating[]", "erotica")
			params.Add("contentRating[]", "pornographic")

			// Build the URL
			apiURL := fmt.Sprintf("%s/manga?%s", m.APIURL(), params.Encode())

			// Fetch and parse the JSON
			err := m.FetchJSON(ctx, apiURL, &response)
			if err != nil {
				return nil, err
			}

			// Convert to our generic Manga type
			results := make([]Manga, 0, len(response.Data))
			for _, item := range response.Data {
				// Extract title (prefer English)
				title := m.extractBestTitle(item.Attributes.Title)
				if title == "" {
					continue // Skip entries with no title
				}

				// Extract all alternative titles
				altTitles := m.extractAltTitles(item.Attributes.Title, item.Attributes.AltTitles, title)

				// Extract description
				description := m.extractBestDescription(item.Attributes.Description)

				// Extract tags
				tags := m.extractTags(item.Attributes.Tags)

				// Extract authors
				authors := m.extractAuthors(item.Relationships)

				// Create manga entry
				manga := Manga{
					ID:          item.ID,
					Title:       title,
					AltTitles:   altTitles,
					Description: description,
					Status:      item.Attributes.Status,
					Tags:        tags,
					Authors:     authors,
				}

				results = append(results, manga)
			}

			return results, nil
		},
	)

	return results, err
}

// GetManga retrieves detailed information about a manga using helper
func (m *MangaDex) GetManga(ctx context.Context, id string) (*MangaInfo, error) {
	// Use the helper function with our manga retrieval implementation
	mangaInfo, err := engine.PerformGetManga[*MangaInfo](
		ctx,
		m,
		id,
		m.mangaCache,
		&m.cacheMutex,
		m.apiURL,
		func(ctx context.Context, id string) (*MangaInfo, error) {
			// Define the response structure for manga details
			var response struct {
				Data struct {
					ID         string `json:"id"`
					Attributes struct {
						Title       map[string]string `json:"title"`
						Description map[string]string `json:"description"`
						Status      string            `json:"status"`
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

			// Build the URL
			apiURL := fmt.Sprintf("%s/manga/%s", m.APIURL(), id)

			// Fetch and parse the JSON
			err := m.FetchJSON(ctx, apiURL, &response)
			if err != nil {
				return nil, err
			}

			// Extract title (prefer English)
			title := m.extractBestTitle(response.Data.Attributes.Title)

			// Extract description
			description := m.extractBestDescription(response.Data.Attributes.Description)

			// Extract tags
			tags := m.extractTags(response.Data.Attributes.Tags)

			// Extract authors
			authors := m.extractAuthors(response.Data.Relationships)

			// Fetch chapter list
			chapters, err := m.getChapterList(ctx, id)
			if err != nil {
				return nil, err
			}

			// Create MangaInfo with all the data we collected
			return &MangaInfo{
				Manga: Manga{
					ID:          id,
					Title:       title,
					Description: description,
					Status:      response.Data.Attributes.Status,
					Tags:        tags,
					Authors:     authors,
				},
				Chapters: chapters,
			}, nil
		},
	)

	return mangaInfo, err
}

// GetChapter retrieves detailed information about a chapter using helper
func (m *MangaDex) GetChapter(ctx context.Context, chapterID string) (*Chapter, error) {
	// Use the helper function with our chapter retrieval implementation
	chapter, err := engine.PerformGetChapter[*Chapter](
		ctx,
		m,
		chapterID,
		m.apiURL,
		func(ctx context.Context, chapterID string) (*Chapter, error) {
			// Define response structure
			var response struct {
				Data struct {
					ID         string `json:"id"`
					Attributes struct {
						Title     string    `json:"title"`
						Chapter   string    `json:"chapter"`
						Volume    string    `json:"volume"`
						PublishAt time.Time `json:"publishAt"`
					} `json:"attributes"`
				} `json:"data"`
			}

			// Build API URL
			apiURL := fmt.Sprintf("%s/chapter/%s", m.APIURL(), chapterID)

			// Fetch and parse JSON
			err := m.FetchJSON(ctx, apiURL, &response)
			if err != nil {
				return nil, err
			}

			// Parse chapter number
			chapterNum := 0.0
			if response.Data.Attributes.Chapter != "" {
				num, err := strconv.ParseFloat(response.Data.Attributes.Chapter, 64)
				if err == nil {
					chapterNum = num
				}
			}

			// Create a title that includes volume if available
			title := response.Data.Attributes.Title
			if response.Data.Attributes.Volume != "" && title == "" {
				title = fmt.Sprintf("Vol. %s Ch. %s", response.Data.Attributes.Volume, response.Data.Attributes.Chapter)
			} else if title == "" {
				title = fmt.Sprintf("Chapter %s", response.Data.Attributes.Chapter)
			}

			chapterInfo := ChapterInfo{
				ID:     chapterID,
				Title:  title,
				Number: chapterNum,
				Date:   response.Data.Attributes.PublishAt,
			}

			// Fetch chapter pages
			pages, err := m.getChapterPages(ctx, chapterID)
			if err != nil {
				return nil, err
			}

			return &Chapter{
				Info:  chapterInfo,
				Pages: pages,
			}, nil
		},
	)

	return chapter, err
}

// DownloadChapter downloads a chapter with using the base implementation but ensuring
// that our GetChapter method is called
func (m *MangaDex) DownloadChapter(ctx context.Context, chapterID, destDir string) error {
	// Initialize if needed
	if err := m.Initialize(ctx); err != nil {
		return err
	}

	// Log download request
	m.Engine.Logger.Info("[%s] Downloading chapter: %s to %s", m.id, chapterID, destDir)

	// Get chapter information
	chapter, err := m.GetChapter(ctx, chapterID)
	if err != nil {
		return err
	}

	// Try to get manga info for proper manga title
	var mangaTitle string
	var mangaID string

	manga, err := m.TryGetMangaForChapter(ctx, chapterID)
	if err == nil && manga != nil {
		mangaTitle = manga.Title
		mangaID = manga.ID
	} else {
		// Fall back to using chapter title
		m.Engine.Logger.Debug("[%s] Couldn't find manga for chapter %s, using fallback title", m.id, chapterID)
		mangaTitle = fmt.Sprintf("%s-%s", m.Name(), chapterID)
	}

	// Extract chapter and volume numbers
	chapterNum := &chapter.Info.Number
	if *chapterNum == 0 {
		chapterNum = nil
	}

	// Check for volume override in context
	var volumeNum *int
	if val := ctx.Value("volume_override"); val != nil {
		if volNum, ok := val.(int); ok && volNum > 0 {
			volumeNum = &volNum
		}
	} else if chapter.Info.Title != "" {
		// Try to extract volume from title if not overridden
		_, extractedVol := m.Engine.Metadata.ExtractChapterInfo(chapter.Info.Title)
		volumeNum = extractedVol
	}

	// Prepare metadata
	metadata := engine.ChapterMetadata{
		MangaID:      mangaID,
		MangaTitle:   mangaTitle,
		ChapterID:    chapterID,
		ChapterNum:   chapterNum,
		VolumeNum:    volumeNum,
		ChapterTitle: chapter.Info.Title,
		AgentID:      m.ID(),
	}

	// Convert pages to download requests
	downloadFiles := make([]engine.DownloadRequest, len(chapter.Pages))
	for i, page := range chapter.Pages {
		downloadFiles[i] = engine.DownloadRequest{
			URL:       page.URL,
			Index:     i + 1,
			Filename:  page.Filename,
			PageCount: len(chapter.Pages),
		}
	}

	// Extract concurrency settings from context or use default
	concurrency := m.Engine.Download.MaxConcurrency
	if contextConcurrency, ok := ctx.Value("concurrency").(int); ok && contextConcurrency > 0 {
		concurrency = contextConcurrency
	}

	// Set up download configuration
	config := engine.DownloadJobConfig{
		Metadata:    metadata,
		OutputDir:   destDir,
		Concurrency: concurrency,
		Files:       downloadFiles,
		WaitDuration: func(isRetry bool) {
			if isRetry {
				time.Sleep(m.Engine.HTTP.ThrottleTimeAPI)
			} else {
				time.Sleep(m.Engine.HTTP.ThrottleTimeImages)
			}
		},
	}

	// Log and start download
	m.Engine.Logger.Info("[%s] Downloading %d pages for chapter %s", m.id, len(chapter.Pages), chapterID)

	// Use the engine's download service to download the chapter
	err = m.Engine.Download.DownloadChapter(ctx, config)
	if err != nil {
		m.Engine.Logger.Error("[%s] Download failed: %v", m.id, err)
		return err
	}

	m.Engine.Logger.Info("[%s] Successfully downloaded chapter %s", m.id, chapterID)
	return nil
}

// TryGetMangaForChapter attempts to get manga info for a chapter
func (m *MangaDex) TryGetMangaForChapter(ctx context.Context, chapterID string) (*Manga, error) {
	// Define response structure
	var response struct {
		Data struct {
			Relationships []struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			} `json:"relationships"`
		} `json:"data"`
	}

	// Build API URL
	apiURL := fmt.Sprintf("%s/chapter/%s", m.APIURL(), chapterID)

	// Fetch and parse JSON
	err := m.FetchJSON(ctx, apiURL, &response)
	if err != nil {
		return nil, err
	}

	// Find the manga ID in relationships
	var mangaID string
	for _, rel := range response.Data.Relationships {
		if rel.Type == "manga" {
			mangaID = rel.ID
			break
		}
	}

	if mangaID == "" {
		return nil, fmt.Errorf("manga ID not found for chapter")
	}

	// Now get the manga details
	manga, err := m.GetManga(ctx, mangaID)
	if err != nil {
		return nil, err
	}

	return &manga.Manga, nil
}

// Helper methods for MangaDex specific functionality

// getChapterList fetches all chapters for a manga
func (m *MangaDex) getChapterList(ctx context.Context, mangaID string) ([]ChapterInfo, error) {
	var allChapters []ChapterInfo

	// Limit to max 10 pages to prevent infinite loop
	maxPages := 10

	for page := 0; page < maxPages; page++ {
		chapters, hasMore, err := m.getChapterPage(ctx, mangaID, page)
		if err != nil {
			return nil, err
		}

		// Add chapters to our result list
		allChapters = append(allChapters, chapters...)

		// If no more pages, break the loop
		if !hasMore {
			break
		}
	}

	return allChapters, nil
}

// getChapterPage fetches a single page of chapters
func (m *MangaDex) getChapterPage(ctx context.Context, mangaID string, page int) ([]ChapterInfo, bool, error) {
	// Define response structure
	var response struct {
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

	// Build API URL
	apiURL := fmt.Sprintf("%s/manga/%s/feed?limit=100&offset=%d&order[chapter]=asc",
		m.APIURL(), mangaID, page*100)

	// Fetch and parse JSON
	err := m.FetchJSON(ctx, apiURL, &response)
	if err != nil {
		return nil, false, err
	}

	// Convert response to our ChapterInfo type
	chapters := make([]ChapterInfo, 0, len(response.Data))
	for _, item := range response.Data {
		// Parse chapter number
		chapterNum := 0.0
		if item.Attributes.Chapter != "" {
			num, err := strconv.ParseFloat(item.Attributes.Chapter, 64)
			if err == nil {
				chapterNum = num
			}
		}

		// Create title that includes volume if available
		title := item.Attributes.Title
		if item.Attributes.Volume != "" && title == "" {
			title = fmt.Sprintf("Vol. %s Ch. %s", item.Attributes.Volume, item.Attributes.Chapter)
		} else if title == "" {
			title = fmt.Sprintf("Chapter %s", item.Attributes.Chapter)
		}

		chapters = append(chapters, ChapterInfo{
			ID:     item.ID,
			Title:  title,
			Number: chapterNum,
			Date:   item.Attributes.PublishAt,
		})
	}

	// Check if there are more chapters to fetch
	hasMore := (response.Offset + response.Limit) < response.Total

	return chapters, hasMore, nil
}

// getChapterPages fetches page data for a chapter
func (m *MangaDex) getChapterPages(ctx context.Context, chapterID string) ([]Page, error) {
	// Define response structure
	var response struct {
		BaseURL string `json:"baseUrl"`
		Chapter struct {
			Hash string   `json:"hash"`
			Data []string `json:"data"`
		} `json:"chapter"`
	}

	// Build API URL
	apiURL := fmt.Sprintf("%s/at-home/server/%s", m.APIURL(), chapterID)

	// Fetch and parse JSON
	err := m.FetchJSON(ctx, apiURL, &response)
	if err != nil {
		return nil, err
	}

	// Convert to Page objects
	pages := make([]Page, len(response.Chapter.Data))
	for i, file := range response.Chapter.Data {
		// Try using the first server in our network
		formattedUrl := fmt.Sprintf("%s%s/%s",
			m.serverNetwork[0],
			response.Chapter.Hash,
			file)

		pages[i] = Page{
			Index:    i,
			URL:      formattedUrl,
			Filename: file,
		}
	}

	return pages, nil
}

// Extraction helper methods

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

func init() {
	Register(NewMangaDex())
}
