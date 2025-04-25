package agents

import (
	"Luminary/utils"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// MangaDex implements the Agent interface for MangaDex
type MangaDex struct {
	*BaseAgent
	serverNetwork []string
}

// Initialize adds cache servers and performs initial setup
func (m *MangaDex) Initialize(ctx context.Context) error {
	// Add additional servers to the network
	m.serverNetwork = append(m.serverNetwork, "https://cache.ayaya.red/mdah/data/")
	fmt.Printf("Added Network Seeds '[ %s ]' to %s\n", strings.Join(m.serverNetwork, ", "), m.Name())
	return nil
}

// Search implements the Agent interface
func (m *MangaDex) Search(ctx context.Context, query string, options SearchOptions) ([]Manga, error) {
	m.Wait(true) // Throttle API requests

	// Build the URL with query parameters
	apiURL, err := url.Parse(fmt.Sprintf("%s/manga", m.ApiURL))
	if err != nil {
		return nil, err
	}

	params := apiURL.Query()
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

	apiURL.RawQuery = params.Encode()

	// Make the request
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Referer", m.SiteURL)
	req.Header.Set("User-Agent", "Luminary/1.0")

	resp, err := m.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch search results: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Parse the response with proper structure for titles
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

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to our generic Manga type
	results := make([]Manga, 0, len(response.Data))
	for _, item := range response.Data {
		// Extract title (prefer English)
		title := ""
		if enTitle, ok := item.Attributes.Title["en"]; ok {
			title = enTitle
		} else {
			// If no English title, try other languages
			for _, t := range item.Attributes.Title {
				title = t
				break // Take the first one we find
			}
		}

		if title == "" {
			continue // Skip entries with no title
		}

		// Extract all alternative titles
		var altTitles []string

		// Process all language variants from the main title
		for _, t := range item.Attributes.Title {
			if t != title { // Don't include the main title again
				altTitles = append(altTitles, t)
			}
		}

		// Process all entries in altTitles array
		for _, titleMap := range item.Attributes.AltTitles {
			for _, t := range titleMap {
				if t != title && t != "" { // Avoid duplicates and empty strings
					altTitles = append(altTitles, t)
				}
			}
		}

		// Extract description
		description := ""
		if enDesc, ok := item.Attributes.Description["en"]; ok {
			description = enDesc
		} else {
			for _, d := range item.Attributes.Description {
				description = d
				break
			}
		}

		// Extract tags
		tags := make([]string, 0, len(item.Attributes.Tags))
		for _, tag := range item.Attributes.Tags {
			if name, ok := tag.Attributes.Name["en"]; ok {
				tags = append(tags, name)
			}
		}

		// Extract authors
		var authors []string
		for _, rel := range item.Relationships {
			if rel.Type == "author" || rel.Type == "artist" {
				if rel.Attributes.Name != "" {
					authors = append(authors, rel.Attributes.Name)
				}
			}
		}

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
}

// GetManga retrieves detailed information about a manga
func (m *MangaDex) GetManga(ctx context.Context, id string) (*MangaInfo, error) {
	m.Wait(true) // Throttle API requests

	// Fetch basic manga info
	apiURL := fmt.Sprintf("%s/manga/%s", m.ApiURL, id)
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Referer", m.SiteURL)
	req.Header.Set("User-Agent", "Luminary/1.0")

	resp, err := m.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manga: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}(resp.Body)

	// Parse the response to extract manga details
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

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode manga details: %w", err)
	}

	// Extract title (prefer English)
	title := ""
	if enTitle, ok := response.Data.Attributes.Title["en"]; ok {
		title = enTitle
	} else {
		// If no English title, try other languages
		for _, t := range response.Data.Attributes.Title {
			title = t
			break // Take the first one we find
		}
	}

	// Extract description
	description := ""
	if enDesc, ok := response.Data.Attributes.Description["en"]; ok {
		description = enDesc
	} else {
		for _, d := range response.Data.Attributes.Description {
			description = d
			break
		}
	}

	// Extract tags
	tags := make([]string, 0, len(response.Data.Attributes.Tags))
	for _, tag := range response.Data.Attributes.Tags {
		if name, ok := tag.Attributes.Name["en"]; ok {
			tags = append(tags, name)
		}
	}

	// Extract authors
	var authors []string
	for _, rel := range response.Data.Relationships {
		if rel.Type == "author" || rel.Type == "artist" {
			if rel.Attributes.Name != "" {
				authors = append(authors, rel.Attributes.Name)
			}
		}
	}

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
}

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
// Returns chapters, hasMore flag, and error
func (m *MangaDex) getChapterPage(ctx context.Context, mangaID string, page int) ([]ChapterInfo, bool, error) {
	m.Wait(true) // Throttle API requests

	// Build API URL for fetching chapters
	apiURL := fmt.Sprintf("%s/manga/%s/feed?limit=100&offset=%d&order[chapter]=asc",
		m.ApiURL, mangaID, page*100)

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Referer", m.SiteURL)
	req.Header.Set("User-Agent", "Luminary/1.0")

	// Make the request
	resp, err := m.Client.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("failed to fetch chapters: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Parse the response
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

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, false, fmt.Errorf("failed to decode response: %w", err)
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

// GetChapter retrieves detailed information about a chapter
func (m *MangaDex) GetChapter(ctx context.Context, chapterID string) (*Chapter, error) {
	m.Wait(true) // Throttle API requests

	// Fetch chapter info
	apiURL := fmt.Sprintf("%s/chapter/%s", m.ApiURL, chapterID)
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := m.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch chapter: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}(resp.Body)

	// Parse chapter response
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

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode chapter info: %w", err)
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
}

// getChapterPages fetches page data for a chapter
func (m *MangaDex) getChapterPages(ctx context.Context, chapterID string) ([]Page, error) {
	m.Wait(true) // Throttle API requests

	// Fetch from at-home API
	apiURL := fmt.Sprintf("%s/at-home/server/%s", m.ApiURL, chapterID)
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := m.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pages: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}(resp.Body)

	var response struct {
		BaseURL string `json:"baseUrl"`
		Chapter struct {
			Hash string   `json:"hash"`
			Data []string `json:"data"`
		} `json:"chapter"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
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

// DownloadChapter downloads a chapter directly to the three-level hierarchy
func (m *MangaDex) DownloadChapter(ctx context.Context, chapterID, destDir string) error {
	// First, get detailed chapter info
	chapter, err := m.GetChapter(ctx, chapterID)
	if err != nil {
		return err
	}

	// Extract chapter number from the existing chapter info
	chapterNum := &chapter.Info.Number
	if *chapterNum == 0 {
		// If chapter number is 0, treat as nil
		chapterNum = nil
	}

	// Extract volume number from the chapter title if available
	var volumeNum *int

	// Check for volume override in context first
	volumeOverride, hasOverride := utils.GetVolumeOverride(ctx)
	if hasOverride {
		volumeNum = volumeOverride
	} else {
		// No override, try to extract from title
		if chapter.Info.Title != "" {
			// Try to extract volume number from title
			volPattern := regexp.MustCompile(`(?i)(?:vol|volume)[.\s]*(\d+)`)
			volMatch := volPattern.FindStringSubmatch(chapter.Info.Title)
			if len(volMatch) > 1 {
				volNum, err := strconv.Atoi(volMatch[1])
				if err == nil {
					volumeNum = &volNum
				}
			}
		}

		// If we couldn't extract from title, try to extract from the API data
		if volumeNum == nil && chapter.Info.Title != "" {
			// MangaDex formats titles like "Vol. 3 Ch. 42" when volume info is available
			volPattern := regexp.MustCompile(`(?i)vol\.\s*(\d+)`)
			volMatch := volPattern.FindStringSubmatch(chapter.Info.Title)
			if len(volMatch) > 1 {
				volNum, err := strconv.Atoi(volMatch[1])
				if err == nil {
					volumeNum = &volNum
				}
			}
		}
	}

	// Try to get manga info for proper manga title
	var mangaTitle string
	var mangaID string = ""
	manga, err := m.tryGetMangaForChapter(ctx, chapterID)
	if err == nil && manga != nil {
		mangaTitle = manga.Title
		mangaID = manga.ID
	} else {
		// Fall back to using the chapter title if available
		if chapter.Info.Title != "" {
			// Try to extract manga name from chapter title
			// For example, "One Piece Chapter 42" -> "One Piece"
			parts := strings.Split(chapter.Info.Title, " Chapter ")
			if len(parts) > 1 {
				mangaTitle = parts[0]
			} else {
				// Another attempt: "One Piece Vol. 3 Ch. 42" -> "One Piece"
				parts = strings.Split(chapter.Info.Title, " Vol. ")
				if len(parts) > 1 {
					mangaTitle = parts[0]
				} else {
					// Use the full title as a last resort
					mangaTitle = chapter.Info.Title
				}
			}
		} else {
			// Use agent name + chapter ID as fallback
			mangaTitle = fmt.Sprintf("%s-%s", m.Name(), chapterID)
		}
	}

	// Prepare download job configuration
	metadata := utils.ChapterMetadata{
		MangaID:      mangaID,
		MangaTitle:   mangaTitle,
		ChapterID:    chapterID,
		ChapterNum:   chapterNum,
		VolumeNum:    volumeNum,
		ChapterTitle: chapter.Info.Title,
		AgentID:      m.ID(),
	}

	// Convert pages to download requests
	downloadFiles := make([]utils.DownloadRequest, len(chapter.Pages))
	for i, page := range chapter.Pages {
		downloadFiles[i] = utils.DownloadRequest{
			URL:       page.URL,
			Index:     i + 1,
			Filename:  page.Filename,
			PageCount: len(chapter.Pages),
		}
	}

	// Create download job config
	config := utils.DownloadJobConfig{
		Metadata:    metadata,
		OutputDir:   destDir,
		Concurrency: GetConcurrency(ctx, 5),
		Files:       downloadFiles,
		Client:      m.Client,
		WaitDuration: func(isRetry bool) {
			m.Wait(isRetry)
		},
	}

	// Use the utility function to download the chapter
	return utils.DownloadChapterConcurrently(ctx, config)
}

// tryGetMangaForChapter attempts to get manga info for a chapter
// This is a best-effort function and doesn't return an error if it fails
func (m *MangaDex) tryGetMangaForChapter(ctx context.Context, chapterID string) (*Manga, error) {
	// Get chapter details first - this should include manga ID
	apiURL := fmt.Sprintf("%s/chapter/%s", m.ApiURL, chapterID)
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	// Set headers
	req.Header.Set("Referer", m.SiteURL)
	req.Header.Set("User-Agent", "Luminary/1.0")

	// Make the request
	resp, err := m.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}(resp.Body)

	// Check for success
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Parse the response to extract manga ID
	var response struct {
		Data struct {
			Relationships []struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			} `json:"relationships"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
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

// NewMangaDex creates a new MangaDex agent
func NewMangaDex() *MangaDex {
	return &MangaDex{
		BaseAgent: NewBaseAgent(
			"mgd",
			"MangaDex",
			"World's largest manga community and scanlation site",
			"Experimental",
			[]string{"manga", "high-quality", "multi-lingual"},
		),
		serverNetwork: []string{
			"https://uploads.mangadex.org/data/",
		},
	}
}

func init() {
	md := NewMangaDex()
	md.SiteURL = "https://mangadex.org"
	md.ApiURL = "https://api.mangadex.org"
	Register(md)
}
