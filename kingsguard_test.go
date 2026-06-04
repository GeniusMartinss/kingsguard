package kingsguard

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestValidateGetRequest(t *testing.T) {
	validName := Field("name").Required().Type("string").Max(10).In("query")
	validSaviour := Field("saviour").Required().Type("string").Regexp("jesus").Max(10).In("query")

	getRequestWithCorrectQueryParams := httptest.NewRequest("GET", "http://google.com?name=martins&saviour=jesus", nil)
	getInvalidRequestWithWrongRegexQueryParams := httptest.NewRequest("GET", "http://google.com?name=martins&saviour=anyoneelse", nil)
	getRequestWithMissingQueryParams := httptest.NewRequest("GET", "http://google.com", nil)
	getInvalidRequestWithLongQueryParams := httptest.NewRequest("GET", "http://google.com?name=martinsdhjjhdhjhjshdjhsjd&saviour=jesus", nil)

	cases := []struct {
		request *http.Request
		schemas []*FieldValidator
		want    bool
	}{
		{getRequestWithCorrectQueryParams, []*FieldValidator{validName, validSaviour}, true},
		{getInvalidRequestWithWrongRegexQueryParams, []*FieldValidator{validName, validSaviour}, false},
		{getRequestWithMissingQueryParams, []*FieldValidator{validName, validSaviour}, false},
		{getInvalidRequestWithLongQueryParams, []*FieldValidator{validName, validSaviour}, false},
	}

	for _, c := range cases {
		got, _ := ValidateRequest(c.request, c.schemas...)
		if got != c.want {
			t.Errorf("Validate(%v) got %v, want %t", c.schemas, got, c.want)
		}
	}
}

func TestValidatePostRequest(t *testing.T) {
	validName := Field("name").Required().Type("string").Min(4).Max(10).In("body")
	validSaviour := Field("saviour").Required().Type("string").Regexp("jesus").Max(10).In("body")
	validBehaviour := Field("behaviour").Required().Type("string").Regexp("love").Min(3).Max(10).In("body")

	goodForm := url.Values{}
	goodForm.Set("name", "martins")
	goodForm.Add("saviour", "jesus")
	goodForm.Add("behaviour", "love")

	postValidRequest := httptest.NewRequest("POST", "http://google.com", strings.NewReader(goodForm.Encode()))
	postValidRequest.PostForm = goodForm

	badForm := url.Values{}
	badForm.Set("name", "mar")
	badForm.Add("saviour", "jesus")
	badForm.Add("behaviour", "love")

	postBadRequestWithMinLength := httptest.NewRequest("POST", "http://google.com", strings.NewReader(badForm.Encode()))
	postBadRequestWithMinLength.PostForm = badForm

	cases := []struct {
		request *http.Request
		schemas []*FieldValidator
		want    bool
	}{
		{postBadRequestWithMinLength, []*FieldValidator{validName, validSaviour, validBehaviour}, false},
		{postValidRequest, []*FieldValidator{validName, validSaviour, validBehaviour}, true},
	}

	for _, c := range cases {
		got, _ := ValidateRequest(c.request, c.schemas...)
		if got != c.want {
			t.Errorf("Validate(%v) got %v, want %t", c.schemas, got, c.want)
		}
	}
}

func TestValidateJsonPostRequest(t *testing.T) {
	validName := Field("name").Required().Type("string").Min(4).Max(10).In("body")
	validSaviour := Field("saviour").Required().Type("string").Regexp("jesus").Max(10).In("body")
	validBehaviour := Field("behaviour").Required().Type("string").Regexp("love").Min(3).Max(10).In("body")

	validPostJson := []byte("{\"name\":\"martins\",\"saviour\":\"jesus\",\"behaviour\":\"love\"}")
	postValidRequest := httptest.NewRequest("POST", "http://google.com", bytes.NewBuffer(validPostJson))
	postValidRequest.Header.Set("Content-Type", "application/json")

	invalidPostJsonWithMissingField := []byte("{\"name\":\"martins\",\"saviour\":\"jesus\"}")
	postInavlidValidRequest := httptest.NewRequest("POST", "http://google.com", bytes.NewBuffer(invalidPostJsonWithMissingField))
	postInavlidValidRequest.Header.Set("Content-Type", "application/json")

	cases := []struct {
		request *http.Request
		schemas []*FieldValidator
		want    bool
	}{
		{postValidRequest, []*FieldValidator{validName, validSaviour, validBehaviour}, true},
		{postInavlidValidRequest, []*FieldValidator{validName, validSaviour, validBehaviour}, false},
	}

	for _, c := range cases {
		got, _ := ValidateRequest(c.request, c.schemas...)
		if got != c.want {
			t.Errorf("Validate(%v) got %v, want %t", c.schemas, got, c.want)
		}
	}
}
