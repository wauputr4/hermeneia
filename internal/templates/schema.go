package templates

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

type inputSchema struct {
	Type       string                 `json:"type"`
	Required   []string               `json:"required"`
	Properties map[string]inputSchema `json:"properties"`
	Items      *inputSchema           `json:"items"`
	Const      any                    `json:"const"`
	Enum       []any                  `json:"enum"`
	MinItems   *int                   `json:"minItems"`
	MaxItems   *int                   `json:"maxItems"`
	Minimum    *float64               `json:"minimum"`
}

func ValidateInput(manifest Manifest, input any) error {
	if len(manifest.InputSchema) == 0 {
		return nil
	}
	var schema inputSchema
	dec := json.NewDecoder(bytes.NewReader(manifest.InputSchema))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&schema); err != nil {
		return fmt.Errorf("template %q input_schema is unsupported: %w", manifest.ID, err)
	}
	data, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("marshal template input for %q: %w", manifest.ID, err)
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("decode template input for %q: %w", manifest.ID, err)
	}
	if err := validateValue(schema, value, "$"); err != nil {
		return fmt.Errorf("template %q input validation failed: %w", manifest.ID, err)
	}
	return nil
}

func validateValue(schema inputSchema, value any, path string) error {
	if schema.Type != "" && !matchesType(schema.Type, value) {
		return fmt.Errorf("%s must be %s", path, schema.Type)
	}
	if schema.Const != nil && !jsonEqual(schema.Const, value) {
		return fmt.Errorf("%s must equal %v", path, schema.Const)
	}
	if len(schema.Enum) > 0 && !containsJSONValue(schema.Enum, value) {
		return fmt.Errorf("%s must be one of %s", path, enumValues(schema.Enum))
	}

	switch typed := value.(type) {
	case map[string]any:
		for _, field := range schema.Required {
			if _, ok := typed[field]; !ok {
				return fmt.Errorf("%s.%s is required", path, field)
			}
		}
		for field, child := range schema.Properties {
			childValue, ok := typed[field]
			if !ok {
				continue
			}
			if err := validateValue(child, childValue, path+"."+field); err != nil {
				return err
			}
		}
	case []any:
		if schema.MinItems != nil && len(typed) < *schema.MinItems {
			return fmt.Errorf("%s must contain at least %d item(s)", path, *schema.MinItems)
		}
		if schema.MaxItems != nil && len(typed) > *schema.MaxItems {
			return fmt.Errorf("%s must contain at most %d item(s)", path, *schema.MaxItems)
		}
		if schema.Items != nil {
			for i, item := range typed {
				if err := validateValue(*schema.Items, item, fmt.Sprintf("%s[%d]", path, i)); err != nil {
					return err
				}
			}
		}
	case float64:
		if schema.Minimum != nil && typed < *schema.Minimum {
			return fmt.Errorf("%s must be at least %s", path, formatNumber(*schema.Minimum))
		}
	}
	return nil
}

func matchesType(name string, value any) bool {
	switch name {
	case "object":
		_, ok := value.(map[string]any)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "integer":
		number, ok := value.(float64)
		return ok && math.Trunc(number) == number
	case "number":
		_, ok := value.(float64)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	default:
		return false
	}
}

func containsJSONValue(values []any, value any) bool {
	for _, candidate := range values {
		if jsonEqual(candidate, value) {
			return true
		}
	}
	return false
}

func jsonEqual(a, b any) bool {
	left, err := json.Marshal(a)
	if err != nil {
		return false
	}
	right, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return bytes.Equal(left, right)
}

func enumValues(values []any) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprint(value))
	}
	return strings.Join(parts, ", ")
}

func formatNumber(value float64) string {
	if math.Trunc(value) == value {
		return fmt.Sprintf("%.0f", value)
	}
	return fmt.Sprint(value)
}
