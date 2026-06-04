package kingsguard

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestValidateGetRequest(t *testing.T) {
	getRequestWithCorrectQueryParams := httptest.NewRequest("GET", "http://google.com?name=martins&saviour=jesus", nil)
	getInvalidRequestWithWrongRegexQueryParams := httptest.NewRequest("GET", "http://google.com?name=martins&saviour=anyoneelse", nil)
	getRequestWithMissingQueryParams := httptest.NewRequest("GET", "http://google.com", nil)
	getInvalidRequestWithLongQueryParams := httptest.NewRequest("GET", "http://google.com?name=martinsdhjjhdhjhjshdjhsjd&saviour=jesus", nil)

	cases := []struct {
		request *http.Request
		schemas []*FieldValidator
		want    bool
		wantErr string // empty string means no error expected
	}{
		{
			request: getRequestWithCorrectQueryParams,
			schemas: []*FieldValidator{
				Field("name").Required().Type("string").Max(10).In("query"),
				Field("saviour").Required().Type("string").Regexp("jesus").Max(10).In("query"),
			},
			want:    true,
			wantErr: "",
		},
		{
			request: getInvalidRequestWithWrongRegexQueryParams,
			schemas: []*FieldValidator{
				Field("name").Required().Type("string").Max(10).In("query"),
				Field("saviour").Required().Type("string").Regexp("jesus").Max(10).In("query"),
			},
			want:    false,
			wantErr: "saviour does not match required pattern",
		},
		{
			request: getRequestWithMissingQueryParams,
			schemas: []*FieldValidator{
				Field("name").Required().Type("string").Max(10).In("query"),
				Field("saviour").Required().Type("string").Regexp("jesus").Max(10).In("query"),
			},
			want:    false,
			wantErr: "name is a required field",
		},
		{
			request: getInvalidRequestWithLongQueryParams,
			schemas: []*FieldValidator{
				Field("name").Required().Type("string").Max(10).In("query"),
				Field("saviour").Required().Type("string").Regexp("jesus").Max(10).In("query"),
			},
			want:    false,
			wantErr: "the maximum accepted length/value for name is 10",
		},
	}

	for _, c := range cases {
		got, err := ValidateRequest(c.request, c.schemas...)
		if got != c.want {
			t.Errorf("ValidateRequest() got %v, want %v", got, c.want)
		}
		if !c.want {
			if err == nil {
				t.Errorf("ValidateRequest() expected error %q, got nil", c.wantErr)
			} else if err.Error() != c.wantErr {
				t.Errorf("ValidateRequest() error = %q, wantErr %q", err.Error(), c.wantErr)
			}
		} else {
			if err != nil {
				t.Errorf("ValidateRequest() unexpected error: %v", err)
			}
		}
	}
}

func TestValidatePostFormRequest(t *testing.T) {
	form := url.Values{}
	form.Set("name", "martins")
	form.Add("saviour", "jesus")
	form.Add("behaviour", "love")
	req := httptest.NewRequest("POST", "http://google.com", strings.NewReader(form.Encode()))
	req.PostForm = form

	badForm := url.Values{}
	badForm.Set("name", "mar")
	badForm.Add("saviour", "jesus")
	badForm.Add("behaviour", "love")
	badReq := httptest.NewRequest("POST", "http://google.com", strings.NewReader(badForm.Encode()))
	badReq.PostForm = badForm

	cases := []struct {
		request *http.Request
		schemas []*FieldValidator
		want    bool
		wantErr string // empty string means no error expected
	}{
		{
			request: badReq,
			schemas: []*FieldValidator{
				Field("name").Required().Type("string").Min(4).Max(10).In("body"),
				Field("saviour").Required().Type("string").Regexp("jesus").Max(10).In("body"),
				Field("behaviour").Required().Type("string").Regexp("love").In("body"),
			},
			want:    false,
			wantErr: "the minimum accepted length/value for name is 4",
		},
		{
			request: req,
			schemas: []*FieldValidator{
				Field("name").Required().Type("string").Min(4).Max(10).In("body"),
				Field("saviour").Required().Type("string").Regexp("jesus").Max(10).In("body"),
				Field("behaviour").Required().Type("string").Regexp("love").In("body"),
			},
			want:    true,
			wantErr: "",
		},
	}

	for _, c := range cases {
		got, err := ValidateRequest(c.request, c.schemas...)
		if got != c.want {
			t.Errorf("ValidateRequest() got %v, want %v", got, c.want)
		}
		if !c.want {
			if err == nil {
				t.Errorf("ValidateRequest() expected error %q, got nil", c.wantErr)
			} else if err.Error() != c.wantErr {
				t.Errorf("ValidateRequest() error = %q, wantErr %q", err.Error(), c.wantErr)
			}
		} else {
			if err != nil {
				t.Errorf("ValidateRequest() unexpected error: %v", err)
			}
		}
	}
}

