// Luminary: A streamlined CLI tool for searching and downloading manga.
// Copyright (C) 2025 Luca M. Schmidt (LuMiSxh)
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package network

import (
	"Luminary/pkg/engine/core"
	"Luminary/pkg/engine/logger"
	"Luminary/pkg/engine/parser"
	"Luminary/pkg/util"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// WebScraperService provides capabilities for scraping web content
type WebScraperService struct {
	HTTP        *HTTPService
	DOM         *parser.DOMService
	RateLimiter *RateLimiterService
	Logger      *logger.Service
}

// NewWebScraperService creates a new web scraper service
func NewWebScraperService(
	http *HTTPService,
	dom *parser.DOMService,
	rateLimiter *RateLimiterService,
	logger *logger.Service,
) *WebScraperService {
	return &WebScraperService{
		HTTP:        http,
		DOM:         dom,
		RateLimiter: rateLimiter,
		Logger:      logger,
	}
}

// ScraperRequest represents a request to scrape a web page
type ScraperRequest struct {
	URL            string            // URL to scrape
	Method         string            // HTTP method (GET or POST)
	Headers        map[string]string // HTTP headers
	Data           url.Values        // Form data for POST requests
	FollowRedirect bool              // Whether to follow redirects
	Timeout        time.Duration     // Request timeout
}

// NewScraperRequest creates a new scraper request
func NewScraperRequest(url string) *ScraperRequest {
	return &ScraperRequest{
		URL:            url,
		Method:         "GET",
		Headers:        make(map[string]string),
		FollowRedirect: true,
		Timeout:        30 * time.Second,
	}
}

// SetMethod sets the HTTP method
func (r *ScraperRequest) SetMethod(method string) *ScraperRequest {
	r.Method = method
	return r
}

// SetHeader sets an HTTP header
func (r *ScraperRequest) SetHeader(key, value string) *ScraperRequest {
	r.Headers[key] = value
	return r
}

// SetFormData sets form data for POST requests
func (r *ScraperRequest) SetFormData(data url.Values) *ScraperRequest {
	r.Data = data
	return r
}

// WebPage represents a scraped web page
type WebPage struct {
	URL         string             // The URL of the page
	StatusCode  int                // HTTP status code
	Headers     map[string]string  // Response headers
	RootElement *parser.Element    // The root HTML element
	DOM         *parser.DOMService // Reference to the DOM service
}

// Find finds elements matching a CSS selector within this page
func (p *WebPage) Find(selector string) ([]*parser.Element, error) {
	if p.RootElement == nil {
		return nil, fmt.Errorf("no root element")
	}

	return p.RootElement.Find(selector)
}

// FindOne finds the first element matching a CSS selector within this page
func (p *WebPage) FindOne(selector string) (*parser.Element, error) {
	if p.RootElement == nil {
		return nil, fmt.Errorf("no root element")
	}

	return p.RootElement.FindOne(selector)
}

// GetText gets the text content of the page
func (p *WebPage) GetText() string {
	if p.RootElement == nil {
		return ""
	}

	return p.RootElement.Text()
}

// GetMetaTags extracts meta tags from the page
func (p *WebPage) GetMetaTags() map[string]string {
	if p.RootElement == nil || p.RootElement.Node == nil {
		return make(map[string]string)
	}

	return p.DOM.ExtractMetaTags(p.RootElement.Node)
}

// GetLinks extracts all links from the page
func (p *WebPage) GetLinks() []map[string]string {
	if p.RootElement == nil || p.RootElement.Node == nil {
		return nil
	}

	// Find all anchor elements
	anchors, err := p.Find("a")
	if err != nil {
		return nil
	}

	var links []map[string]string
	for _, a := range anchors {
		href := a.Attr("href")
		if href == "" {
			continue
		}

		// Make absolute URL if needed
		if !strings.HasPrefix(href, "http") {
			href = parser.UrlJoin(p.URL, href)
		}

		link := map[string]string{
			"href": href,
			"text": a.Text(),
		}

		// Get title if available
		title := a.Attr("title")
		if title != "" {
			link["title"] = title
		}

		links = append(links, link)
	}

	return links
}

// GetImages extracts all images from the page
func (p *WebPage) GetImages() []map[string]string {
	if p.RootElement == nil || p.RootElement.Node == nil {
		return nil
	}

	// Find all image elements
	images, err := p.Find("img")
	if err != nil {
		return nil
	}

	var results []map[string]string
	for _, img := range images {
		src := img.Attr("src")
		if src == "" {
			// Try data-src
			src = img.Attr("data-src")
		}

		if src == "" {
			continue
		}

		// Make absolute URL if needed
		if !strings.HasPrefix(src, "http") {
			src = parser.UrlJoin(p.URL, src)
		}

		imgData := map[string]string{
			"src": src,
		}

		// Add alt text if available
		alt := img.Attr("alt")
		if alt != "" {
			imgData["alt"] = alt
		}

		results = append(results, imgData)
	}

	return results
}

// GetTitle gets the page title
func (p *WebPage) GetTitle() string {
	titleElem, err := p.FindOne("title")
	if err != nil || titleElem == nil {
		return ""
	}

	return titleElem.Text()
}

// FetchPage fetches and parses a web page
func (s *WebScraperService) FetchPage(ctx context.Context, req *ScraperRequest) (*WebPage, error) {
	// Apply rate limiting
	host, err := extractHost(req.URL)
	if err == nil {
		s.RateLimiter.Wait(host)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Add default headers if not set
	if httpReq.Header.Get("User-Provider") == "" {
		httpReq.Header.Set("User-Provider", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.131 Safari/537.36")
	}

	if httpReq.Header.Get("Accept") == "" {
		httpReq.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	}

	// Set form data for POST requests
	if req.Method == "POST" && req.Data != nil {
		httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		httpReq.Body = http.NoBody
		httpReq.GetBody = func() (r io.ReadCloser, err error) {
			return http.NoBody, nil
		}
		httpReq.ContentLength = 0

		if len(req.Data) > 0 {
			body := strings.NewReader(req.Data.Encode())
			rc := io.NopCloser(body)
			httpReq.Body = rc
			httpReq.GetBody = func() (io.ReadCloser, error) {
				return io.NopCloser(strings.NewReader(req.Data.Encode())), nil
			}
			httpReq.ContentLength = int64(len(req.Data.Encode()))
		}
	}

	// Create HTTP client with timeout
	client := s.HTTP.DefaultClient
	if req.Timeout > 0 {
		client = &http.Client{
			Timeout:   req.Timeout,
			Transport: s.HTTP.DefaultClient.Transport,
		}
	}

	// Set redirect policy
	if !req.FollowRedirect {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	// Make the request
	s.Logger.Debug("Fetching URL: %s", req.URL)
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			s.Logger.Warn("Failed to close response body: %v", err)
		}
	}(resp.Body)

	// Check status code
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("page not found (404): %s", req.URL)
		}

		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse HTML
	rootElement, err := s.DOM.ParseHTML(string(body))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Create response headers map
	headers := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	// Create web page
	page := &WebPage{
		URL:         req.URL,
		StatusCode:  resp.StatusCode,
		Headers:     headers,
		RootElement: rootElement,
		DOM:         s.DOM,
	}

	return page, nil
}

// MadaraScraper provides specialized methods for scraping Madara-based manga sites
type MadaraScraper struct {
	WebScraper    *WebScraperService
	SiteURL       string
	Logger        *logger.Service
	QueryMangas   string
	QueryChapters string
	QueryPages    string
}

// NewMadaraScraper creates a new Madara scraper
func NewMadaraScraper(webScraper *WebScraperService, siteURL string, logger *logger.Service) *MadaraScraper {
	return &MadaraScraper{
		WebScraper:    webScraper,
		SiteURL:       siteURL,
		Logger:        logger,
		QueryMangas:   "div.post-title h3 a, div.post-title h5 a", // Default selectors
		QueryChapters: "li.wp-manga-chapter > a",
		QueryPages:    "div.page-break source",
	}
}

// SetSelectors sets the CSS selectors for the scraper
func (m *MadaraScraper) SetSelectors(mangas, chapters, pages string) {
	if mangas != "" {
		m.QueryMangas = mangas
	}

	if chapters != "" {
		m.QueryChapters = chapters
	}

	if pages != "" {
		m.QueryPages = pages
	}
}

// FetchMangaList fetches the manga list with pagination
func (m *MadaraScraper) FetchMangaList(ctx context.Context, pageNum int, limit int) ([]core.Manga, error) {
	m.Logger.Debug("Fetching Madara manga list page %d with limit %d", pageNum, limit)

	// Create form data for the AJAX request
	formData := url.Values{}
	formData.Set("action", "madara_load_more")
	formData.Set("template", "madara-core/content/content-archive")
	formData.Set("page", fmt.Sprintf("%d", pageNum))
	formData.Set("vars[paged]", "0")
	formData.Set("vars[post_type]", "wp-manga")

	if limit <= 0 {
		limit = 250 // Default to 250 manga per page
	}
	formData.Set("vars[posts_per_page]", fmt.Sprintf("%d", limit))

	// Create request
	req := NewScraperRequest(parser.UrlJoin(m.SiteURL, "/wp-admin/admin-ajax.php"))
	req.SetMethod("POST")
	req.SetHeader("Content-Type", "application/x-www-form-urlencoded")
	req.SetHeader("X-Requested-With", "XMLHttpRequest")
	req.SetHeader("Referer", m.SiteURL)
	req.SetFormData(formData)

	// Fetch page
	webPage, err := m.WebScraper.FetchPage(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manga list: %w", err)
	}

	// Find manga elements
	mangaElements, err := webPage.Find(m.QueryMangas)
	if err != nil {
		return nil, fmt.Errorf("failed to find manga elements: %w", err)
	}

	// Extract manga data
	var mangas []core.Manga
	for _, elem := range mangaElements {
		href := elem.Attr("href")
		if href == "" {
			continue
		}

		// Extract manga ID from URL
		id := ExtractPathFromURL(href)

		manga := core.Manga{
			ID:    id,
			Title: elem.Text(),
		}

		mangas = append(mangas, manga)
	}

	m.Logger.Info("Found %d manga on page %d", len(mangas), pageNum)
	return mangas, nil
}

// FetchMangaChapters fetches chapters for a manga
func (m *MadaraScraper) FetchMangaChapters(ctx context.Context, mangaID string) ([]core.ChapterInfo, error) {
	m.Logger.Debug("Fetching chapters for manga: %s", mangaID)

	// Create request for the manga page
	mangaURL := parser.UrlJoin(m.SiteURL, mangaID)
	req := NewScraperRequest(mangaURL)

	// Fetch manga page
	webPage, err := m.WebScraper.FetchPage(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manga page: %w", err)
	}

	// Try to find the chapters placeholder for AJAX loading
	placeholder, err := webPage.FindOne("[id^=\"manga-chapters-holder\"][data-id]")

	var chapters []core.ChapterInfo

	// If found, try to get chapters via AJAX
	if err == nil && placeholder != nil {
		dataID := placeholder.Attr("data-id")
		if dataID != "" {
			chapters, err = m.fetchChaptersViaAjax(ctx, mangaID, dataID)
			if err != nil {
				m.Logger.Warn("Failed to get chapters via AJAX: %v", err)
				// Fall back to direct scraping
			} else if len(chapters) > 0 {
				return chapters, nil
			}
		}
	}

	// Direct scraping from the page
	chapterElements, err := webPage.Find(m.QueryChapters)
	if err != nil {
		return nil, fmt.Errorf("failed to find chapter elements: %w", err)
	}

	// Extract chapter data
	for _, elem := range chapterElements {
		href := elem.Attr("href")
		if href == "" {
			continue
		}

		// Extract chapter ID
		id := ExtractPathFromURL(href)

		// Get chapter title
		title := elem.Text()

		// Extract chapter number
		chapterNumber := parser.ExtractChapterNumber(title)

		// Try to detect language from title or URL
		language := util.DetectLanguageFromText(title)
		if language == nil {
			language = util.DetectLanguageFromText(href)
		}

		// Try to parse date if available
		var date *time.Time
		dateElement, err := elem.FindOne("span.chapter-release-date, .chapter-time")
		if err == nil && dateElement != nil {
			dateText := dateElement.Text()
			date = util.ParseNullableDate(dateText)
		}

		chapter := core.ChapterInfo{
			ID:       id,
			Title:    title,
			Number:   chapterNumber,
			Date:     date,
			Language: language,
		}

		chapters = append(chapters, chapter)
	}

	m.Logger.Info("Found %d chapters for manga: %s", len(chapters), mangaID)
	return chapters, nil
}

// fetchChaptersViaAjax tries to get chapters via the WordPress AJAX API
func (m *MadaraScraper) fetchChaptersViaAjax(ctx context.Context, mangaID, dataID string) ([]core.ChapterInfo, error) {
	// Try the new AJAX endpoint first
	newEndpoint := parser.UrlJoin(m.SiteURL, mangaID, "ajax/chapters/")
	newReq := NewScraperRequest(newEndpoint)
	newReq.SetMethod("POST")
	newReq.SetHeader("X-Requested-With", "XMLHttpRequest")

	newPage, err := m.WebScraper.FetchPage(ctx, newReq)
	if err == nil {
		chapters, err := m.extractChaptersFromPage(newPage)
		if err == nil && len(chapters) > 0 {
			return chapters, nil
		}
	}

	// Try the old AJAX endpoint
	oldEndpoint := parser.UrlJoin(m.SiteURL, "wp-admin/admin-ajax.php")
	formData := url.Values{}
	formData.Set("action", "manga_get_chapters")
	formData.Set("manga", dataID)

	oldReq := NewScraperRequest(oldEndpoint)
	oldReq.SetMethod("POST")
	oldReq.SetHeader("Content-Type", "application/x-www-form-urlencoded")
	oldReq.SetHeader("X-Requested-With", "XMLHttpRequest")
	oldReq.SetFormData(formData)

	oldPage, err := m.WebScraper.FetchPage(ctx, oldReq)
	if err != nil {
		return nil, fmt.Errorf("both AJAX methods failed: %w", err)
	}

	return m.extractChaptersFromPage(oldPage)
}

// extractChaptersFromPage extracts chapter information from a page
func (m *MadaraScraper) extractChaptersFromPage(page *WebPage) ([]core.ChapterInfo, error) {
	chapterElements, err := page.Find(m.QueryChapters)
	if err != nil {
		return nil, fmt.Errorf("failed to find chapter elements: %w", err)
	}

	var chapters []core.ChapterInfo

	for _, elem := range chapterElements {
		href := elem.Attr("href")
		if href == "" {
			continue
		}

		// Extract chapter ID
		id := ExtractPathFromURL(href)

		// Get chapter title
		title := elem.Text()

		// Extract chapter number
		chapterNumber := parser.ExtractChapterNumber(title)

		// Try to detect language from title or URL
		language := util.DetectLanguageFromText(title)
		if language == nil {
			language = util.DetectLanguageFromText(href)
		}

		// Try to parse date if available
		var date *time.Time
		dateElement, err := elem.FindOne("span.chapter-release-date, .chapter-time")
		if err == nil && dateElement != nil {
			dateText := dateElement.Text()
			date = util.ParseNullableDate(dateText)
		}

		chapter := core.ChapterInfo{
			ID:       id,
			Title:    title,
			Number:   chapterNumber,
			Date:     date,
			Language: language,
		}

		chapters = append(chapters, chapter)
	}

	return chapters, nil
}

// FetchChapterPages fetches pages for a chapter
func (m *MadaraScraper) FetchChapterPages(ctx context.Context, chapterID string) ([]core.Page, error) {
	m.Logger.Debug("Fetching pages for chapter: %s", chapterID)

	// Create request for the chapter page with style=list
	chapterURL := parser.UrlJoin(m.SiteURL, chapterID)
	if !strings.Contains(chapterURL, "?") {
		chapterURL += "?style=list"
	} else {
		chapterURL += "&style=list"
	}

	req := NewScraperRequest(chapterURL)

	// Fetch chapter page
	webPage, err := m.WebScraper.FetchPage(ctx, req)
	if err != nil {
		// If the style=list parameter didn't work (CloudFlare rule), try without it
		if strings.Contains(chapterURL, "style=list") {
			m.Logger.Warn("Failed with style=list, trying without style parameter")

			chapterURL = strings.Replace(chapterURL, "?style=list", "", 1)
			chapterURL = strings.Replace(chapterURL, "&style=list", "", 1)

			req = NewScraperRequest(chapterURL)
			webPage, err = m.WebScraper.FetchPage(ctx, req)

			if err != nil {
				return nil, fmt.Errorf("failed to fetch chapter page: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to fetch chapter page: %w", err)
		}
	}

	// Find page elements
	pageElements, err := webPage.Find(m.QueryPages)
	if err != nil {
		return nil, fmt.Errorf("failed to find page elements: %w", err)
	}

	// Extract page data
	var pages []core.Page
	for i, elem := range pageElements {
		// Get image URL from various attributes
		imageURL := elem.Attr("src")
		if imageURL == "" {
			imageURL = elem.Attr("data-src")
		}
		if imageURL == "" {
			imageURL = elem.Attr("data-url")
		}
		if imageURL == "" {
			imageURL = elem.Attr("data-lazy-src")
		}

		// Skip if no image URL found
		if imageURL == "" {
			continue
		}

		// Extract filename from URL
		urlParts := strings.Split(imageURL, "/")
		filename := urlParts[len(urlParts)-1]

		// Create page
		page := core.Page{
			Index:    i,
			URL:      imageURL,
			Filename: filename,
		}

		pages = append(pages, page)
	}

	m.Logger.Info("Found %d pages for chapter: %s", len(pages), chapterID)
	return pages, nil
}

// ExtractPathFromURL extracts the path part of a URL relative to the site URL
func ExtractPathFromURL(fullURL string) string {
	// Remove protocol and domain
	path := fullURL

	// Remove protocol
	if strings.Contains(path, "://") {
		parts := strings.SplitN(path, "://", 2)
		if len(parts) > 1 {
			path = parts[1]
		}
	}

	// Remove domain
	if strings.Contains(path, "/") {
		domainEnd := strings.Index(path, "/")
		if domainEnd > 0 {
			path = path[domainEnd+1:]
		}
	}

	// Remove trailing slash
	path = strings.TrimSuffix(path, "/")

	// Remove query parameters
	if strings.Contains(path, "?") {
		parts := strings.SplitN(path, "?", 2)
		path = parts[0]
	}

	// Remove fragment
	if strings.Contains(path, "#") {
		parts := strings.SplitN(path, "#", 2)
		path = parts[0]
	}

	return path
}

// extractHost extracts the hostname from a URL
func extractHost(urlStr string) (string, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	return u.Host, nil
}
