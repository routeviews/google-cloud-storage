package main

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestErrorHandlerEscaping(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)

	unsafeAlert := "<script>alert('xss')</script>"
	ErrorHandler(rec, req, http.StatusInternalServerError, unsafeAlert, fmt.Errorf("test error"))

	body := rec.Body.String()
	if strings.Contains(body, unsafeAlert) {
		t.Errorf("Response body contains unescaped alert: %q", body)
	}

	escapedAlert := html.EscapeString(unsafeAlert)
	if !strings.Contains(body, escapedAlert) {
		t.Errorf("Response body does not contain escaped alert: %q", body)
	}
}

func TestPullToDB_MissingCronHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/update", nil)

	// Should fail with Forbidden and NOT panic because it exits before BQ calls.
	pullToDB(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("Expected status %v, got %v", http.StatusForbidden, rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/plain") {
		t.Errorf("Expected Content-Type to start with 'text/plain', got %q", contentType)
	}

	body := rec.Body.String()
	expectedBody := "Error 403: Forbidden: OIDC verification failed: missing Authorization header\n"
	if body != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, body)
	}
}

func TestPullToDB_WrongCronHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/update", nil)
	req.Header.Set("X-Appengine-Cron", "false")

	pullToDB(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("Expected status %v, got %v", http.StatusForbidden, rec.Code)
	}
}

func TestTextErrorHandler(t *testing.T) {
	rec := httptest.NewRecorder()
	TextErrorHandler(rec, http.StatusBadRequest, "Bad Request occurred", fmt.Errorf("some detail"))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %v, got %v", http.StatusBadRequest, rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/plain") {
		t.Errorf("Expected Content-Type to start with 'text/plain', got %q", contentType)
	}

	body := rec.Body.String()
	expectedBody := "Error 400: Bad Request occurred: some detail\n"
	if body != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, body)
	}
}

