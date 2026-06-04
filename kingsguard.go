package kingsguard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// FieldValidator is a fluent builder for describing a single request-field validation rule.
// Use the Field constructor to create an instance and the chainable methods to configure it.
type FieldValidator struct {
	fieldName     string
	required      bool
	datatype      string
	regexpPattern string
	min           int
	max           int
	paramType     string
}

// Field creates a new FieldValidator for the named request field.
// min and max are initialised to -1 (sentinel meaning "no limit applied") because
// the Go zero value 0 would incorrectly activate min/max validation immediately.
func Field(name string) *FieldValidator {
	return &FieldValidator{
		fieldName: name,
		min:       -1,
		max:       -1,
	}
}

// Required marks the field as mandatory in the request.
func (f *FieldValidator) Required() *FieldValidator {
	f.required = true
	return f
}

// Type sets the expected data type for the field value.
// The type string is normalised to lower-case so that e.g. "Float" and "float" are equivalent.
func (f *FieldValidator) Type(t string) *FieldValidator {
	f.datatype = strings.ToLower(t)
	return f
}

// Regexp sets a regular-expression pattern that the field value must match.
func (f *FieldValidator) Regexp(pattern string) *FieldValidator {
	f.regexpPattern = pattern
	return f
}

// Min sets the minimum accepted length (for strings) or value (for ints).
func (f *FieldValidator) Min(n int) *FieldValidator {
	f.min = n
	return f
}

// Max sets the maximum accepted length (for strings) or value (for ints).
func (f *FieldValidator) Max(n int) *FieldValidator {
	f.max = n
	return f
}

// In sets where in the request the field should be looked up ("query" or "body").
func (f *FieldValidator) In(paramType string) *FieldValidator {
	f.paramType = paramType
	return f
}

// ValidateRequest validates an incoming request against the provided field schemas.
// It returns true and a nil error when the request is valid, or false and a descriptive
// error when a validation rule is violated.
func ValidateRequest(r *http.Request, schemas ...*FieldValidator) (bool, error) {
	for _, schema := range schemas {
		if schema.required {
			if !isrequiredFieldPresent(r, schema.fieldName, schema.paramType) {
				return false, fmt.Errorf("%s is a required field", schema.fieldName)
			}
		} else {
			if !isrequiredFieldPresent(r, schema.fieldName, schema.paramType) {
				//don't continue validation if field is not required and also not present in request
				return true, nil
			}
		}
		if ok, err := isDataTypeCorrect(r, schema); !ok {
			return false, err
		}
		if schema.regexpPattern != "" {
			if !isRegexMatching(r, schema) {
				return false, fmt.Errorf("%s does not match required pattern", schema.fieldName)
			}
		}
		if schema.min != -1 && schema.datatype != "bool" {
			if !isMinCorrect(r, schema) {
				return false, fmt.Errorf("the minimum accepted length/value for %s is %d", schema.fieldName, schema.min)
			}
		}
		if schema.max != -1 && schema.datatype != "bool" {
			if !isMaxCorrect(r, schema) {
				return false, fmt.Errorf("the maximum accepted length/value for %s is %d", schema.fieldName, schema.max)
			}
		}
	}

	return true, nil
}

// readBody reads the entire request body, restores r.Body so subsequent reads
// within the same ValidateRequest loop receive a non-empty stream, and returns
// the raw bytes. Any read error is propagated to the caller.
func readBody(r *http.Request) ([]byte, error) {
	buf, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}
	r.Body = io.NopCloser(bytes.NewBuffer(buf))
	return buf, nil
}

func isrequiredFieldPresent(r *http.Request, field string, paramType string) bool {
	switch paramType {
	case "query":
		if _, ok := r.URL.Query()[field]; !ok {
			return false
		}
	case "body":
		buf, err := readBody(r)
		if err != nil {
			return false
		}
		bodyData := io.NopCloser(bytes.NewBuffer(buf))
		switch r.Header.Get("Content-type") {
		case "application/json":
			requestBody := make(map[string]interface{})
			if err := json.NewDecoder(bodyData).Decode(&requestBody); err != nil {
				return false
			}
			if value := requestBody[field]; value == nil {
				return false
			}
		default:
			if err := r.FormValue(field); err == "" {
				return false
			}
		}
	}
	return true
}

