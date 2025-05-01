package parser

import (
	"Luminary/pkg/engine/logger"
	"fmt"
	"reflect"
	"strings"
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
				return nil, fmt.Errorf("required field extraction failed: %w", err)
			}
			e.Logger.Debug("Optional field extraction failed: %v", err)
			continue
		}

		// Apply transformation if provided
		if extractor.Transform != nil && sourceValue != nil {
			sourceValue = extractor.Transform(sourceValue)
		}

		// Set value in target model
		if err := e.setValueInPath(model, extractor.TargetPath, sourceValue); err != nil {
			return nil, fmt.Errorf("failed to set value in target: %w", err)
		}
	}

	return model, nil
}

// ExtractList extracts a list of items using an extractor set
func (e *ExtractorService) ExtractList(extractorSet ExtractorSet, responseData interface{}, listPath []string) ([]interface{}, error) {
	// Extract the list from the response
	listData, err := e.GetValueFromPath(responseData, listPath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract list data: %w", err)
	}

	// Ensure it's a slice
	listValue := reflect.ValueOf(listData)
	if listValue.Kind() != reflect.Slice {
		return nil, fmt.Errorf("expected slice, got %T", listData)
	}

	// Extract each item
	results := make([]interface{}, listValue.Len())
	for i := 0; i < listValue.Len(); i++ {
		item := listValue.Index(i).Interface()
		result, err := e.Extract(extractorSet, item)
		if err != nil {
			e.Logger.Warn("Failed to extract item %d: %v", i, err)
			continue
		}
		results[i] = result
	}

	return results, nil
}

// GetValueFromPath retrieves a value from a nested structure using a path
func (e *ExtractorService) GetValueFromPath(data interface{}, path []string) (interface{}, error) {
	if len(path) == 0 {
		return data, nil
	}

	if data == nil {
		return nil, fmt.Errorf("data is nil")
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
			return nil, fmt.Errorf("struct field not found: %s", current)
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

		// Otherwise, look for objects with matching properties in the array
		// (More complex logic could be implemented here)
		return nil, fmt.Errorf("array/slice access not supported for path: %s", current)

	default:
		return nil, fmt.Errorf("cannot navigate path %s in %T", current, data)
	}
}

// setValueInPath sets a value in a model using a path
func (e *ExtractorService) setValueInPath(model interface{}, path string, value interface{}) error {
	modelValue := reflect.ValueOf(model)

	// Ensure we're working with a pointer
	if modelValue.Kind() != reflect.Ptr {
		return fmt.Errorf("model must be a pointer")
	}

	// Dereference pointer
	modelValue = modelValue.Elem()

	// Find the field
	field := modelValue.FieldByName(path)
	if !field.IsValid() {
		return fmt.Errorf("field not found: %s", path)
	}

	// Check if the field is settable
	if !field.CanSet() {
		return fmt.Errorf("field cannot be set: %s", path)
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

	// Handle special cases for slices, maps, etc.
	// Depending on your domain models, you may need to add more conversions

	return fmt.Errorf("cannot assign %T to field %s of type %s", value, path, field.Type())
}

// Helper function to parse a string as an integer
func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

// CreateExtractor is a helper to create a new extractor
func CreateExtractor(name string, sourcePath []string, targetPath string, transform func(interface{}) interface{}, required bool) Extractor {
	return Extractor{
		Name:       name,
		SourcePath: sourcePath,
		TargetPath: targetPath,
		Transform:  transform,
		Required:   required,
	}
}

// JoinSourcePath is a helper to join source paths for readability
func JoinSourcePath(parts ...string) []string {
	return parts
}

// SplitSourcePath is a helper to split a dot-separated path
func SplitSourcePath(path string) []string {
	return strings.Split(path, ".")
}
