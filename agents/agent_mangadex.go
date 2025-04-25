package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// MangaDex implements the Agent interface for MangaDex
type MangaDex struct {
	*BaseAgent
	serverNetwork     []string
	licensedGroupsIDs []string
}

// Initialize adds cache servers and performs initial setup
func (m *MangaDex) Initialize(ctx context.Context) error {
	// Add additional servers to the network
	m.serverNetwork = append(m.serverNetwork, "https://cache.ayaya.red/mdah/data/")
	fmt.Printf("Added Network Seeds '[ %s ]' to %s\n", strings.Join(m.serverNetwork, ", "), m.Name())
	return nil
}

// CanHandleURI checks if this agent can handle the given URI
func (m *MangaDex) CanHandleURI(uri string) bool {
	patterns := []string{
		`https?:\/\/mangadex\.org\/title\/`,
		`https?:\/\/mangastack\.cf\/manga\/`,
		`https?:\/\/manga\.ayaya\.red\/manga\/`,
		`https?:\/\/(www\.)?chibiview\.app\/manga\/`,
		`https?:\/\/cubari\.moe\/read\/mangadex\/`,
	}

	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, uri)
		if matched {
			return true
		}
	}

	return false
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

	// Parse the response
	var response struct {
		Data []struct {
			ID         string `json:"id"`
			Attributes struct {
				Title       map[string]string `json:"title"`
				Description map[string]string `json:"description"`
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
			for _, t := range item.Attributes.Title {
				title = t
				break
			}
		}

		if title == "" {
			continue
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

	// Parse basic manga info
	// (Implementation omitted for brevity - similar to Search method)

	// Fetch chapter list
	chapters, err := m.getChapterList(ctx, id)
	if err != nil {
		return nil, err
	}

	// Create MangaInfo with chapters
	return &MangaInfo{
		Manga: Manga{
			ID:          id,
			Title:       "Example Manga", // Replace with actual data
			Description: "Example description",
		},
		Chapters: chapters,
	}, nil
}

// getChapterList fetches all chapters for a manga
func (m *MangaDex) getChapterList(ctx context.Context, mangaID string) ([]ChapterInfo, error) {
	var allChapters []ChapterInfo

	for page := 0; ; page++ {
		chapters, err := m.getChapterPage(ctx, mangaID, page)
		if err != nil {
			return nil, err
		}

		if len(chapters) == 0 {
			break
		}

		allChapters = append(allChapters, chapters...)
	}

	return allChapters, nil
}

// getChapterPage fetches a single page of chapters
func (m *MangaDex) getChapterPage(ctx context.Context, mangaID string, page int) ([]ChapterInfo, error) {
	m.Wait(true) // Throttle API requests

	// Similar implementation to the earlier getChaptersFromPage function
	// ...

	return []ChapterInfo{
		{
			ID:     "example-chapter-id",
			Title:  "Example Chapter",
			Number: 1.0,
			Date:   time.Now(),
		},
	}, nil
}

// GetChapter retrieves detailed information about a chapter
func (m *MangaDex) GetChapter(ctx context.Context, mangaID, chapterID string) (*Chapter, error) {
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

	// Parse chapter info (omitted for brevity)
	chapterInfo := ChapterInfo{
		ID:     chapterID,
		Title:  "Example Chapter",
		Number: 1.0,
		Date:   time.Now(),
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

// agents/mangadex.go - Updated DownloadChapter method

// DownloadChapter downloads a chapter to the specified directory using concurrent downloads
func (m *MangaDex) DownloadChapter(ctx context.Context, mangaID, chapterID, destDir string) error {
	// Get chapter info and pages
	chapter, err := m.GetChapter(ctx, mangaID, chapterID)
	if err != nil {
		return err
	}

	// Create the destination directory if it doesn't exist
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Helper function to download a single page
	downloadPage := func(page Page) error {
		// Throttle image requests
		m.Wait(false)

		// Create a unique filename for the page
		filename := fmt.Sprintf("%s/%03d_%s", destDir, page.Index, page.Filename)

		// Create the output file
		outFile, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create file for page %d: %w", page.Index, err)
		}
		defer func(outFile *os.File) {
			err := outFile.Close()
			if err != nil {
				fmt.Printf("Error closing file %s: %v\n", filename, err)
			}
		}(outFile)

		// Make HTTP request for the image
		req, err := http.NewRequestWithContext(ctx, "GET", page.URL, nil)
		if err != nil {
			return fmt.Errorf("failed to create request for page %d: %w", page.Index, err)
		}

		resp, err := m.Client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to download page %d: %w", page.Index, err)
		}
		// Close the response body in this scope, not with defer
		// This prevents resource leaks during concurrent downloads
		if resp.Body != nil {
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
					fmt.Printf("Error closing response body for page %d: %v\n", page.Index, err)
				}
			}(resp.Body)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("server returned %d for page %d", resp.StatusCode, page.Index)
		}

		// Copy the image data to the file
		_, err = io.Copy(outFile, resp.Body)
		if err != nil {
			return fmt.Errorf("failed to save page %d: %w", page.Index, err)
		}

		fmt.Printf("Downloaded page %d to %s\n", page.Index, filename)
		return nil
	}

	// Set up concurrency control - read from context with default of 5
	concurrentLimit := GetConcurrency(ctx, 5)

	// Create a worker pool
	type downloadTask struct {
		page Page
		err  error
	}

	// Create channels for the worker pool
	tasks := make(chan Page, len(chapter.Pages))
	results := make(chan downloadTask, len(chapter.Pages))

	// Start worker goroutines
	for w := 0; w < concurrentLimit; w++ {
		go func() {
			for page := range tasks {
				err := downloadPage(page)
				results <- downloadTask{page: page, err: err}
			}
		}()
	}

	// Send tasks to the workers
	for _, page := range chapter.Pages {
		tasks <- page
	}
	close(tasks)

	// Collect results and check for errors
	var errs []error
	for i := 0; i < len(chapter.Pages); i++ {
		result := <-results
		if result.err != nil {
			errs = append(errs, result.err)
		}
	}

	// If any errors occurred, return them
	if len(errs) > 0 {
		return fmt.Errorf("failed to download %d pages: %v", len(errs), errs[0])
	}

	return nil
}

// NewMangaDex creates a new MangaDex agent
func NewMangaDex() *MangaDex {
	return &MangaDex{
		BaseAgent: NewBaseAgent(
			"mangadex",
			"MangaDex",
			"World's largest manga community and scanlation site",
			StatusStable,
			[]string{"manga", "high-quality", "multi-lingual"},
		),
		serverNetwork: []string{
			"https://uploads.mangadex.org/data/",
		},
		licensedGroupsIDs: []string{
			"4f1de6a2-f0c5-4ac5-bce5-02c7dbb67deb", // MangaPlus
			"8d8ecf83-8d42-4f8c-add8-60963f9f28d9", // Comikey
		},
	}
}

func init() {
	md := NewMangaDex()
	md.SiteURL = "https://mangadex.org"
	md.ApiURL = "https://api.mangadex.org"
	Register(md)
}
