package common

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	"gopkg.in/yaml.v3"
)

// OutputFormat represents the output format type
type OutputFormat string

const (
	OutputFormatTable OutputFormat = "table"
	OutputFormatJSON  OutputFormat = "json"
	OutputFormatYAML  OutputFormat = "yaml"
	OutputFormatCSV   OutputFormat = "csv"
)

// ValidateOutputFormat checks if the output format is valid
func ValidateOutputFormat(format string) error {
	switch OutputFormat(format) {
	case OutputFormatTable, OutputFormatJSON, OutputFormatYAML, OutputFormatCSV:
		return nil
	default:
		return fmt.Errorf("invalid output format '%s': must be one of [table, json, yaml, csv]", format)
	}
}

// OutputJSON writes data as JSON
func OutputJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// OutputYAML writes data as YAML
func OutputYAML(data interface{}) error {
	encoder := yaml.NewEncoder(os.Stdout)
	encoder.SetIndent(2)
	defer encoder.Close()
	return encoder.Encode(data)
}

// OutputCSV writes data as CSV using reflection to get field names and values
func OutputCSV(data interface{}) error {
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	// Use reflection to handle slices of structs
	v := reflect.ValueOf(data)
	if v.Kind() != reflect.Slice {
		return fmt.Errorf("CSV output requires a slice of structs")
	}

	if v.Len() == 0 {
		return nil
	}

	// Get the first element to determine headers
	first := v.Index(0)
	if first.Kind() == reflect.Ptr {
		first = first.Elem()
	}
	if first.Kind() != reflect.Struct {
		return fmt.Errorf("CSV output requires a slice of structs")
	}

	// Extract field names for header
	var headers []string
	for i := 0; i < first.NumField(); i++ {
		field := first.Type().Field(i)
		// Use json tag if available, otherwise use field name
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" && jsonTag != "-" {
			headers = append(headers, jsonTag)
		} else {
			headers = append(headers, field.Name)
		}
	}

	// Write header
	if err := writer.Write(headers); err != nil {
		return err
	}

	// Write data rows
	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i)
		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}

		var row []string
		for j := 0; j < elem.NumField(); j++ {
			fieldValue := elem.Field(j)
			row = append(row, fmt.Sprintf("%v", fieldValue.Interface()))
		}

		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// CSVWriter wraps a CSV writer for streaming output
type CSVWriter struct {
	writer         *csv.Writer
	headersWritten bool
}

// NewCSVWriter creates a new streaming CSV writer
func NewCSVWriter() *CSVWriter {
	return &CSVWriter{
		writer:         csv.NewWriter(os.Stdout),
		headersWritten: false,
	}
}

// WriteItems writes items to CSV, writing headers on first call
func (w *CSVWriter) WriteItems(items interface{}) error {
	v := reflect.ValueOf(items)
	if v.Kind() != reflect.Slice {
		return fmt.Errorf("CSV output requires a slice of structs")
	}

	if v.Len() == 0 {
		return nil
	}

	// Get the first element to determine headers
	first := v.Index(0)
	if first.Kind() == reflect.Ptr {
		first = first.Elem()
	}
	if first.Kind() != reflect.Struct {
		return fmt.Errorf("CSV output requires a slice of structs")
	}

	// Write headers if this is the first call
	if !w.headersWritten {
		var headers []string
		for i := 0; i < first.NumField(); i++ {
			field := first.Type().Field(i)
			jsonTag := field.Tag.Get("json")
			if jsonTag != "" && jsonTag != "-" {
				headers = append(headers, jsonTag)
			} else {
				headers = append(headers, field.Name)
			}
		}
		if err := w.writer.Write(headers); err != nil {
			return err
		}
		w.headersWritten = true
	}

	// Write data rows
	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i)
		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}

		var row []string
		for j := 0; j < elem.NumField(); j++ {
			fieldValue := elem.Field(j)
			row = append(row, fmt.Sprintf("%v", fieldValue.Interface()))
		}

		if err := w.writer.Write(row); err != nil {
			return err
		}
	}

	w.writer.Flush()
	return w.writer.Error()
}

// JSONStreamWriter writes items as individual JSON objects (one per line)
type JSONStreamWriter struct {
	encoder *json.Encoder
}

// NewJSONStreamWriter creates a new streaming JSON writer
func NewJSONStreamWriter() *JSONStreamWriter {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return &JSONStreamWriter{encoder: encoder}
}

// WriteItems writes items as JSON objects (one per line)
func (w *JSONStreamWriter) WriteItems(items interface{}) error {
	v := reflect.ValueOf(items)
	if v.Kind() != reflect.Slice {
		return w.encoder.Encode(items)
	}

	for i := 0; i < v.Len(); i++ {
		if err := w.encoder.Encode(v.Index(i).Interface()); err != nil {
			return err
		}
	}
	return nil
}

// YAMLStreamWriter writes items as individual YAML documents
type YAMLStreamWriter struct {
	encoder *yaml.Encoder
}

// NewYAMLStreamWriter creates a new streaming YAML writer
func NewYAMLStreamWriter() *YAMLStreamWriter {
	encoder := yaml.NewEncoder(os.Stdout)
	encoder.SetIndent(2)
	return &YAMLStreamWriter{encoder: encoder}
}

// WriteItems writes items as YAML documents
func (w *YAMLStreamWriter) WriteItems(items interface{}) error {
	v := reflect.ValueOf(items)
	if v.Kind() != reflect.Slice {
		return w.encoder.Encode(items)
	}

	for i := 0; i < v.Len(); i++ {
		if err := w.encoder.Encode(v.Index(i).Interface()); err != nil {
			return err
		}
	}
	return nil
}

// Close closes the YAML encoder
func (w *YAMLStreamWriter) Close() error {
	return w.encoder.Close()
}
