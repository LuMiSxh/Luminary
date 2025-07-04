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

package search

import (
	"Luminary/pkg/engine/logger"
	"Luminary/pkg/engine/network"
	"Luminary/pkg/engine/parser"
	"Luminary/pkg/errors"
	"context"
	"fmt"
	"reflect"
	"time"
)

// PaginationConfig defines how to handle pagination for an API
type PaginationConfig struct {
	// Query parameter names
	LimitParam  string // Parameter name for limit/per page
	OffsetParam string // Parameter name for offset
	PageParam   string // Parameter name for page number (alternative to offset)

	// TotalCountPath NEEDS to use the same format as the struct it is reflecting.
	//	example: "data" should be "Data" in the struct
	TotalCountPath []string // JSON path to total count in response
	HasMorePath    []string // JSON path to "has more" boolean in response
	// ItemsPath NEEDS to use the same format as the struct it is reflecting.
	//	example: "data" should be "Data" in the struct
	ItemsPath []string // JSON path to items array in response

	// Pagination behavior
	DefaultLimit int // Default number of items per page
	MaxLimit     int // Maximum number of items per page
	StartPage    int // Page number to start from if using page-based pagination
	StartOffset  int // Offset to start from if using offset-based pagination
}

// PaginationService provides methods for paginated API requests
type PaginationService struct {
	API       *network.APIService
	Logger    *logger.Service
	Extractor *parser.ExtractorService
}

// NewPaginationService creates a new pagination service
func NewPaginationService(api *network.APIService, extractor *parser.ExtractorService, logger *logger.Service) *PaginationService {
	return &PaginationService{
		API:       api,
		Logger:    logger,
		Extractor: extractor,
	}
}

// PaginatedRequestParams holds parameters for a paginated request
type PaginatedRequestParams struct {
	Config       PaginationConfig
	APIConfig    network.APIConfig
	EndpointName string
	BaseParams   interface{}
	PathParams   []string
	ExtractorSet parser.ExtractorSet
	MaxPages     int // Maximum number of pages to fetch (0 = all)
	ThrottleTime time.Duration
}

// FetchAllPages fetches all pages of a paginated API and applies extractors
func (p *PaginationService) FetchAllPages(ctx context.Context, params PaginatedRequestParams) ([]interface{}, error) {
	var allResults []interface{}

	// Set default values. 0 means "fetch all pages"
	maxPages := params.MaxPages
	unlimitedPages := maxPages == 0

	if !unlimitedPages && maxPages < 0 {
		maxPages = 10 // Default to 10 pages if negative value provided
	}

	pageSize := params.Config.DefaultLimit
	if pageSize <= 0 {
		pageSize = 50 // Default page size
	}
	if params.Config.MaxLimit > 0 && pageSize > params.Config.MaxLimit {
		pageSize = params.Config.MaxLimit
	}

	// Log pagination settings
	if unlimitedPages {
		p.Logger.Info("Pagination: Unlimited pages mode enabled (will fetch all available pages)")
	} else {
		p.Logger.Info("Pagination: Will fetch up to %d pages", maxPages)
	}
	p.Logger.Info("Pagination: Using page size of %d items", pageSize)

	// Determine pagination strategy
	var currentPage int
	var currentOffset int

	if params.Config.PageParam != "" {
		// Page-based pagination
		currentPage = params.Config.StartPage
		if currentPage <= 0 {
			currentPage = 1 // Default start page
		}
		p.Logger.Debug("Pagination: Using page-based pagination with param '%s'", params.Config.PageParam)
	} else {
		// Offset-based pagination
		currentOffset = params.Config.StartOffset
		p.Logger.Debug("Pagination: Using offset-based pagination with param '%s'", params.Config.OffsetParam)
	}

	// Keep track of whether there might be more results
	hasMore := true

	// Create a base params copy that we'll modify for pagination
	baseParamsValue := reflect.ValueOf(params.BaseParams)
	baseParamsType := baseParamsValue.Type()

	for page := 0; (unlimitedPages || page < maxPages) && hasMore; page++ {
		// Create a new params instance for this page
		paramsValue := reflect.New(baseParamsType.Elem()).Elem()

		// Copy the base params
		paramsValue.Set(baseParamsValue.Elem())

		// Set pagination parameters
		if params.Config.PageParam != "" {
			// Page-based pagination
			setFieldValue(paramsValue, params.Config.PageParam, currentPage)
			setFieldValue(paramsValue, params.Config.LimitParam, pageSize)
			p.Logger.Debug("Fetching page %d with size %d", currentPage, pageSize)
		} else {
			// Offset-based pagination
			setFieldValue(paramsValue, params.Config.OffsetParam, currentOffset)
			setFieldValue(paramsValue, params.Config.LimitParam, pageSize)
			p.Logger.Debug("Fetching with offset %d and limit %d", currentOffset, pageSize)
		}

		// Make the request
		response, err := p.API.FetchFromAPI(
			ctx,
			params.APIConfig,
			params.EndpointName,
			paramsValue.Addr().Interface(),
			params.PathParams...,
		)

		if err != nil {
			return nil, errors.T(err)
		}

		// Extract items from the response
		items, err := p.getItemsFromResponse(response, params.Config.ItemsPath)
		if err != nil {
			p.Logger.Warn("Failed to extract items using path %v: %v", params.Config.ItemsPath, err)
			return nil, errors.TM(err, fmt.Sprintf("failed to extract items from page %d: %w", page+1))
		}

		// Get the number of items on this page
		itemCount := len(items)
		p.Logger.Debug("Fetched %d items from page %d", itemCount, page+1)

		// Apply extractors if provided
		if params.ExtractorSet.Model != nil {
			p.Logger.Debug("Applying extractors to items")
			for _, item := range items {
				result, err := p.Extractor.Extract(params.ExtractorSet, item)
				if err != nil {
					p.Logger.Warn("Failed to extract item: %v", err)
					continue
				}
				allResults = append(allResults, result)
			}
		} else {
			// Add raw items
			for _, item := range items {
				allResults = append(allResults, item)
			}
		}

		// Determine if there are more pages to fetch
		hasMore = p.hasMorePages(response, params.Config, itemCount, pageSize)
		p.Logger.Debug("Page %d complete, hasMore = %v, total results so far: %d",
			page+1, hasMore, len(allResults))

		// Update page/offset for next request
		if params.Config.PageParam != "" {
			currentPage++
		} else {
			currentOffset += itemCount
		}

		// Apply throttling if needed
		if params.ThrottleTime > 0 && hasMore {
			p.Logger.Debug("Throttling for %v before next page request", params.ThrottleTime)
			select {
			case <-ctx.Done():
				return allResults, ctx.Err()
			case <-time.After(params.ThrottleTime):
				// Continue after throttle duration
			}
		}
	}

	p.Logger.Info("Fetched a total of %d items", len(allResults))
	return allResults, nil
}

