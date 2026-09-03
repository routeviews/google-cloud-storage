package main

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/option"
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
	tests := []struct {
		name         string
		status       int
		alert        string
		err          error
		expectedCode int
		expectedBody string
	}{
		{
			name:         "WithError",
			status:       http.StatusBadRequest,
			alert:        "Bad Request occurred",
			err:          fmt.Errorf("some detail"),
			expectedCode: http.StatusBadRequest,
			expectedBody: "Error 400: Bad Request occurred: some detail\n",
		},
		{
			name:         "WithoutError",
			status:       http.StatusNotFound,
			alert:        "Not Found",
			err:          nil,
			expectedCode: http.StatusNotFound,
			expectedBody: "Error 404: Not Found\n",
		},
	}

	for _, tc := range tests {
		rec := httptest.NewRecorder()
		TextErrorHandler(rec, tc.status, tc.alert, tc.err)

		if rec.Code != tc.expectedCode {
			t.Errorf("[%s] Expected status %v, got %v", tc.name, tc.expectedCode, rec.Code)
		}
		contentType := rec.Header().Get("Content-Type")
		if !strings.HasPrefix(contentType, "text/plain") {
			t.Errorf("[%s] Expected Content-Type to start with 'text/plain', got %q", tc.name, contentType)
		}
		if body := rec.Body.String(); body != tc.expectedBody {
			t.Errorf("[%s] Expected body %q, got %q", tc.name, tc.expectedBody, body)
		}
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
	d1 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	d2 := d1.Add(24 * time.Hour)
	d3 := d2.Add(24 * time.Hour)
	d4 := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	d5 := d4.Add(24 * time.Hour)

	tests := []struct {
		name      string
		times     []time.Time
		threshold time.Duration
		expected  []string
	}{
		{
			name:      "EmptyTimes",
			times:     nil,
			threshold: 26 * time.Hour,
			expected:  nil,
		},
		{
			name:      "SingleTime",
			times:     []time.Time{d1},
			threshold: 26 * time.Hour,
			expected:  []string{"Jan 1 2026"},
		},
		{
			name:      "SameDayTimes",
			times:     []time.Time{d1, d1.Add(2 * time.Hour)},
			threshold: 26 * time.Hour,
			expected:  []string{"Jan 1 2026"},
		},
		{
			name:      "ConsecutiveAndGaps",
			times:     []time.Time{d1, d2, d3, d4, d5},
			threshold: 26 * time.Hour,
			expected: []string{
				"Jan 1 2026 -> Jan 3 2026",
				"Feb 1 2026 -> Feb 2 2026",
			},
		},
	}

	for _, tc := range tests {
		got := computeAvailabilityRanges(tc.times, tc.threshold)
		if len(got) != len(tc.expected) {
			t.Errorf("[%s] got %d ranges, want %d", tc.name, len(got), len(tc.expected))
			continue
		}
		for i := range got {
			if got[i] != tc.expected[i] {
				t.Errorf("[%s] range %d = %q, want %q", tc.name, i, got[i], tc.expected[i])
			}
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

func TestInitBQClient(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		emulatorHost string
		wantErr      bool
	}{
		{
			name:         "WithEmulatorHost",
			emulatorHost: "localhost:9050",
			wantErr:      false,
		},
		{
			name:         "WithoutEmulatorHost",
			emulatorHost: "",
			wantErr:      true,
		},
	}

	for _, tc := range tests {
		origHost := os.Getenv("BIGQUERY_EMULATOR_HOST")
		if tc.emulatorHost != "" {
			os.Setenv("BIGQUERY_EMULATOR_HOST", tc.emulatorHost)
		} else {
			os.Unsetenv("BIGQUERY_EMULATOR_HOST")
		}

		c, err := initBQClient(ctx)
		if (err != nil) != tc.wantErr {
			t.Errorf("[%s] initBQClient() error = %v, wantErr = %v", tc.name, err, tc.wantErr)
		}
		if c != nil {
			c.Close()
		}

		if origHost != "" {
			os.Setenv("BIGQUERY_EMULATOR_HOST", origHost)
		} else {
			os.Unsetenv("BIGQUERY_EMULATOR_HOST")
		}
	}
}

func TestMainPage_QueryExecutionError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": {"code": 400, "message": "unretryable query error", "status": "INVALID_ARGUMENT"}}`))
	}))
	defer ts.Close()

	ctx := context.Background()
	dummyClient, err := bigquery.NewClient(ctx, "public-routing-data-backup",
		option.WithEndpoint(ts.URL),
		option.WithoutAuthentication(),
	)
	if err != nil {
		t.Fatalf("failed to create dummy client: %v", err)
	}
	defer dummyClient.Close()

	origClient := client
	client = dummyClient
	defer func() { client = origClient }()

	tests := []struct {
		name        string
		targetURL   string
		expectedErr string
	}{
		{
			name:        "ASNQueryFails",
			targetURL:   "/?asn=AS15169",
			expectedErr: "Error with query",
		},
		{
			name:        "PrefixQueryFails",
			targetURL:   "/?prefix=8.8.8.0/24",
			expectedErr: "Error with query",
		},
		{
			name:        "BothQueryFails",
			targetURL:   "/?asn=AS15169&prefix=8.8.8.0/24",
			expectedErr: "Error with query",
		},
		{
			name:        "ParseCIDRQueryFails",
			targetURL:   "/?prefix=8.8.8.8/24&parsecidr=true",
			expectedErr: "Error with query",
		},
	}

	for _, tc := range tests {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", tc.targetURL, nil)

		mainPage(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("[%s] Expected status %v, got %v", tc.name, http.StatusInternalServerError, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), tc.expectedErr) {
			t.Errorf("[%s] Body does not contain %q: %s", tc.name, tc.expectedErr, rec.Body.String())
		}
	}
}

