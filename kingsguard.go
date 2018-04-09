package kingsguard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
)

type Lannister struct {
	Field     string
	Required  bool
	Datatype  string
	Regexp    string
	Min int
	Max int
	ParamType string
}

// ValidateRequest validates an incoming request that it matches a specified schema
// it takes in a http request and an array of scehmas of the form lannisters
// it returns a boolean true if the request is valid. and false otherwise
// it returns an error nil for valid requests and an appropriate error for bad requests
func ValidateRequest(r *http.Request, schemas ...Lannister) (bool, error) {
	for _, schema := range schemas {
		if schema.Required == true {
			if !isrequiredFieldPresent(r, schema.Field, schema.ParamType) {
				return false, fmt.Errorf("%s is a required field", schema.Field)
			}
		} else {
			if !isrequiredFieldPresent(r, schema.Field, schema.ParamType) {
				//don't continue validation if field is not required and also not present in request
				return true, nil
			}
		}
		if !isDataTypeCorrect(r, &schema) {
			return false, fmt.Errorf("%s must be of type %s", schema.Field, schema.Datatype)
		}
		if schema.Regexp != "" {
			if !isRegexMatching(r, &schema) {
				return false, fmt.Errorf("%s does not match required pattern", schema.Field)
			}
		}
		if schema.Min != -1 && schema.Datatype != "bool" {
			if !isMinCorrect(r, &schema) {
				return false, fmt.Errorf("the minimum accepted length/value for %s is %d", schema.Field, schema.Min)
			}
		}
		if schema.Max != -1 && schema.Datatype != "bool" {
			if !isMaxCorrect(r, &schema) {
				return false, fmt.Errorf("the maximum accepted length/value for %s is %d", schema.Field, schema.Max)
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

func isDataTypeCorrect(r *http.Request, schema *Lannister) bool {
	switch schema.ParamType {
	case "query":
		val, ok := r.URL.Query()[schema.Field]
		if !ok {
			return false
		} else {
			switch schema.Datatype {
			case "int":
				if _, err := strconv.ParseInt(val[0], 10, 64); err != nil {
					return false
				}
			case "bool":
				if _, err := strconv.ParseBool(val[0]); err != nil {
					return false
				}
			case "Float":
				if _, err := strconv.ParseFloat(val[0], 64); err != nil {
					return false
				}
			case "String":
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
			val := requestBody[schema.Field]

			if val.(string) == "" {
				return false
			} else {
				switch schema.Datatype {
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
			if value := r.FormValue(schema.Field); len(value) > 0 {
				switch schema.Datatype {
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

func isRegexMatching(r *http.Request, schema *Lannister) bool {
	exp, err := regexp.Compile(schema.Regexp)
	if err != nil {
		return false
	}
	switch schema.ParamType {
	case "query":
		val, ok := r.URL.Query()[schema.Field]
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
			val := requestBody[schema.Field]
			if val == "" {
				return false
			} else {
				if !exp.MatchString(val.(string)) {
					return false
				}
			}
		default:
			if value := r.FormValue(schema.Field); !exp.MatchString(value) {
				return false
			}
		}
	}
	return true
}

func isMinCorrect(r *http.Request, schema *Lannister) bool {
	switch schema.ParamType {
	case "query":
		val, ok := r.URL.Query()[schema.Field]
		if !ok {
			return false
		} else {
			if schema.Datatype == "string" {
				if len(val[0]) < schema.Min {
					return false
				}
			}else if schema.Datatype == "int"{
				if value,_ := strconv.ParseInt(val[0], 10, 64); int(value) < schema.Min {
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
			val := requestBody[schema.Field]
			if val == "" {
				return false
			} else {
				if schema.Datatype == "string" {
					if len(val.(string)) < schema.Min {
						return false
					}
				}else if schema.Datatype == "int"{
					if value,_ := strconv.ParseInt(val.(string), 10, 64); int(value) < schema.Min {
						return false
					}
				}
			}
		default:
			if schema.Datatype == "string" {
				if value := r.FormValue(schema.Field); len(value) < schema.Min {
					return false
				}
			}else if schema.Datatype == "int"{
				if value,_ := strconv.ParseInt(r.FormValue(schema.Field), 10, 64); int(value) < schema.Min  {
					return false
				}
			}

		}
	}
	return true
}

func isMaxCorrect(r *http.Request, schema *Lannister) bool {
	switch schema.ParamType {
	case "query":
		val, ok := r.URL.Query()[schema.Field]
		if !ok {
			return false
		} else {
			if schema.Datatype == "string" {
				if len(val[0]) > schema.Max {
					return false
				}
			}else if schema.Datatype == "int"{
				if value,_ := strconv.ParseInt(val[0], 10, 64); int(value) > schema.Max {
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
			val := requestBody[schema.Field]
			if val == "" {
				return false
			} else {
				if schema.Datatype == "string" {
					if len(val.(string)) > schema.Max {
						return false
					}
				}else if schema.Datatype == "int"{
					if value,_ := strconv.ParseInt(val.(string), 10, 64); int(value) > schema.Max {
						return false
					}
				}
			}
		default:
			if schema.Datatype == "string" {
				if value := r.FormValue(schema.Field); len(value) > schema.Max {
					return false
				}
			}else if schema.Datatype == "int"{
				if value,_ := strconv.ParseInt(r.FormValue(schema.Field), 10, 64); int(value) > schema.Max {
					return false
				}
			}
		}
	}
	return true
}
