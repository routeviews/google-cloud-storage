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
	fakeJSON := `{"roas":[{"asn":"AS15169","prefix":"8.8.8.0/24","maxLength":24,"ta":"arin"}]}`
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

	// 3. Test Web Interface Querying via mainPage
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("POST", "/?asn=AS15169", nil)

	mainPage(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Errorf("mainPage query failed: status %v, body: %s", rec2.Code, rec2.Body.String())
	}

	body2 := rec2.Body.String()
	if !strings.Contains(body2, "AS15169") || !strings.Contains(body2, "8.8.8.0/24") {
		t.Errorf("mainPage did not render expected BQ data: %s", body2)
	}

	// 4. Test JSON Endpoint via mainPage
	rec3 := httptest.NewRecorder()
	req3 := httptest.NewRequest("GET", "/?asn=AS15169&json=true", nil)

	mainPage(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Errorf("mainPage json failed: status %v", rec3.Code)
	}

	body3 := rec3.Body.String()
	if !strings.Contains(body3, "AS15169") || !strings.Contains(body3, "8.8.8.0/24") {
		t.Errorf("mainPage json did not return expected structure: %s", body3)
	}
}