func TestPullToDB_DownloadError(t *testing.T) {
	tsErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": {"code": 400, "message": "unretryable error", "status": "INVALID_ARGUMENT"}}`))
	}))
	defer tsErr.Close()

	ctx := context.Background()
	dummyClient, err := bigquery.NewClient(ctx, "public-routing-data-backup",
		option.WithEndpoint(tsErr.URL),
		option.WithoutAuthentication(),
	)
	if err != nil {
		t.Fatalf("failed to create dummy client: %v", err)
	}
	defer dummyClient.Close()

	origClient := client
	client = dummyClient
	defer func() { client = origClient }()

	origURL := roaURL
	roaURL = tsErr.URL
	defer func() { roaURL = origURL }()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/update", nil)
	req.Header.Set("X-Appengine-Cron", "true")

	pullToDB(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %v", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Error parsing JSON") {
		t.Errorf("Expected body to contain 'Error parsing JSON', got %s", rec.Body.String())
	}
}

func TestPullToDB_Cooldown(t *testing.T) {
	nowMillis := time.Now().UnixMilli()
	oldMillis := time.Now().Add(-2 * time.Hour).UnixMilli()

	tests := []struct {
		name             string
		lastModifiedTime int64
		cooldown         time.Duration
		expectCooldown   bool
	}{
		{
			name:             "RecentUpdateSkips",
			lastModifiedTime: nowMillis,
			cooldown:         50 * time.Minute,
			expectCooldown:   true,
		},
		{
			name:             "OldUpdateProceeds",
			lastModifiedTime: oldMillis,
			cooldown:         50 * time.Minute,
			expectCooldown:   false,
		},
	}

	for _, tc := range tests {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if strings.Contains(r.URL.Path, "/tables/roas_arr") {
				resp := fmt.Sprintf(`{
					"kind": "bigquery#table",
					"tableReference": {
						"projectId": "public-routing-data-backup",
						"datasetId": "historical",
						"tableId": "roas_arr"
					},
					"lastModifiedTime": "%d"
				}`, tc.lastModifiedTime)
				w.Write([]byte(resp))
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": {"code": 400, "message": "stop here", "status": "INVALID_ARGUMENT"}}`))
		}))

		ctx := context.Background()
		dummyClient, err := bigquery.NewClient(ctx, "public-routing-data-backup",
			option.WithEndpoint(ts.URL),
			option.WithoutAuthentication(),
		)
		if err != nil {
			ts.Close()
			t.Fatalf("[%s] failed to create dummy client: %v", tc.name, err)
		}

		origClient := client
		client = dummyClient
		origCooldown := updateCooldown
		updateCooldown = tc.cooldown
		origURL := roaURL
		roaURL = ts.URL

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/update", nil)
		req.Header.Set("X-Appengine-Cron", "true")

		pullToDB(rec, req)

		client = origClient
		updateCooldown = origCooldown
		roaURL = origURL
		dummyClient.Close()
		ts.Close()

		if tc.expectCooldown {
			if rec.Code != http.StatusOK {
				t.Errorf("[%s] Expected status 200, got %v", tc.name, rec.Code)
			}
			if !strings.Contains(rec.Body.String(), "Skipped: already updated in last 50 mins") {
				t.Errorf("[%s] Expected body to contain 'Skipped', got %s", tc.name, rec.Body.String())
			}
		} else {
			if strings.Contains(rec.Body.String(), "Skipped: already updated in last 50 mins") {
				t.Errorf("[%s] Did not expect cooldown skip, got %s", tc.name, rec.Body.String())
			}
		}
	}
}



