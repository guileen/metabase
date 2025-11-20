package common

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// ValidationRule represents a validation rule
type ValidationRule struct {
	Name    string
	Message string
	Validate func(interface{}) bool
}

// Validator provides input validation functionality
type Validator struct {
	rules map[string][]ValidationRule
}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{
		rules: make(map[string][]ValidationRule),
	}
}

// AddRule adds a validation rule for a field
func (v *Validator) AddRule(field string, rule ValidationRule) *Validator {
	if v.rules[field] == nil {
		v.rules[field] = make([]ValidationRule, 0)
	}
	v.rules[field] = append(v.rules[field], rule)
	return v
}

// Required adds a required field validation
func (v *Validator) Required(field string) *Validator {
	return v.AddRule(field, ValidationRule{
		Name:    "required",
		Message: fmt.Sprintf("%s is required", field),
		Validate: func(value interface{}) bool {
			if value == nil {
				return false
			}
			if str, ok := value.(string); ok {
				return strings.TrimSpace(str) != ""
			}
			return true
		},
	})
}

// MaxLength adds a max length validation
func (v *Validator) MaxLength(field string, max int) *Validator {
	return v.AddRule(field, ValidationRule{
		Name:    "max_length",
		Message: fmt.Sprintf("%s must be at most %d characters", field, max),
		Validate: func(value interface{}) bool {
			if str, ok := value.(string); ok {
				return utf8.RuneCountInString(str) <= max
			}
			return true
		},
	})
}

// MinLength adds a min length validation
func (v *Validator) MinLength(field string, min int) *Validator {
	return v.AddRule(field, ValidationRule{
		Name:    "min_length",
		Message: fmt.Sprintf("%s must be at least %d characters", field, min),
		Validate: func(value interface{}) bool {
			if str, ok := value.(string); ok {
				return utf8.RuneCountInString(str) >= min
			}
			return true
		},
	})
}

// Email adds an email format validation
func (v *Validator) Email(field string) *Validator {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return v.AddRule(field, ValidationRule{
		Name:    "email",
		Message: fmt.Sprintf("%s must be a valid email address", field),
		Validate: func(value interface{}) bool {
			if str, ok := value.(string); ok {
				return emailRegex.MatchString(str)
			}
			return true
		},
	})
}

// URL adds a URL format validation
func (v *Validator) URL(field string) *Validator {
	return v.AddRule(field, ValidationRule{
		Name:    "url",
		Message: fmt.Sprintf("%s must be a valid URL", field),
		Validate: func(value interface{}) bool {
			if str, ok := value.(string); ok {
				_, err := url.ParseRequestURI(str)
				return err == nil
			}
			return true
		},
	})
}

// Numeric adds a numeric validation
func (v *Validator) Numeric(field string) *Validator {
	return v.AddRule(field, ValidationRule{
		Name:    "numeric",
		Message: fmt.Sprintf("%s must be a number", field),
		Validate: func(value interface{}) bool {
			switch v := value.(type) {
			case int, int32, int64, float32, float64:
				return true
			case string:
				_, err := strconv.ParseFloat(v, 64)
				return err == nil
			default:
				return false
			}
		},
	})
}

// Range adds a range validation for numbers
func (v *Validator) Range(field string, min, max float64) *Validator {
	return v.AddRule(field, ValidationRule{
		Name:    "range",
		Message: fmt.Sprintf("%s must be between %.2f and %.2f", field, min, max),
		Validate: func(value interface{}) bool {
			var num float64
			var err error

			switch v := value.(type) {
			case int:
				num = float64(v)
			case int32:
				num = float64(v)
			case int64:
				num = float64(v)
			case float32:
				num = float64(v)
			case float64:
				num = v
			case string:
				num, err = strconv.ParseFloat(v, 64)
				if err != nil {
					return false
				}
			default:
				return false
			}

			return num >= min && num <= max
		},
	})
}

// InList adds an "in list" validation
func (v *Validator) InList(field string, allowedValues []interface{}) *Validator {
	return v.AddRule(field, ValidationRule{
		Name:    "in_list",
		Message: fmt.Sprintf("%s must be one of: %v", field, allowedValues),
		Validate: func(value interface{}) bool {
			for _, allowed := range allowedValues {
				if value == allowed {
					return true
				}
			}
			return false
		},
	})
}

// Validate validates the provided data against all rules
func (v *Validator) Validate(data map[string]interface{}) *AppError {
	var errors []string

	for field, rules := range v.rules {
		value := data[field]

		for _, rule := range rules {
			if !rule.Validate(value) {
				errors = append(errors, rule.Message)
			}
		}
	}

	if len(errors) > 0 {
		return NewInvalidInputError("Validation failed").WithDetail("errors", errors)
	}

	return nil
}

// ValidationMiddleware provides HTTP middleware for validation
type ValidationMiddleware struct {
	validator *Validator
}

// NewValidationMiddleware creates validation middleware
func NewValidationMiddleware(validator *Validator) *ValidationMiddleware {
	return &ValidationMiddleware{
		validator: validator,
	}
}

// ValidateJSON validates JSON request body
func ValidateJSON(validator *Validator) func(map[string]interface{}) *AppError {
	return func(data map[string]interface{}) *AppError {
		return validator.Validate(data)
	}
}

// SanitizeInput provides basic input sanitization
func SanitizeInput(input string) string {
	// Basic HTML sanitization
	input = strings.ReplaceAll(input, "<", "&lt;")
	input = strings.ReplaceAll(input, ">", "&gt;")
	input = strings.ReplaceAll(input, "\"", "&quot;")
	input = strings.ReplaceAll(input, "'", "&#x27;")
	input = strings.ReplaceAll(input, "/", "&#x2F;")

	// Trim whitespace
	input = strings.TrimSpace(input)

	return input
}

// ValidateJSONSize checks JSON payload size
func ValidateJSONSize(data []byte, maxSize int64) *AppError {
	if int64(len(data)) > maxSize {
		return NewInvalidInputError(fmt.Sprintf("JSON payload too large. Max size: %d bytes", maxSize))
	}
	return nil
}

// ParseAndValidateJSON parses and validates JSON input
func ParseAndValidateJSON(jsonData []byte, validator *Validator, maxSize int64) (map[string]interface{}, *AppError) {
	// Check size first
	if err := ValidateJSONSize(jsonData, maxSize); err != nil {
		return nil, err
	}

	// Parse JSON
	var data map[string]interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, NewInvalidInputError("Invalid JSON format").WithCause(err)
	}

	// Sanitize string values
	for key, value := range data {
		if str, ok := value.(string); ok {
			data[key] = SanitizeInput(str)
		}
	}

	// Validate
	if err := validator.Validate(data); err != nil {
		return nil, err
	}

	return data, nil
}

// Common validators
var (
	CommonValidators = struct {
		ID        *Validator
		User      *Validator
		TableName *Validator
		Query     *Validator
	}{
		ID: NewValidator().
			Required("id").
			MaxLength("id", 100),

		User: NewValidator().
			Required("name").
			Required("email").
			MaxLength("name", 100).
			MaxLength("email", 255).
			Email("email"),

		TableName: NewValidator().
			Required("table").
			MaxLength("table", 64).
			InList("table", []interface{}{"users", "posts", "comments", "settings"}),

		Query: NewValidator().
			MaxLength("query", 1000).
			Range("limit", 1, 1000).
			Range("offset", 0, 10000),
	}
)