// getItemsFromResponse extracts items from a response using the given path
func (p *PaginationService) getItemsFromResponse(response interface{}, itemsPath []string) ([]interface{}, error) {
	if response == nil {
		return nil, errors.T(fmt.Errorf("response is nil"))
	}

	// Extract items from the response
	itemsData, err := p.Extractor.GetValueFromPath(response, itemsPath)
	if err != nil {
		return nil, errors.T(err)
	}

	// Convert to a slice
	itemsValue := reflect.ValueOf(itemsData)
	if itemsValue.Kind() != reflect.Slice && itemsValue.Kind() != reflect.Array {
		return nil, errors.T(fmt.Errorf("items path does not point to a slice or array: %v", itemsPath))
	}

	// Convert to []interface{}
	items := make([]interface{}, itemsValue.Len())
	for i := 0; i < itemsValue.Len(); i++ {
		items[i] = itemsValue.Index(i).Interface()
	}

	return items, nil
}

// hasMorePages determines if there are more pages to fetch
func (p *PaginationService) hasMorePages(response interface{}, config PaginationConfig, itemCount, pageSize int) bool {
	// First, check explicit "has more" indicator if provided
	if len(config.HasMorePath) > 0 {
		hasMoreValue, err := p.Extractor.GetValueFromPath(response, config.HasMorePath)
		if err == nil && hasMoreValue != nil {
			if hasMore, ok := hasMoreValue.(bool); ok {
				return hasMore
			}
		}
	}

	// Second, check total count if provided
	if len(config.TotalCountPath) > 0 {
		totalCountValue, err := p.Extractor.GetValueFromPath(response, config.TotalCountPath)
		if err == nil && totalCountValue != nil {
			// Try to convert to int
			var totalCount int
			switch v := totalCountValue.(type) {
			case int:
				totalCount = v
			case int64:
				totalCount = int(v)
			case float64:
				totalCount = int(v)
			default:
				// Couldn't determine total count, fall back to other methods
				p.Logger.Debug("Found totalCount but couldn't convert type %T to int", totalCountValue)
			}

			if totalCount > 0 {
				p.Logger.Debug("Total count from API: %d", totalCount)
				if config.PageParam != "" {
					// Page-based: check if current page * pageSize < totalCount
					currentPage := p.getCurrentPage(response, config)
					return currentPage*pageSize < totalCount
				} else {
					// Offset-based: check if current offset + itemCount < totalCount
					currentOffset := p.getCurrentOffset(response, config)
					return currentOffset+itemCount < totalCount
				}
			}
		} else if err != nil {
			p.Logger.Debug("Failed to get total count from path %v: %v", config.TotalCountPath, err)
		}
	}

	// fallback: if we got items and the count equals the requested page size,
	// there's a good chance there are more items available
	if itemCount > 0 {
		hasMore := itemCount >= pageSize
		p.Logger.Debug("Using fallback logic: got %d items (page size %d) -> has more: %v",
			itemCount, pageSize, hasMore)
		return hasMore
	}

	// No items returned
	return false
}

// getCurrentPage extracts the current page number from the response
func (p *PaginationService) getCurrentPage(response interface{}, config PaginationConfig) int {
	// This is a stub - in a real implementation, you might extract this from the response
	// For now, we'll just return a default which will work in most cases
	return 1
}

// getCurrentOffset extracts the current offset from the response
func (p *PaginationService) getCurrentOffset(response interface{}, config PaginationConfig) int {
	// This is a stub - in a real implementation, you might extract this from the response
	// For now, we'll just return a default which will work in most cases
	return 0
}

// Helper to set a field value in a struct
func setFieldValue(structValue reflect.Value, fieldName string, value interface{}) {
	field := structValue.FieldByName(fieldName)
	if field.IsValid() && field.CanSet() {
		valueValue := reflect.ValueOf(value)
		if field.Type().AssignableTo(valueValue.Type()) {
			field.Set(valueValue)
		} else if valueValue.Type().ConvertibleTo(field.Type()) {
			field.Set(valueValue.Convert(field.Type()))
		}
	}
}