func TestValidatePostJsonRequest(t *testing.T) {
	body := []byte(`{"name":"martins","saviour":"jesus","behaviour":"love"}`)
	req := httptest.NewRequest("POST", "http://google.com", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	missingFieldBody := []byte(`{"name":"martins","saviour":"jesus"}`)
	missingFieldReq := httptest.NewRequest("POST", "http://google.com", bytes.NewBuffer(missingFieldBody))
	missingFieldReq.Header.Set("Content-Type", "application/json")

	// Case A — GAP-04 regression: JSON body where count is float64 after
	// json.Unmarshal — would have panicked in v1.
	countBody := []byte(`{"count": 5}`)
	gapRegressionReq := httptest.NewRequest("POST", "http://google.com", bytes.NewBuffer(countBody))
	gapRegressionReq.Header.Set("Content-Type", "application/json")

	// Case B — double body-validator replay: confirms r.Body is correctly
	// replayed after the first schema reads it, so the second schema can
	// still read the body.
	twoFieldBody := []byte(`{"name":"martins","email":"test@example.com"}`)
	twoFieldReq := httptest.NewRequest("POST", "http://google.com", bytes.NewBuffer(twoFieldBody))
	twoFieldReq.Header.Set("Content-Type", "application/json")

	cases := []struct {
		request *http.Request
		schemas []*FieldValidator
		want    bool
		wantErr string // empty string means no error expected
	}{
		{
			request: req,
			schemas: []*FieldValidator{
				Field("name").Required().Type("string").In("body"),
				Field("saviour").Required().Type("string").Regexp("jesus").In("body"),
				Field("behaviour").Required().Type("string").Regexp("love").In("body"),
			},
			want:    true,
			wantErr: "",
		},
		{
			request: missingFieldReq,
			schemas: []*FieldValidator{
				Field("name").Required().Type("string").In("body"),
				Field("saviour").Required().Type("string").Regexp("jesus").In("body"),
				Field("behaviour").Required().Type("string").Regexp("love").In("body"),
			},
			want:    false,
			wantErr: "behaviour is a required field",
		},
		{
			request: gapRegressionReq,
			schemas: []*FieldValidator{Field("count").Required().Type("string").In("body")},
			want:    false,
			wantErr: "count must be of type string",
		},
		{
			request: twoFieldReq,
			schemas: []*FieldValidator{
				Field("name").Required().Type("string").In("body"),
				Field("email").Required().Type("string").In("body"),
			},
			want:    true,
			wantErr: "",
		},
	}

	for _, c := range cases {
		got, err := ValidateRequest(c.request, c.schemas...)
		if got != c.want {
			t.Errorf("ValidateRequest() got %v, want %v", got, c.want)
		}
		if !c.want {
			if err == nil {
				t.Errorf("ValidateRequest() expected error %q, got nil", c.wantErr)
			} else if err.Error() != c.wantErr {
				t.Errorf("ValidateRequest() error = %q, wantErr %q", err.Error(), c.wantErr)
			}
		} else {
			if err != nil {
				t.Errorf("ValidateRequest() unexpected error: %v", err)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// GAP-04: isDataTypeCorrect — JSON body type assertion panic fixes
// ---------------------------------------------------------------------------

// makeJSONRequest creates an httptest.Request with a JSON body and
// Content-Type: application/json header.
func makeJSONRequest(body string) *http.Request {
	req := httptest.NewRequest("POST", "http://example.com", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// TestIsDataTypeCorrect_JSONInt verifies that a JSON number (decoded as float64
// by json.Unmarshal) is accepted as valid for a field declared with datatype "int",
// and does NOT panic — which was the original GAP-04 bug.
func TestIsDataTypeCorrect_JSONInt(t *testing.T) {
	schema := Field("age").Required().Type("int").In("body")
	req := makeJSONRequest(`{"age": 25}`)

	ok, err := isDataTypeCorrect(req, schema)
	if !ok {
		t.Errorf("isDataTypeCorrect: expected true for JSON int (float64), got false; err=%v", err)
	}
	if err != nil {
		t.Errorf("isDataTypeCorrect: expected nil error for valid int, got %v", err)
	}
}

// TestIsDataTypeCorrect_JSONFloat verifies that a JSON number is accepted as
// valid for a field declared with datatype "float".
func TestIsDataTypeCorrect_JSONFloat(t *testing.T) {
	schema := Field("price").Required().Type("float").In("body")
	req := makeJSONRequest(`{"price": 9.99}`)

	ok, err := isDataTypeCorrect(req, schema)
	if !ok {
		t.Errorf("isDataTypeCorrect: expected true for JSON float (float64), got false; err=%v", err)
	}
	if err != nil {
		t.Errorf("isDataTypeCorrect: expected nil error for valid float, got %v", err)
	}
}

// TestIsDataTypeCorrect_JSONBool verifies that a JSON boolean (decoded as Go bool)
// is accepted as valid for a field declared with datatype "bool".
func TestIsDataTypeCorrect_JSONBool(t *testing.T) {
	schema := Field("active").Required().Type("bool").In("body")
	req := makeJSONRequest(`{"active": true}`)

	ok, err := isDataTypeCorrect(req, schema)
	if !ok {
		t.Errorf("isDataTypeCorrect: expected true for JSON bool, got false; err=%v", err)
	}
	if err != nil {
		t.Errorf("isDataTypeCorrect: expected nil error for valid bool, got %v", err)
	}
}

// TestIsDataTypeCorrect_JSONString verifies that a JSON string is accepted as
// valid for a field declared with datatype "string".
func TestIsDataTypeCorrect_JSONString(t *testing.T) {
	schema := Field("name").Required().Type("string").In("body")
	req := makeJSONRequest(`{"name": "alice"}`)

	ok, err := isDataTypeCorrect(req, schema)
	if !ok {
		t.Errorf("isDataTypeCorrect: expected true for JSON string, got false; err=%v", err)
	}
	if err != nil {
		t.Errorf("isDataTypeCorrect: expected nil error for valid string, got %v", err)
	}
}

// TestIsDataTypeCorrect_JSONIntMismatch verifies that a JSON boolean value
// sent for a field declared with datatype "int" returns (false, error) rather
// than panicking — the primary GAP-04 scenario.
func TestIsDataTypeCorrect_JSONIntMismatch(t *testing.T) {
	schema := Field("age").Required().Type("int").In("body")
	req := makeJSONRequest(`{"age": true}`)

	ok, err := isDataTypeCorrect(req, schema)
	if ok {
		t.Error("isDataTypeCorrect: expected false for bool value in int field, got true")
	}
	wantErr := fmt.Sprintf("%s must be of type %s", "age", "int")
	if err == nil || err.Error() != wantErr {
		t.Errorf("isDataTypeCorrect: expected error %q, got %v", wantErr, err)
	}
}

// TestIsDataTypeCorrect_JSONStringMismatch verifies that a JSON number sent for a
// field declared with datatype "string" returns (false, error) rather than panicking.
func TestIsDataTypeCorrect_JSONStringMismatch(t *testing.T) {
	schema := Field("name").Required().Type("string").In("body")
	req := makeJSONRequest(`{"name": 42}`)

	ok, err := isDataTypeCorrect(req, schema)
	if ok {
		t.Error("isDataTypeCorrect: expected false for number value in string field, got true")
	}
	wantErr := fmt.Sprintf("%s must be of type %s", "name", "string")
	if err == nil || err.Error() != wantErr {
		t.Errorf("isDataTypeCorrect: expected error %q, got %v", wantErr, err)
	}
}

// TestIsDataTypeCorrect_JSONBoolMismatch verifies that a JSON number sent for a
// field declared with datatype "bool" returns (false, error).
func TestIsDataTypeCorrect_JSONBoolMismatch(t *testing.T) {
	schema := Field("active").Required().Type("bool").In("body")
	req := makeJSONRequest(`{"active": 1}`)

	ok, err := isDataTypeCorrect(req, schema)
	if ok {
		t.Error("isDataTypeCorrect: expected false for number value in bool field, got true")
	}
	wantErr := fmt.Sprintf("%s must be of type %s", "active", "bool")
	if err == nil || err.Error() != wantErr {
		t.Errorf("isDataTypeCorrect: expected error %q, got %v", wantErr, err)
	}
}

// TestIsDataTypeCorrect_NilVal verifies that a nil interface{} value (absent JSON key
// for a required field that somehow reached isDataTypeCorrect) returns (false, error)
// rather than panicking.
func TestIsDataTypeCorrect_NilVal(t *testing.T) {
	// We reach isDataTypeCorrect directly via ValidateRequest with a body that
	// is missing the field. isrequiredFieldPresent will return false for a required
	// field, so we use a non-required schema and verify ValidateRequest short-circuits.
	// To test isDataTypeCorrect itself with a nil guard, we call it directly.
	schema := Field("score").Required().Type("int").In("body")
	// Body does not contain "score" at all.
	req := makeJSONRequest(`{"other": 1}`)

	ok, err := isDataTypeCorrect(req, schema)
	if ok {
		t.Error("isDataTypeCorrect: expected false for absent (nil) JSON key, got true")
	}
	wantErr := fmt.Sprintf("%s must be of type %s", "score", "int")
	if err == nil || err.Error() != wantErr {
		t.Errorf("isDataTypeCorrect: expected error %q, got %v", wantErr, err)
	}
}

// ---------------------------------------------------------------------------
// GAP-04: ValidateRequest — end-to-end tests with non-string JSON types
// ---------------------------------------------------------------------------

// TestValidateRequest_JSONIntField verifies end-to-end that a JSON body with an
// integer field (decoded as float64) passes validation for datatype "int".
func TestValidateRequest_JSONIntField(t *testing.T) {
	schema := Field("age").Required().Type("int").In("body")
	req := makeJSONRequest(`{"age": 25}`)

	ok, err := ValidateRequest(req, schema)
	if !ok {
		t.Errorf("ValidateRequest: expected true for JSON int body, got false; err=%v", err)
	}
}

// TestValidateRequest_JSONIntFieldTypeMismatch verifies end-to-end that a JSON body
// with a boolean in an int field returns (false, error) with the correct message.
func TestValidateRequest_JSONIntFieldTypeMismatch(t *testing.T) {
	schema := Field("age").Required().Type("int").In("body")
	req := makeJSONRequest(`{"age": true}`)

	ok, err := ValidateRequest(req, schema)
	if ok {
		t.Error("ValidateRequest: expected false for bool value in int field, got true")
	}
	wantErr := "age must be of type int"
	if err == nil || err.Error() != wantErr {
		t.Errorf("ValidateRequest: expected error %q, got %v", wantErr, err)
	}
}

// TestValidateRequest_JSONFloatField verifies end-to-end that a JSON body with a
// float field passes validation for datatype "float".
func TestValidateRequest_JSONFloatField(t *testing.T) {
	schema := Field("price").Required().Type("float").In("body")
	req := makeJSONRequest(`{"price": 9.99}`)

	ok, err := ValidateRequest(req, schema)
	if !ok {
		t.Errorf("ValidateRequest: expected true for JSON float body, got false; err=%v", err)
	}
}

// TestValidateRequest_JSONBoolField verifies end-to-end that a JSON body with a
// boolean field passes validation for datatype "bool".
func TestValidateRequest_JSONBoolField(t *testing.T) {
	schema := Field("active").Required().Type("bool").In("body")
	req := makeJSONRequest(`{"active": false}`)

	ok, err := ValidateRequest(req, schema)
	if !ok {
		t.Errorf("ValidateRequest: expected true for JSON bool body, got false; err=%v", err)
	}
}

// TestValidateRequest_JSONRequiredFieldMissing verifies the exact error format for a
// missing required field: "<fieldName> is a required field".
func TestValidateRequest_JSONRequiredFieldMissing(t *testing.T) {
	schema := Field("name").Required().Type("string").In("body")
	req := makeJSONRequest(`{}`)

	ok, err := ValidateRequest(req, schema)
	if ok {
		t.Error("ValidateRequest: expected false for missing required field, got true")
	}
	wantErr := "name is a required field"
	if err == nil || err.Error() != wantErr {
		t.Errorf("ValidateRequest: expected error %q, got %v", wantErr, err)
	}
}

// TestValidateRequest_JSONIntWithMinMax verifies that min/max constraints work
// correctly for a JSON int field (where the value is decoded as float64).
func TestValidateRequest_JSONIntWithMinMax(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		min     int
		max     int
		wantOK  bool
		wantErr string
	}{
		{
			name:   "int within range",
			body:   `{"age": 25}`,
			min:    18,
			max:    100,
			wantOK: true,
		},
		{
			name:    "int below min",
			body:    `{"age": 5}`,
			min:     18,
			max:     100,
			wantOK:  false,
			wantErr: "the minimum accepted length/value for age is 18",
		},
		{
			name:    "int above max",
			body:    `{"age": 200}`,
			min:     18,
			max:     100,
			wantOK:  false,
			wantErr: "the maximum accepted length/value for age is 100",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			schema := Field("age").Required().Type("int").Min(tc.min).Max(tc.max).In("body")
			req := makeJSONRequest(tc.body)

			ok, err := ValidateRequest(req, schema)
			if ok != tc.wantOK {
				t.Errorf("ValidateRequest: got ok=%v, want %v; err=%v", ok, tc.wantOK, err)
			}
			if tc.wantErr != "" {
				if err == nil || err.Error() != tc.wantErr {
					t.Errorf("ValidateRequest: got error %v, want %q", err, tc.wantErr)
				}
			}
		})
	}
}

// TestValidateRequest_JSONStringWithMinMax verifies that min/max constraints work
// correctly for a JSON string field.
func TestValidateRequest_JSONStringWithMinMax(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		min     int
		max     int
		wantOK  bool
		wantErr string
	}{
		{
			name:   "string within length range",
			body:   `{"username": "alice"}`,
			min:    3,
			max:    10,
			wantOK: true,
		},
		{
			name:    "string too short",
			body:    `{"username": "ab"}`,
			min:     3,
			max:     10,
			wantOK:  false,
			wantErr: "the minimum accepted length/value for username is 3",
		},
		{
			name:    "string too long",
			body:    `{"username": "averylongusername"}`,
			min:     3,
			max:     10,
			wantOK:  false,
			wantErr: "the maximum accepted length/value for username is 10",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			schema := Field("username").Required().Type("string").Min(tc.min).Max(tc.max).In("body")
			req := makeJSONRequest(tc.body)

			ok, err := ValidateRequest(req, schema)
			if ok != tc.wantOK {
				t.Errorf("ValidateRequest: got ok=%v, want %v; err=%v", ok, tc.wantOK, err)
			}
			if tc.wantErr != "" {
				if err == nil || err.Error() != tc.wantErr {
					t.Errorf("ValidateRequest: got error %v, want %q", err, tc.wantErr)
				}
			}
		})
	}
}

// TestValidateRequest_JSONRegexWithStringField verifies regex matching still works
// correctly for JSON string fields after the type-assertion fixes.
func TestValidateRequest_JSONRegexWithStringField(t *testing.T) {
	cases := []struct {
		name   string
		body   string
		wantOK bool
	}{
		{"matching pattern", `{"email": "user@example.com"}`, true},
		{"non-matching pattern", `{"email": "notanemail"}`, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			schema := Field("email").Required().Type("string").Regexp(`^[^@]+@[^@]+\.[^@]+$`).In("body")
			req := makeJSONRequest(tc.body)

			ok, err := ValidateRequest(req, schema)
			if ok != tc.wantOK {
				t.Errorf("ValidateRequest: got ok=%v, want %v; err=%v", ok, tc.wantOK, err)
			}
		})
	}
}

// TestTypeNormalisationLowercase verifies that type names passed to Type() are
// normalised to lowercase, ensuring no uppercase case branches are needed in
// the helper switch statements.
func TestTypeNormalisationLowercase(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"String", "string"},
		{"INT", "int"},
		{"Float", "float"},
		{"Bool", "bool"},
		{"string", "string"},
	}

	for _, tc := range cases {
		fv := Field("x").Type(tc.input)
		if fv.datatype != tc.want {
			t.Errorf("Type(%q).datatype = %q, want %q", tc.input, fv.datatype, tc.want)
		}
	}
}
