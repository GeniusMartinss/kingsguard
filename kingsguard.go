package kingsguard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
		if schema.required == true {
			if !isrequiredFieldPresent(r, schema.fieldName, schema.paramType) {
				return false, fmt.Errorf("%s is a required field", schema.fieldName)
			}
		} else {
			if !isrequiredFieldPresent(r, schema.fieldName, schema.paramType) {
				//don't continue validation if field is not required and also not present in request
				return true, nil
			}
		}
		if !isDataTypeCorrect(r, schema) {
			return false, fmt.Errorf("%s must be of type %s", schema.fieldName, schema.datatype)
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

func isrequiredFieldPresent(r *http.Request, field string, paramType string) bool {
	switch paramType {
	case "query":
		if _, ok := r.URL.Query()[field]; !ok {
			return false
		}
	case "body":
		buf, _ := ioutil.ReadAll(r.Body)
		bodyData := ioutil.NopCloser(bytes.NewBuffer(buf))
		nextData := ioutil.NopCloser(bytes.NewBuffer(buf))
		r.Body = nextData
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

func isDataTypeCorrect(r *http.Request, schema *FieldValidator) bool {
	switch schema.paramType {
	case "query":
		val, ok := r.URL.Query()[schema.fieldName]
		if !ok {
			return false
		} else {
			switch schema.datatype {
			case "int":
				if _, err := strconv.ParseInt(val[0], 10, 64); err != nil {
					return false
				}
			case "bool":
				if _, err := strconv.ParseBool(val[0]); err != nil {
					return false
				}
			case "float":
				if _, err := strconv.ParseFloat(val[0], 64); err != nil {
					return false
				}
			case "string":
				return true
			}
		}
	case "body":
		buf, _ := ioutil.ReadAll(r.Body)
		bodyData := ioutil.NopCloser(bytes.NewBuffer(buf))
		nextData := ioutil.NopCloser(bytes.NewBuffer(buf))
		r.Body = nextData
		switch r.Header.Get("Content-type") {
		case "application/json":
			requestBody := make(map[string]interface{})
			if err := json.NewDecoder(bodyData).Decode(&requestBody); err != nil {
				return false
			}
			val := requestBody[schema.fieldName]

			if val.(string) == "" {
				return false
			} else {
				switch schema.datatype {
				case "int":
					if _, err := strconv.ParseInt(val.(string), 10, 64); err != nil {
						return false
					}
				case "bool":
					if _, err := strconv.ParseBool(val.(string)); err != nil {
						return false
					}
				case "float":
					if _, err := strconv.ParseFloat(val.(string), 64); err != nil {
						return false
					}
				case "string":
					return true
				}
			}
		default:
			if value := r.FormValue(schema.fieldName); len(value) > 0 {
				switch schema.datatype {
				case "int":
					if _, err := strconv.ParseInt(value, 10, 64); err != nil {
						return false
					}
				case "bool":
					if _, err := strconv.ParseBool(value); err != nil {
						return false
					}
				case "float":
					if _, err := strconv.ParseFloat(value, 64); err != nil {
						return false
					}
				case "string":
					return true
				}
			}
		}
	}
	return true
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
		} else {
			if !exp.MatchString(val[0]) {
				return false
			}
		}
	case "body":
		buf, _ := ioutil.ReadAll(r.Body)
		bodyData := ioutil.NopCloser(bytes.NewBuffer(buf))
		nextData := ioutil.NopCloser(bytes.NewBuffer(buf))
		r.Body = nextData
		switch r.Header.Get("Content-type") {
		case "application/json":
			requestBody := make(map[string]interface{})
			if err := json.NewDecoder(bodyData).Decode(&requestBody); err != nil {
				return false
			}
			val := requestBody[schema.fieldName]
			if val == "" {
				return false
			} else {
				if !exp.MatchString(val.(string)) {
					return false
				}
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
		} else {
			if schema.datatype == "string" {
				if len(val[0]) < schema.min {
					return false
				}
			} else if schema.datatype == "int" {
				if value, _ := strconv.ParseInt(val[0], 10, 64); int(value) < schema.min {
					return false
				}
			}
		}
	case "body":
		buf, _ := ioutil.ReadAll(r.Body)
		bodyData := ioutil.NopCloser(bytes.NewBuffer(buf))
		nextData := ioutil.NopCloser(bytes.NewBuffer(buf))
		r.Body = nextData
		switch r.Header.Get("Content-type") {
		case "application/json":
			requestBody := make(map[string]interface{})
			if err := json.NewDecoder(bodyData).Decode(&requestBody); err != nil {
				return false
			}
			val := requestBody[schema.fieldName]
			if val == "" {
				return false
			} else {
				if schema.datatype == "string" {
					if len(val.(string)) < schema.min {
						return false
					}
				} else if schema.datatype == "int" {
					if value, _ := strconv.ParseInt(val.(string), 10, 64); int(value) < schema.min {
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
		} else {
			if schema.datatype == "string" {
				if len(val[0]) > schema.max {
					return false
				}
			} else if schema.datatype == "int" {
				if value, _ := strconv.ParseInt(val[0], 10, 64); int(value) > schema.max {
					return false
				}
			}
		}
	case "body":
		buf, _ := ioutil.ReadAll(r.Body)
		bodyData := ioutil.NopCloser(bytes.NewBuffer(buf))
		nextData := ioutil.NopCloser(bytes.NewBuffer(buf))
		r.Body = nextData
		switch r.Header.Get("Content-type") {
		case "application/json":
			requestBody := make(map[string]interface{})
			if err := json.NewDecoder(bodyData).Decode(&requestBody); err != nil {
				return false
			}
			val := requestBody[schema.fieldName]
			if val == "" {
				return false
			} else {
				if schema.datatype == "string" {
					if len(val.(string)) > schema.max {
						return false
					}
				} else if schema.datatype == "int" {
					if value, _ := strconv.ParseInt(val.(string), 10, 64); int(value) > schema.max {
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
