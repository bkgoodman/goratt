package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestHTTPValidation(t *testing.T) {
	printer, err := NewPrinter("/dev/null")
	if err != nil {
		t.Fatalf("NewPrinter failed: %v", err)
	}

	server := StartHTTPServer(0, printer)
	defer server.Shutdown()

	handler := server.server.Handler

	tests := []struct {
		name           string
		method         string
		path           string
		form           url.Values
		expectedStatus int
		expectPrint    bool
	}{
		{
			name:           "GET / should be rejected with 405 Method Not Allowed",
			method:         http.MethodGet,
			path:           "/",
			form:           nil,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "GET /favicon.ico should be rejected with 405",
			method:         http.MethodGet,
			path:           "/favicon.ico",
			form:           nil,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "GET /health should return 200 OK",
			method:         http.MethodGet,
			path:           "/health",
			form:           nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST with empty form should be rejected with 400 Bad Request",
			method:         http.MethodPost,
			path:           "/",
			form:           url.Values{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "POST with missing member name should be rejected with 400",
			method: http.MethodPost,
			path:   "/",
			form: url.Values{
				"number": []string{"60"},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "POST with missing number should be rejected with 400",
			method: http.MethodPost,
			path:   "/",
			form: url.Values{
				"name": []string{"Frank.Pallien"},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "POST with non-numeric number should be rejected with 400",
			method: http.MethodPost,
			path:   "/",
			form: url.Values{
				"name":   []string{"Frank.Pallien"},
				"number": []string{"notanumber"},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "POST with out-of-range number should be rejected with 400",
			method: http.MethodPost,
			path:   "/",
			form: url.Values{
				"name":   []string{"Frank.Pallien"},
				"number": []string{"999999"},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "Valid POST with member and number should succeed with 200 OK",
			method: http.MethodPost,
			path:   "/",
			form: url.Values{
				"name":   []string{"Frank.Pallien"},
				"number": []string{"60"},
			},
			expectedStatus: http.StatusOK,
			expectPrint:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var body string
			if tc.form != nil {
				body = tc.form.Encode()
			}
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(body))
			if tc.form != nil {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tc.expectedStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tc.expectedStatus, rec.Code, rec.Body.String())
			}
		})
	}
}
