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
	validName := Lannister{
		"name",
		true,
		"string",
		"",
		-1,
		10,
		"query",
	}
	validSaviour := Lannister{
		"saviour",
		true,
		"string",
		"jesus",
		-1,
		10,
		"query",
	}
	getRequestWithCorrectQueryParams := httptest.NewRequest("GET", "http://google.com?name=martins&saviour=jesus", nil)
	getInvalidRequestWithWrongRegexQueryParams := httptest.NewRequest("GET", "http://google.com?name=martins&saviour=anyoneelse", nil)
	getRequestWithMissingQueryParams := httptest.NewRequest("GET", "http://google.com", nil)
	getInvalidRequestWithLongQueryParams := httptest.NewRequest("GET", "http://google.com?name=martinsdhjjhdhjhjshdjhsjd&saviour=jesus", nil)

	cases := []struct {
		request *http.Request
		schemas []Lannister
		want    bool
	}{
		{getRequestWithCorrectQueryParams, []Lannister{validName, validSaviour}, true},
		{getInvalidRequestWithWrongRegexQueryParams, []Lannister{validName, validSaviour}, false},
		{getRequestWithMissingQueryParams, []Lannister{validName, validSaviour}, false},
		{getInvalidRequestWithLongQueryParams, []Lannister{validName, validSaviour}, false},
	}

	for _, c := range cases {
		got, _ := ValidateRequest(c.request, c.schemas...)
		if got != c.want {
			t.Errorf("Validate(%v) got %v, want %t", c.schemas, got, c.want)
		}
	}
}

func TestValidatePostRequest(t *testing.T) {
	validName := Lannister{
		"name",
		true,
		"string",
		"",
		4,
		10,
		"body",
	}
	validSaviour := Lannister{
		"saviour",
		true,
		"string",
		"jesus",
		-1,
		10,
		"body",
	}
	validBehaviour := Lannister{
		"behaviour",
		true,
		"string",
		"love",
		3,
		10,
		"body",
	}

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
		schemas []Lannister
		want    bool
	}{
		{postBadRequestWithMinLength, []Lannister{validName, validSaviour, validBehaviour}, false},
		{postValidRequest, []Lannister{validName, validSaviour, validBehaviour}, true},
	}

	for _, c := range cases {
		got, _ := ValidateRequest(c.request, c.schemas...)
		if got != c.want {
			t.Errorf("Validate(%v) got %v, want %t", c.schemas, got, c.want)
		}
	}

}

func TestValidateJsonPostRequest(t *testing.T) {
	validName := Lannister{
		"name",
		true,
		"string",
		"",
		4,
		10,
		"body",
	}
	validSaviour := Lannister{
		"saviour",
		true,
		"string",
		"jesus",
		-1,
		10,
		"body",
	}
	validBehaviour := Lannister{
		"behaviour",
		true,
		"string",
		"love",
		3,
		10,
		"body",
	}

	validPostJson := []byte(`{"name":"martins","saviour":"jesus","behaviour":"love"}`)
	postValidRequest := httptest.NewRequest("POST", "http://google.com", bytes.NewBuffer(validPostJson))
	postValidRequest.Header.Set("Content-Type", "application/json")

	invalidPostJsonWithMissingField := []byte(`{"name":"martins","saviour":"jesus"}`)
	postInavlidValidRequest := httptest.NewRequest("POST", "http://google.com", bytes.NewBuffer(invalidPostJsonWithMissingField))
	postInavlidValidRequest.Header.Set("Content-Type", "application/json")

	cases := []struct {
		request *http.Request
		schemas []Lannister
		want    bool
	}{
		{postValidRequest, []Lannister{validName, validSaviour, validBehaviour}, true},
		{postInavlidValidRequest, []Lannister{validName, validSaviour, validBehaviour}, false},
	}

	for _, c := range cases {
		got, _ := ValidateRequest(c.request, c.schemas...)
		if got != c.want {
			t.Errorf("Validate(%v) got %v, want %t", c.schemas, got, c.want)
		}
	}

}
