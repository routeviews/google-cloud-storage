package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
)

func TestIntegration_Emulator(t *testing.T) {
	if os.Getenv("BIGQUERY_EMULATOR_HOST") == "" {
		t.Skip("BIGQUERY_EMULATOR_HOST not set, skipping BigQuery integration test")
	}

	ctx := context.Background()
	var err error
	client, err = initBQClient(ctx)
	if err != nil {
		t.Fatalf("initBQClient failed: %v", err)
	}
	client.Location = projectLocation

	// Set cooldown to negative so pullToDB never skips due to table creation timestamp
	origCooldown := updateCooldown
	updateCooldown = -1 * time.Second
	defer func() { updateCooldown = origCooldown }()

	// Override MERGE condition to single expression for emulator compatibility
	origMergeCond := mergeOnCond
	mergeOnCond = "b.prefix = arr.prefix"
	defer func() { mergeOnCond = origMergeCond }()

	// Override SELECT queries to return empty arrays for inserttimes (bypassing SQLite timestamp string unmarshaling issues)
	origQueryASN, origQueryPrefix, origQueryBoth := queryASN, queryPrefix, queryBoth
	queryASN = "SELECT asn, prefix, mask, maxlen, ta, [] as inserttimes FROM public-routing-data-backup.historical.roas_arr WHERE asn = @asn"
	queryPrefix = "SELECT asn, prefix, mask, maxlen, ta, [] as inserttimes FROM public-routing-data-backup.historical.roas_arr WHERE prefix = @prefix AND mask = @mask"
	queryBoth = "SELECT asn, prefix, mask, maxlen, ta, [] as inserttimes FROM public-routing-data-backup.historical.roas_arr WHERE asn = @asn AND prefix = @prefix AND mask = @mask"
	defer func() {
		queryASN, queryPrefix, queryBoth = origQueryASN, origQueryPrefix, origQueryBoth
	}()

	// 1. Setup Emulator Dataset and Tables
	err = client.Dataset("historical").Create(ctx, &bigquery.DatasetMetadata{})
	if err != nil && !strings.Contains(err.Error(), "Already Exists") && !strings.Contains(err.Error(), "is already created") {
		t.Fatalf("failed to create historical dataset: %v", err)
	}

	// Create primary roas_arr table
	_, err = client.Query(`CREATE TABLE IF NOT EXISTS public-routing-data-backup.historical.roas_arr (
		asn STRING,
		prefix STRING,
		maxlen INT64,
		ta STRING,
		mask INT64,
		inserttimes ARRAY<TIMESTAMP>
	) CLUSTER BY prefix, mask, asn`).Run(ctx)
	if err != nil {
		t.Fatalf("failed to create roas_arr table: %v", err)
	}

	// 2. Test Ingestion via pullToDB
	fakeJSON := `{
		"roas": [
			{"asn": "AS15169", "prefix": "8.8.8.0/24", "maxLength": 24, "ta": "arin"},
			{"asn": "AS13335", "prefix": "1.1.1.0/24", "maxLength": 28, "ta": "apnic"},
			{"asn": "AS3356", "prefix": "invalid-no-slash", "maxLength": 24, "ta": "arin"},
			{"asn": "AS3356", "prefix": "4.0.0.0/badmask", "maxLength": 24, "ta": "arin"}
		]
	}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fakeJSON))
	}))
	defer ts.Close()

	origURL := roaURL
	roaURL = ts.URL
	defer func() { roaURL = origURL }()

	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/update", nil)
	req1.Header.Set("X-Appengine-Cron", "true")

	pullToDB(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("pullToDB failed: status %v, body: %s", rec1.Code, rec1.Body.String())
	}
	if !strings.Contains(rec1.Body.String(), "Update successful") {
		t.Errorf("Unexpected pullToDB response: %s", rec1.Body.String())
	}

	// Test download error in pullToDB
	tsErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("download failure"))
	}))
	defer tsErr.Close()

	roaURL = tsErr.URL
	recErr := httptest.NewRecorder()
	pullToDB(recErr, req1)
	if recErr.Code != http.StatusInternalServerError {
		t.Errorf("pullToDB error expected 500, got %v", recErr.Code)
	}
	roaURL = ts.URL

	// 3. Test Web Interface and JSON Querying via mainPage
	queries := []struct {
		name           string
		method         string
		url            string
		expectedSubstr []string
	}{
		{
			name:           "QueryByASN",
			method:         "POST",
			url:            "/?asn=AS15169",
			expectedSubstr: []string{"AS15169", "8.8.8.0/24"},
		},
		{
			name:           "QueryByPrefix",
			method:         "POST",
			url:            "/?prefix=8.8.8.0/24",
			expectedSubstr: []string{"AS15169", "8.8.8.0/24"},
		},
		{
			name:           "QueryByBoth",
			method:         "POST",
			url:            "/?asn=AS15169&prefix=8.8.8.0/24",
			expectedSubstr: []string{"AS15169", "8.8.8.0/24"},
		},
		{
			name:           "QueryWithParseCIDR",
			method:         "POST",
			url:            "/?prefix=8.8.8.8/24&parsecidr=true",
			expectedSubstr: []string{"AS15169", "8.8.8.0/24"},
		},
		{
			name:           "QueryMaxlenNotEqualMask",
			method:         "POST",
			url:            "/?asn=AS13335",
			expectedSubstr: []string{"AS13335", "1.1.1.0/24 =&gt; 28"},
		},
		{
			name:           "QueryJSON",
			method:         "GET",
			url:            "/?asn=AS15169&json=true",
			expectedSubstr: []string{"AS15169", "8.8.8.0/24"},
		},
		{
			name:           "QueryJSON_MaxlenNotEqualMask",
			method:         "GET",
			url:            "/?asn=AS13335&json=true",
			expectedSubstr: []string{"AS13335", "1.1.1.0/24 => 28"},
		},
		{
			name:           "QueryEmptyResults",
			method:         "POST",
			url:            "/?asn=AS999999",
			expectedSubstr: []string{"Historical ROA Query"},
		},
	}

	for _, tc := range queries {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.url, nil)

		mainPage(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("[%s] mainPage query failed: status %v, body: %s", tc.name, rec.Code, rec.Body.String())
			continue
		}

		body := rec.Body.String()
		for _, substr := range tc.expectedSubstr {
			if !strings.Contains(body, substr) {
				t.Errorf("[%s] body does not contain %q: %s", tc.name, substr, body)
			}
		}
	}
}
