package providers

import (
	"Luminary/pkg/core"
	"Luminary/pkg/engine"
	"Luminary/pkg/engine/network"
	"Luminary/pkg/errors"
	"Luminary/pkg/provider/base"
	"Luminary/pkg/provider/common"
	"Luminary/pkg/provider/registry"
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

type MgdSearchResp struct {
	Data   []MgdMangaData `json:"data"`
	Total  int            `json:"total"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

type MgdMangaResp struct {
	Data MgdMangaData `json:"data"`
}

type MgdMangaData struct {
	ID            string             `json:"id"`
	Type          string             `json:"type"`
	Attributes    MgdMangaAttributes `json:"attributes"`
	Relationships []MgdRelationship  `json:"relationships"`
}

type MgdMangaAttributes struct {
	Title       map[string]string   `json:"title"`
	AltTitles   []map[string]string `json:"altTitles"`
	Description map[string]string   `json:"description"`
	Status      string              `json:"status"`
	Tags        []struct {
		Attributes struct {
			Name map[string]string `json:"name"`
		} `json:"attributes"`
	} `json:"tags"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type MgdRelationship struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes *struct {
		Name string `json:"name"`
	} `json:"attributes"`
}

type MgdChapterListResp struct {
	Data   []MgdChapterData `json:"data"`
	Total  int              `json:"total"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
}

type MgdChapterResp struct {
	Data MgdChapterData `json:"data"`
}

type MgdChapterData struct {
	ID         string `json:"id"`
	Attributes struct {
		Title              string    `json:"title"`
		Chapter            string    `json:"chapter"`
		Volume             string    `json:"volume"`
		PublishAt          time.Time `json:"publishAt"`
		TranslatedLanguage string    `json:"translatedLanguage"`
	} `json:"attributes"`
	Relationships []MgdRelationship `json:"relationships"`
}

type MgdPagesResp struct {
	BaseURL string `json:"baseUrl"`
	Chapter struct {
		Hash      string   `json:"hash"`
		Data      []string `json:"data"`
		DataSaver []string `json:"dataSaver"`
	} `json:"chapter"`
}

// Register the provider automatically on startup
func init() {
	registry.Register(NewMangaDexProvider)
}

// NewMangaDexProvider creates a new MangaDex provider instance.
// It uses the base provider framework but overrides most operations
// with custom logic to handle the specifics of the MangaDex API.
func NewMangaDexProvider(e *engine.Engine) engine.Provider {
	// Create a new base provider builder
	b := base.New(e, base.Config{
		// Identity
		ID:          "mgd",
		Name:        "MangaDex",
		Description: "World's largest manga community and scanlation site",
		SiteURL:     "https://mangadex.org",
		// Type
		Type: base.TypeAPI,
		// API Config
		API: &base.APIConfig{
			BaseURL: "https://api.mangadex.org",
		},
		// Common Settings
		Headers: map[string]string{
			"User-Agent": "Luminary/1.0 (https://github.com/lumisxh/luminary)",
			"Referer":    "https://mangadex.org",
		},
		RateLimit: 1 * time.Second, // MangaDex API has a rate limit of 5 requests/second
	})

	// Inject custom implementations for MangaDex's complex API
	// The builder returns a provider instance that we can pass to our custom functions.
	p := b.Build().(*base.Provider)
	return b.WithSearch(customMangaDexSearch(p)).
		WithGetManga(customMangaDexGetManga(p)).
		WithGetChapter(customMangaDexGetChapter(p)).
		Build()
}

// customMangaDexSearch provides the implementation for the Search operation.
func customMangaDexSearch(p *base.Provider) func(context.Context, string, core.SearchOptions) ([]core.Manga, error) {
	return func(ctx context.Context, query string, options core.SearchOptions) ([]core.Manga, error) {
		// Build search URL
		searchURL := p.Config.API.BaseURL + "/manga"
		queryParams := formatSearchQuery(query, options)
		reqURL := searchURL + "?" + queryParams.Encode()
		// Perform network request
		resp, err := p.Engine.Network.Request(ctx, &network.Request{
			URL:     reqURL,
			Method:  "GET",
			Headers: p.Config.Headers,
		})

		if err != nil {
			return nil, errors.Track(err).AsProvider(p.ID()).Error()
		}

		// Parse response
		var searchResp MgdSearchResp
		if err := resp.JSON(&searchResp); err != nil {
			return nil, errors.Track(err).AsProvider(p.ID()).Error()
		}

		// Map API response to core.Manga model
		var results []core.Manga
		for _, mangaData := range searchResp.Data {
			results = append(results, core.Manga{
				ID:    mangaData.ID,
				Title: common.ExtractBestTitle(mangaData.Attributes.Title), // Use common helper
			})
		}

		return results, nil
	}
}

// customMangaDexGetManga provides the implementation for retrieving detailed manga info.
func customMangaDexGetManga(p *base.Provider) func(context.Context, string) (*core.MangaInfo, error) {
	return func(ctx context.Context, id string) (*core.MangaInfo, error) {
		// 1. Fetch main manga details
		mangaURL := fmt.Sprintf("%s/manga/%s?includes[]=author&includes[]=artist&includes[]=cover_art", p.Config.API.BaseURL, id)
		resp, err := p.Engine.Network.Request(ctx, &network.Request{URL: mangaURL, Headers: p.Config.Headers})
		if err != nil {
			return nil, errors.Track(err).AsProvider(p.ID()).Error()
		}

		var mangaResp MgdMangaResp
		if err := resp.JSON(&mangaResp); err != nil {
			return nil, errors.Track(err).AsProvider(p.ID()).Error()
		}

		// Map to core.MangaInfo
		mangaInfo := mapMangaDataToInfo(mangaResp.Data)

		// 2. Fetch all chapters using pagination
		chapters, err := fetchAllChapters(ctx, p, id)
		if err != nil {
			// Log error but continue, so user can see manga info even if chapters fail
			p.Engine.Logger.Error("Failed to fetch chapters for manga %s: %v", id, err)
		}
		mangaInfo.Chapters = chapters

		return mangaInfo, nil
	}
}

// customMangaDexGetChapter provides the implementation for retrieving a chapter's pages.
func customMangaDexGetChapter(p *base.Provider) func(context.Context, string) (*core.Chapter, error) {
	return func(ctx context.Context, chapterID string) (*core.Chapter, error) {
		// 1. Fetch chapter details to get manga ID and other info
		chapterInfoURL := fmt.Sprintf("%s/chapter/%s", p.Config.API.BaseURL, chapterID)
		infoResp, err := p.Engine.Network.Request(ctx, &network.Request{URL: chapterInfoURL, Headers: p.Config.Headers})
		if err != nil {
			return nil, errors.Track(err).AsProvider(p.ID()).Error()
		}
		var chapterResp MgdChapterResp
		if err := infoResp.JSON(&chapterResp); err != nil {
			return nil, errors.Track(err).AsProvider(p.ID()).Error()
		}

		// 2. Fetch page URLs from the at-home server
		pagesURL := fmt.Sprintf("%s/at-home/server/%s", p.Config.API.BaseURL, chapterID)
		pagesResp, err := p.Engine.Network.Request(ctx, &network.Request{URL: pagesURL, Headers: p.Config.Headers})
		if err != nil {
			return nil, errors.Track(err).AsProvider(p.ID()).Error()
		}
		var pagesData MgdPagesResp
		if err := pagesResp.JSON(&pagesData); err != nil {
			return nil, errors.Track(err).AsProvider(p.ID()).Error()
		}

		// 3. Construct the full Chapter object
		return mapChapterDataToChapter(chapterResp.Data, pagesData)
	}
}

// fetchAllChapters handles pagination to retrieve all chapters for a manga.
func fetchAllChapters(ctx context.Context, p *base.Provider, mangaID string) ([]core.ChapterInfo, error) {
	var allChapters []core.ChapterInfo
	var offset = 0
	const limit = 500 // Max limit for this endpoint

	for {
		// Build paginated request URL
		queryParams := formatChaptersQuery(offset, limit)
		reqURL := fmt.Sprintf("%s/manga/%s/feed?%s", p.Config.API.BaseURL, mangaID, queryParams.Encode())

		resp, err := p.Engine.Network.Request(ctx, &network.Request{URL: reqURL, Headers: p.Config.Headers})
		if err != nil {
			return nil, err
		}

		var listResp MgdChapterListResp
		if err := resp.JSON(&listResp); err != nil {
			return nil, err
		}

		// Map and append chapters from the current page
		for _, chapterData := range listResp.Data {
			allChapters = append(allChapters, mapChapterDataToInfo(chapterData))
		}

		// Check if we've fetched all chapters
		if listResp.Total <= offset+listResp.Limit {
			break
		}

		// Prepare for the next page
		offset += listResp.Limit
	}

	return allChapters, nil
}

// mapMangaDataToInfo maps the API response to the core.MangaInfo struct.
func mapMangaDataToInfo(data MgdMangaData) *core.MangaInfo {
	info := &core.MangaInfo{
		Manga: core.Manga{
			ID:          data.ID,
			Title:       common.ExtractBestTitle(data.Attributes.Title),       // Use common helper
			Description: common.ExtractBestTitle(data.Attributes.Description), // Use common helper
			Status:      data.Attributes.Status,
		},
		LastUpdated: &data.Attributes.UpdatedAt,
	}

	for _, alt := range data.Attributes.AltTitles {
		info.Manga.AlternativeTitles = append(info.Manga.AlternativeTitles, common.ExtractBestTitle(alt))
	}

	for _, tag := range data.Attributes.Tags {
		info.Manga.Tags = append(info.Manga.Tags, common.ExtractBestTitle(tag.Attributes.Name))
	}

	for _, rel := range data.Relationships {
		if (rel.Type == "author" || rel.Type == "artist") && rel.Attributes != nil {
			info.Manga.Authors = append(info.Manga.Authors, rel.Attributes.Name)
		}
	}
	return info
}

// mapChapterDataToInfo maps chapter list data to core.ChapterInfo.
func mapChapterDataToInfo(data MgdChapterData) core.ChapterInfo {
	chapterNum, _ := strconv.ParseFloat(data.Attributes.Chapter, 64)
	return core.ChapterInfo{
		ID:       data.ID,
		Title:    data.Attributes.Title,
		Number:   chapterNum,
		Volume:   data.Attributes.Volume,
		Language: data.Attributes.TranslatedLanguage,
		Date:     &data.Attributes.PublishAt,
	}
}

// mapChapterDataToChapter constructs a full core.Chapter from API responses.
func mapChapterDataToChapter(chapterData MgdChapterData, pagesData MgdPagesResp) (*core.Chapter, error) {
	chapterInfo := mapChapterDataToInfo(chapterData)

	chapter := &core.Chapter{
		Info: chapterInfo,
	}

	// Find MangaID from relationships
	for _, rel := range chapterData.Relationships {
		if rel.Type == "manga" {
			chapter.MangaID = rel.ID
			break
		}
	}

	// Construct full page URLs
	pageURLs := pagesData.Chapter.Data
	if len(pageURLs) == 0 {
		pageURLs = pagesData.Chapter.DataSaver // Fallback to data-saver images
	}

	for i, filename := range pageURLs {
		pageURL := fmt.Sprintf("%s/data/%s/%s", pagesData.BaseURL, pagesData.Chapter.Hash, filename)
		chapter.Pages = append(chapter.Pages, core.Page{
			Index:    i,
			URL:      pageURL,
			Filename: filename,
		})
	}

	return chapter, nil
}

// formatSearchQuery creates the query parameters for a manga search request.
func formatSearchQuery(query string, options core.SearchOptions) url.Values {
	p := url.Values{}
	p.Set("title", query)
	p.Set("limit", strconv.Itoa(options.Limit))
	p.Add("order[relevance]", "desc")
	// Include a wide range of content ratings
	for _, rating := range []string{"safe", "suggestive", "erotica", "pornographic"} {
		p.Add("contentRating[]", rating)
	}
	return p
}

// formatChaptersQuery creates query parameters for fetching a manga's chapter feed.
func formatChaptersQuery(offset, limit int) url.Values {
	p := url.Values{}
	p.Set("limit", strconv.Itoa(limit))
	p.Set("offset", strconv.Itoa(offset))
	p.Add("order[volume]", "asc")
	p.Add("order[chapter]", "asc")
	// Include all languages by default in the feed
	p.Add("translatedLanguage[]", "en") // Start with english, but MD returns all if not specified
	for _, rating := range []string{"safe", "suggestive", "erotica", "pornographic"} {
		p.Add("contentRating[]", rating)
	}
	return p
}