// isDataTypeCorrect checks that the value for schema.fieldName in the request
// matches the declared schema.datatype. It returns (false, error) on type
// mismatch and (true, nil) when the value is valid.
//
// For JSON bodies, json.Unmarshal decodes all JSON numbers as float64 — so both
// "int" and "float" validations accept a float64 Go value as a valid representation.
func isDataTypeCorrect(r *http.Request, schema *FieldValidator) (bool, error) {
	switch schema.paramType {
	case "query":
		val, ok := r.URL.Query()[schema.fieldName]
		if !ok {
			return false, fmt.Errorf("%s must be of type %s", schema.fieldName, schema.datatype)
		}
		switch schema.datatype {
		case "int":
			if _, err := strconv.ParseInt(val[0], 10, 64); err != nil {
				return false, fmt.Errorf("%s must be of type %s", schema.fieldName, schema.datatype)
			}
		case "bool":
			if _, err := strconv.ParseBool(val[0]); err != nil {
				return false, fmt.Errorf("%s must be of type %s", schema.fieldName, schema.datatype)
			}
		case "float":
			if _, err := strconv.ParseFloat(val[0], 64); err != nil {
				return false, fmt.Errorf("%s must be of type %s", schema.fieldName, schema.datatype)
			}
		case "string":
			return true, nil
		}

	case "body":
		buf, err := readBody(r)
		if err != nil {
			return false, fmt.Errorf("%s must be of type %s", schema.fieldName, schema.datatype)
		}
		bodyData := io.NopCloser(bytes.NewBuffer(buf))
		switch r.Header.Get("Content-type") {
		case "application/json":
			requestBody := make(map[string]interface{})
			if err := json.NewDecoder(bodyData).Decode(&requestBody); err != nil {
				return false, fmt.Errorf("%s must be of type %s", schema.fieldName, schema.datatype)
			}
			val := requestBody[schema.fieldName]

			// Guard against absent keys: a nil value must not reach the type switch.
			if val == nil {
				return false, fmt.Errorf("%s must be of type %s", schema.fieldName, schema.datatype)
			}

			switch schema.datatype {
			case "string":
				if _, ok := val.(string); !ok {
					return false, fmt.Errorf("%s must be of type %s", schema.fieldName, schema.datatype)
				}
			case "int":
				switch val.(type) {
				case float64:
					// json.Unmarshal decodes all JSON numbers as float64 — accept as valid int representation.
				case string:
					if _, err := strconv.ParseInt(val.(string), 10, 64); err != nil {
						return false, fmt.Errorf("%s must be of type %s", schema.fieldName, schema.datatype)
					}
				default:
					return false, fmt.Errorf("%s must be of type %s", schema.fieldName, schema.datatype)
				}
			case "float":
				switch val.(type) {
				case float64:
					// already the correct Go type for JSON numbers.
				case string:
					if _, err := strconv.ParseFloat(val.(string), 64); err != nil {
						return false, fmt.Errorf("%s must be of type %s", schema.fieldName, schema.datatype)
					}
				default:
					return false, fmt.Errorf("%s must be of type %s", schema.fieldName, schema.datatype)
				}
			case "bool":
				switch val.(type) {
				case bool:
					// already correct Go type for JSON booleans.
				case string:
					if _, err := strconv.ParseBool(val.(string)); err != nil {
						return false, fmt.Errorf("%s must be of type %s", schema.fieldName, schema.datatype)
					}
				default:
					return false, fmt.Errorf("%s must be of type %s", schema.fieldName, schema.datatype)
				}
			}

		default:
			if value := r.FormValue(schema.fieldName); len(value) > 0 {
				switch schema.datatype {
				case "int":
					if _, err := strconv.ParseInt(value, 10, 64); err != nil {
						return false, fmt.Errorf("%s must be of type %s", schema.fieldName, schema.datatype)
					}
				case "bool":
					if _, err := strconv.ParseBool(value); err != nil {
						return false, fmt.Errorf("%s must be of type %s", schema.fieldName, schema.datatype)
					}
				case "float":
					if _, err := strconv.ParseFloat(value, 64); err != nil {
						return false, fmt.Errorf("%s must be of type %s", schema.fieldName, schema.datatype)
					}
				case "string":
					return true, nil
				}
			}
		}
	}
	return true, nil
}

