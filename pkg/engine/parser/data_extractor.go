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

package parser

import (
	"Luminary/pkg/engine/logger"
	"Luminary/pkg/errors"
	"fmt"
	"reflect"
)

// Extractor maps API response data to domain models
type Extractor struct {
	Name string // Name of this extractor
	// SourcePath NEEDS to use the same format as the struct it is reflecting.
	//	example: "data" should be "Data" in the struct
	SourcePath []string // Path to extract from in source data
	// TargetPath NEEDS to use the same format as the struct it is reflecting.
	//	example: "data" should be "Data" in the struct
	TargetPath string                        // Path to assign to in target object
	Transform  func(interface{}) interface{} // Optional transformation function
	Required   bool                          // Whether this field is required
}

// ExtractorSet is a collection of extractors for a specific model
type ExtractorSet struct {
	Name       string      // Name of this extractor set
	Model      interface{} // Target model type (e.g., Manga, Chapter)
	Extractors []Extractor // List of extractors to apply
}

// ExtractorService provides methods for extracting data
type ExtractorService struct {
	Logger *logger.Service
}

// NewExtractorService creates a new extractor service
func NewExtractorService(logger *logger.Service) *ExtractorService {
	return &ExtractorService{
		Logger: logger,
	}
}

// Extract data from source to target model using extractors
func (e *ExtractorService) Extract(extractorSet ExtractorSet, responseData interface{}) (interface{}, error) {
	// Create a new instance of the target model
	modelType := reflect.TypeOf(extractorSet.Model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	// Create a new instance of the model
	modelValue := reflect.New(modelType)
	model := modelValue.Interface()

	// Apply each extractor
	for _, extractor := range extractorSet.Extractors {
		// Extract value from source
		sourceValue, err := e.GetValueFromPath(responseData, extractor.SourcePath)
		if err != nil {
			if extractor.Required {
				return nil, errors.TM(err, "required field extraction failed")
			}
			e.Logger.Debug("Optional field extraction failed: %v", err)
			continue
		}

		// Apply transformation if provided
		if extractor.Transform != nil && sourceValue != nil {
			sourceValue = extractor.Transform(sourceValue)
		}

		// Set value in the target model
		if err := e.setValueInPath(model, extractor.TargetPath, sourceValue); err != nil {
			return nil, errors.TM(err, "failed to set value in target model")
		}
	}

	return model, nil
}

// ExtractList extracts a list of items using an extractor set
func (e *ExtractorService) ExtractList(extractorSet ExtractorSet, responseData interface{}, listPath []string) ([]interface{}, error) {
	// Extract the list from the response
	listData, err := e.GetValueFromPath(responseData, listPath)
	if err != nil {
		e.Logger.Warn("Failed to extract list data from path %v: %v", listPath, err)
		return nil, errors.TM(err, "failed to extract list data from path")
	}

	// Ensure it's a slice
	listValue := reflect.ValueOf(listData)
	if listValue.Kind() != reflect.Slice {
		e.Logger.Warn("Expected slice at path %v, got %T", listPath, listData)
		return nil, errors.T(fmt.Errorf("expected slice at path %v, got %T", listPath, listData))
	}

	e.Logger.Debug("Found %d items at path %v", listValue.Len(), listPath)

	// Extract each item
	var results []interface{}
	var extractErrors int

	for i := 0; i < listValue.Len(); i++ {
		item := listValue.Index(i).Interface()
		result, err := e.Extract(extractorSet, item)
		if err != nil {
			e.Logger.Warn("Failed to extract item %d: %v", i, err)
			extractErrors++
			continue
		}
		results = append(results, result)
	}

	if extractErrors > 0 {
		e.Logger.Warn("Failed to extract %d out of %d items", extractErrors, listValue.Len())
	}

	e.Logger.Debug("Successfully extracted %d items", len(results))
	return results, nil
}

// GetValueFromPath retrieves a value from a nested structure using a path
func (e *ExtractorService) GetValueFromPath(data interface{}, path []string) (interface{}, error) {
	if len(path) == 0 {
		return data, nil
	}

	if data == nil {
		return nil, errors.T(fmt.Errorf("data is nil"))
	}

	value := reflect.ValueOf(data)

	// Handle pointers
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return nil, fmt.Errorf("nil pointer")
		}
		value = value.Elem()
	}

	// Navigate through the path
	current := path[0]
	rest := path[1:]

	switch value.Kind() {
	case reflect.Map:
		// For maps, look up the key
		mapKeyValue := reflect.ValueOf(current)
		mapValue := value.MapIndex(mapKeyValue)
		if !mapValue.IsValid() {
			return nil, fmt.Errorf("map key not found: %s", current)
		}

		if len(rest) == 0 {
			return mapValue.Interface(), nil
		}
		return e.GetValueFromPath(mapValue.Interface(), rest)

	case reflect.Struct:
		// For structs, look up the field
		field := value.FieldByName(current)
		if !field.IsValid() {
			return nil, errors.T(fmt.Errorf("field not found: %s", current))
		}

		if len(rest) == 0 {
			return field.Interface(), nil
		}
		return e.GetValueFromPath(field.Interface(), rest)

	case reflect.Slice, reflect.Array:
		// Special case: if the current path segment is "0", "1", etc., treat as index
		if index, err := parseInt(current); err == nil && index >= 0 && index < value.Len() {
			item := value.Index(index)
			if len(rest) == 0 {
				return item.Interface(), nil
			}
			return e.GetValueFromPath(item.Interface(), rest)
		}

		return nil, errors.T(fmt.Errorf("slice of length %d != %d", len(rest), value.Len()))

	default:
		return nil, errors.T(fmt.Errorf("cannot navigate path %s in %T", current, data))
	}
}

// setValueInPath sets a value in a model using a path
func (e *ExtractorService) setValueInPath(model interface{}, path string, value interface{}) error {
	modelValue := reflect.ValueOf(model)

	// Ensure we're working with a pointer
	if modelValue.Kind() != reflect.Ptr {
		return errors.T(fmt.Errorf("expected pointer model, got %T", model))
	}

	// Dereference pointer
	modelValue = modelValue.Elem()

	// Find the field
	field := modelValue.FieldByName(path)
	if !field.IsValid() {
		return errors.T(fmt.Errorf("field not found: %s", path))
	}

	// Check if the field is settable
	if !field.CanSet() {
		return errors.T(fmt.Errorf("cannot set field %s", path))
	}

	// Convert value to field type if possible
	valueValue := reflect.ValueOf(value)
	if value == nil {
		// Check if the field can be nil
		if field.Kind() == reflect.Chan || field.Kind() == reflect.Func ||
			field.Kind() == reflect.Interface || field.Kind() == reflect.Map ||
			field.Kind() == reflect.Ptr || field.Kind() == reflect.Slice {
			field.Set(reflect.Zero(field.Type()))
			return nil
		}
		return fmt.Errorf("cannot set nil to field %s of kind %s", path, field.Kind())
	}

	// Check if types are directly assignable
	if field.Type().AssignableTo(valueValue.Type()) {
		field.Set(valueValue)
		return nil
	}

	// Try to convert the value to the field's type
	if valueValue.Type().ConvertibleTo(field.Type()) {
		field.Set(valueValue.Convert(field.Type()))
		return nil
	}

	return errors.T(fmt.Errorf("cannot set value of path %s of kind %s", path, field.Kind()))
}

// Helper function to parse a string as an integer
func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}