func TestNormalizeASN(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{"15169", "AS15169", false},
		{"AS15169", "AS15169", false},
		{"as15169", "AS15169", false},
		{"", "", false},
		{"   15169   ", "AS15169", false},
		{"AS", "", true},
		{"15169foo", "", true},
		{"ASfoobar", "", true},
	}

	for _, tc := range tests {
		got, err := normalizeASN(tc.input)
		if (err != nil) != tc.wantErr {
			t.Errorf("normalizeASN(%q) error = %v, wantErr %v", tc.input, err, tc.wantErr)
			continue
		}
		if got != tc.expected {
			t.Errorf("normalizeASN(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestNormalizePrefix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{"8.8.8.0/24", "8.8.8.0/24", false},
		{"1.1.1.1", "1.1.1.0/24", false},
		{"2001:4860:4860::8888", "2001:4860:4860::/48", false},
		{"2001:db8::/32", "2001:db8::/32", false},
		{"", "", false},
		{"   8.8.8.8   ", "8.8.8.0/24", false},
		{"invalid-ip", "", true},
		{"8.8.8.0/99", "", true},
	}

	for _, tc := range tests {
		got, err := normalizePrefix(tc.input)
		if (err != nil) != tc.wantErr {
			t.Errorf("normalizePrefix(%q) error = %v, wantErr %v", tc.input, err, tc.wantErr)
			continue
		}
		if got != tc.expected {
			t.Errorf("normalizePrefix(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestComputeAvailabilityRanges(t *testing.T) {
	// Consecutive daily updates
	d1 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	d2 := d1.Add(24 * time.Hour)
	d3 := d2.Add(24 * time.Hour)

	// Large gap (interruption)
	d4 := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	d5 := d4.Add(24 * time.Hour)

	times := []time.Time{d1, d2, d3, d4, d5}

	got := computeAvailabilityRanges(times, 26*time.Hour)
	expected := []string{
		"Jan 1 2026 -> Jan 3 2026",
		"Feb 1 2026 -> Feb 2 2026",
	}

	if len(got) != len(expected) {
		t.Fatalf("got %d ranges, want %d", len(got), len(expected))
	}
	for i := range got {
		if got[i] != expected[i] {
			t.Errorf("range %d = %q, want %q", i, got[i], expected[i])
		}
	}
}

func TestConvInToStored(t *testing.T) {
	tests := []struct {
		input    inputROA
		expected storedROA
		wantErr  bool
	}{
		{
			input:    inputROA{Asn: "AS15169", Prefix: "8.8.8.0/24", MaxLength: 24, Ta: "arin"},
			expected: storedROA{Asn: "AS15169", Prefix: "8.8.8.0", MaxLength: 24, Ta: "arin", Subnet: 24},
			wantErr:  false,
		},
		{
			input:    inputROA{Prefix: "8.8.8.0"},
			expected: storedROA{},
			wantErr:  true,
		},
		{
			input:    inputROA{Prefix: "8.8.8.0/invalid"},
			expected: storedROA{},
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		got, err := convInToStored(tc.input)
		if (err != nil) != tc.wantErr {
			t.Errorf("convInToStored(%v) error = %v, wantErr %v", tc.input, err, tc.wantErr)
			continue
		}
		if !tc.wantErr && got != tc.expected {
			t.Errorf("convInToStored(%v) = %+v, want %+v", tc.input, got, tc.expected)
		}
	}
}

func TestHSTS(t *testing.T) {
	// Test HTTP redirection
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	req.Header.Set("X-Forwarded-Proto", "http")

	hsts(rec, req)

	if rec.Code != http.StatusMovedPermanently {
		t.Errorf("Expected redirect status %v, got %v", http.StatusMovedPermanently, rec.Code)
	}
	if got := rec.Header().Get("Strict-Transport-Security"); got != "max-age=2629800" {
		t.Errorf("Expected HSTS header, got %q", got)
	}

	// Test HTTPS passthrough
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "https://example.com/foo", nil)
	req2.Header.Set("X-Forwarded-Proto", "https")

	hsts(rec2, req2)

	if rec2.Code == http.StatusMovedPermanently {
		t.Errorf("Did not expect redirect for HTTPS request")
	}
}

func TestDownloadRARC(t *testing.T) {
	// Test successful download
	fakeJSON := `{"roas":[{"asn":"AS15169","prefix":"8.8.8.0/24","maxLength":24,"ta":"arin"}]}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fakeJSON))
	}))
	defer ts.Close()

	// Override roaURL
	origURL := roaURL
	roaURL = ts.URL
	defer func() { roaURL = origURL }()

	res, err := downloadRARC()
	if err != nil {
		t.Fatalf("downloadRARC failed: %v", err)
	}
	if len(res.Roas) != 1 || res.Roas[0].Asn != "AS15169" {
		t.Errorf("Unexpected result: %+v", res)
	}

	// Test 500 error from server
	tsErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Server crash"))
	}))
	defer tsErr.Close()

	roaURL = tsErr.URL
	_, err = downloadRARC()
	if err == nil {
		t.Errorf("Expected error for 500 response, got nil")
	}

	// Test invalid JSON
	tsBadJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("{invalid-json"))
	}))
	defer tsBadJSON.Close()

	roaURL = tsBadJSON.URL
	_, err = downloadRARC()
	if err == nil {
		t.Errorf("Expected error for bad JSON, got nil")
	}
}

func TestVerifyOIDCToken_Errors(t *testing.T) {
	ctx := context.Background()

	// Missing header
	req1 := httptest.NewRequest("GET", "/update", nil)
	if err := verifyOIDCToken(ctx, req1); err == nil {
		t.Errorf("Expected error for missing header, got nil")
	}

	// Invalid header format
	req2 := httptest.NewRequest("GET", "/update", nil)
	req2.Header.Set("Authorization", "Basic somedata")
	if err := verifyOIDCToken(ctx, req2); err == nil {
		t.Errorf("Expected error for invalid format, got nil")
	}

	// Invalid token (idtoken.Validate should fail)
	req3 := httptest.NewRequest("GET", "/update", nil)
	req3.Header.Set("Authorization", "Bearer fake-invalid-jwt-token")
	if err := verifyOIDCToken(ctx, req3); err == nil {
		t.Errorf("Expected error for fake token, got nil")
	}
}

func TestMainPage_InitialGet(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)

	mainPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %v", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Historical ROA Query") {
		t.Errorf("Response body does not contain expected title")
	}
}

func TestMainPage_InvalidCriteria(t *testing.T) {
	// Test Bad ASN
	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("POST", "/?asn=invalid_asn", nil)
	mainPage(rec1, req1)
	if rec1.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for bad ASN, got %v", rec1.Code)
	}

	// Test Bad Prefix
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("POST", "/?prefix=invalid_prefix", nil)
	mainPage(rec2, req2)
	if rec2.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for bad Prefix, got %v", rec2.Code)
	}

	// Test Missing both criteria
	rec3 := httptest.NewRecorder()
	req3 := httptest.NewRequest("POST", "/", nil)
	mainPage(rec3, req3)
	if rec3.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing criteria, got %v", rec3.Code)
	}
}