func isRegexMatching(r *http.Request, schema *FieldValidator) bool {
	exp, err := regexp.Compile(schema.regexpPattern)
	if err != nil {
		return false
	}
	switch schema.paramType {
	case "query":
		val, ok := r.URL.Query()[schema.fieldName]
		if !ok {
			return false
		}
		if !exp.MatchString(val[0]) {
			return false
		}
	case "body":
		buf, err := readBody(r)
		if err != nil {
			return false
		}
		bodyData := io.NopCloser(bytes.NewBuffer(buf))
		switch r.Header.Get("Content-type") {
		case "application/json":
			requestBody := make(map[string]interface{})
			if err := json.NewDecoder(bodyData).Decode(&requestBody); err != nil {
				return false
			}
			val := requestBody[schema.fieldName]
			if val == nil {
				return false
			}
			// Use fmt.Sprintf("%v", val) for safe string coercion — avoids panic on
			// non-string types (float64, bool, etc.) that json.Unmarshal may produce.
			if !exp.MatchString(fmt.Sprintf("%v", val)) {
				return false
			}
		default:
			if value := r.FormValue(schema.fieldName); !exp.MatchString(value) {
				return false
			}
		}
	}
	return true
}

func isMinCorrect(r *http.Request, schema *FieldValidator) bool {
	switch schema.paramType {
	case "query":
		val, ok := r.URL.Query()[schema.fieldName]
		if !ok {
			return false
		}
		if schema.datatype == "string" {
			if len(val[0]) < schema.min {
				return false
			}
		} else if schema.datatype == "int" {
			if value, _ := strconv.ParseInt(val[0], 10, 64); int(value) < schema.min {
				return false
			}
		}
	case "body":
		buf, err := readBody(r)
		if err != nil {
			return false
		}
		bodyData := io.NopCloser(bytes.NewBuffer(buf))
		switch r.Header.Get("Content-type") {
		case "application/json":
			requestBody := make(map[string]interface{})
			if err := json.NewDecoder(bodyData).Decode(&requestBody); err != nil {
				return false
			}
			val := requestBody[schema.fieldName]
			if val == nil {
				return false
			}
			if schema.datatype == "string" {
				// Use fmt.Sprintf("%v", val) for safe string coercion.
				if len(fmt.Sprintf("%v", val)) < schema.min {
					return false
				}
			} else if schema.datatype == "int" {
				// json.Unmarshal produces float64 for JSON numbers; compare directly.
				switch v := val.(type) {
				case float64:
					if int(v) < schema.min {
						return false
					}
				case string:
					if value, _ := strconv.ParseInt(v, 10, 64); int(value) < schema.min {
						return false
					}
				}
			}
		default:
			if schema.datatype == "string" {
				if value := r.FormValue(schema.fieldName); len(value) < schema.min {
					return false
				}
			} else if schema.datatype == "int" {
				if value, _ := strconv.ParseInt(r.FormValue(schema.fieldName), 10, 64); int(value) < schema.min {
					return false
				}
			}
		}
	}
	return true
}

func isMaxCorrect(r *http.Request, schema *FieldValidator) bool {
	switch schema.paramType {
	case "query":
		val, ok := r.URL.Query()[schema.fieldName]
		if !ok {
			return false
		}
		if schema.datatype == "string" {
			if len(val[0]) > schema.max {
				return false
			}
		} else if schema.datatype == "int" {
			if value, _ := strconv.ParseInt(val[0], 10, 64); int(value) > schema.max {
				return false
			}
		}
	case "body":
		buf, err := readBody(r)
		if err != nil {
			return false
		}
		bodyData := io.NopCloser(bytes.NewBuffer(buf))
		switch r.Header.Get("Content-type") {
		case "application/json":
			requestBody := make(map[string]interface{})
			if err := json.NewDecoder(bodyData).Decode(&requestBody); err != nil {
				return false
			}
			val := requestBody[schema.fieldName]
			if val == nil {
				return false
			}
			if schema.datatype == "string" {
				// Use fmt.Sprintf("%v", val) for safe string coercion.
				if len(fmt.Sprintf("%v", val)) > schema.max {
					return false
				}
			} else if schema.datatype == "int" {
				// json.Unmarshal produces float64 for JSON numbers; compare directly.
				switch v := val.(type) {
				case float64:
					if int(v) > schema.max {
						return false
					}
				case string:
					if value, _ := strconv.ParseInt(v, 10, 64); int(value) > schema.max {
						return false
					}
				}
			}
		default:
			if schema.datatype == "string" {
				if value := r.FormValue(schema.fieldName); len(value) > schema.max {
					return false
				}
			} else if schema.datatype == "int" {
				if value, _ := strconv.ParseInt(r.FormValue(schema.fieldName), 10, 64); int(value) > schema.max {
					return false
				}
			}
		}
	}
	return true
}
