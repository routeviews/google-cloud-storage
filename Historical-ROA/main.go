// Package main is the main web-server for handling user requests for Historical ROA
// lookups. It also handles updates through a simple URL + auth scheme.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"html/template"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shomali11/util/xhashes"

	"flag"

	"cloud.google.com/go/bigquery"
	pb "github.com/gidoBOSSftw5731/Historical-ROA/proto"
	"github.com/golang/glog"
	"google.golang.org/api/idtoken"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/encoding/protojson"
)

// inputROA is a Struct with all the data from the json
// we do NOT store this directly.
// https://mholt.github.io/json-to-go/
type inputROA struct {
	Asn       string `json:"asn"`
	Prefix    string `json:"prefix"`
	MaxLength int    `json:"maxLength"`
	Ta        string `json:"ta"`
	ParseCIDR string
}

type inputROAArr struct {
	Roas []inputROA `json:"roas"`
}

// storedROAs is what we store, we simply trim the subnet
// from the input ROA and store it seperately.
type storedROA struct {
	Asn       string `json:"asn"`
	Prefix    string `json:"prefix"`
	MaxLength int    `json:"maxLength"`
	Ta        string `json:"ta"`
	Subnet    int
}

type storedROAWithTime struct {
	Asn       string      `bigquery:"asn" json:"asn"`
	Prefix    string      `bigquery:"prefix" json:"prefix"`
	MaxLength int         `bigquery:"maxlen" json:"maxLength"`
	Ta        string      `bigquery:"ta" json:"ta"`
	Subnet    int         `bigquery:"mask"`
	Times     []time.Time `bigquery:"inserttimes"`
}

var (
	client *bigquery.Client
)

var (
	// The actual data is used at 'hotsed-routinator.rarc.net'. but that appears
	// to not work well? Use the docs.as701.net proxy path instead.
	roaURL = "https://hosted-routinator-east.rarc.net/json"
	// roaURL = "https://docs.as701.net/roa/update/"
	// The above SHOULD work, it does not. Proxy the requests through
	// an endpoint that we know works properly, external to cloud.
	projectLocation = "us-central2"
	updateCooldown  = 50 * time.Minute
	mergeOnCond     = "b.asn = arr.asn AND b.maxlen = arr.maxlen AND b.prefix = arr.prefix AND b.ta = arr.ta AND b.mask = arr.mask"
	queryASN        = "SELECT asn, prefix, mask, maxlen, ta, inserttimes FROM public-routing-data-backup.historical.roas_arr WHERE asn = @asn"
	queryPrefix     = "SELECT asn, prefix, mask, maxlen, ta, inserttimes FROM public-routing-data-backup.historical.roas_arr WHERE prefix = @prefix AND mask = @mask"
	queryBoth       = "SELECT asn, prefix, mask, maxlen, ta, inserttimes FROM public-routing-data-backup.historical.roas_arr WHERE asn = @asn AND prefix = @prefix AND mask = @mask"
)

func initBQClient(ctx context.Context) (*bigquery.Client, error) {
	if bqHost := os.Getenv("BIGQUERY_EMULATOR_HOST"); bqHost != "" {
		return bigquery.NewClient(ctx, "public-routing-data-backup",
			option.WithEndpoint("http://"+bqHost),
			option.WithoutAuthentication(),
		)
	}
	return bigquery.NewClient(ctx, bigquery.DetectProjectID)
}

func main() {
	flag.Parse()
	// set http port
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
		glog.V(2).Infof("using default port: %v", port)
	}

	// open bigquery connection
	var err error
	client, err = initBQClient(context.Background())
	if err != nil {
		glog.Fatalln(err)
	}

	client.Location = projectLocation

	http.HandleFunc("/update", pullToDB)
	http.HandleFunc("/", mainPage)
	http.HandleFunc("/hsts", hsts)
	//http.HandleFunc("/aaaaaaaaaaaaaaaa", movefromoldtonew.Main)
	glog.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

func hsts(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("strict-transport-security", "max-age=2629800")
	// If the X-Forwarded-Proto was set upstream as HTTP, then the request came in without TLS.
	if r.Header.Get("X-Forwarded-Proto") == "http" || r.URL.Scheme != "https" {
		r.URL.Scheme = "https"
		http.Redirect(w, r, r.URL.String(), http.StatusMovedPermanently)
	}
}

type PageData struct {
	ASN          string
	Prefix       string
	ParseCIDR    bool
	HasResults   bool
	Results      []*RenderedROA
	ErrorMessage string
}

type RenderedROA struct {
	ASN             string
	FullPrefixRange string
	TrustAnchor     string
	DateRanges      []string
	RFC3339Times    []string
	UnixTimes       []int64
}

type dateRange struct {
	start time.Time
	end   time.Time
}

// computeAvailabilityRanges groups sorted observation timestamps into contiguous runs
func computeAvailabilityRanges(times []time.Time, gapThreshold time.Duration) []string {
	if len(times) == 0 {
		return nil
	}

	var ranges []dateRange
	current := dateRange{start: times[0], end: times[0]}

	for i := 1; i < len(times); i++ {
		t := times[i]
		diff := t.Sub(current.end)
		if diff <= gapThreshold {
			current.end = t
		} else {
			ranges = append(ranges, current)
			current = dateRange{start: t, end: t}
		}
	}
	ranges = append(ranges, current)

	var formatted []string
	for _, r := range ranges {
		startStr := r.start.Format("Jan 2 2006")
		endStr := r.end.Format("Jan 2 2006")
		if startStr == endStr {
			formatted = append(formatted, startStr)
		} else {
			formatted = append(formatted, fmt.Sprintf("%s -> %s", startStr, endStr))
		}
	}
	return formatted
}

func mainPage(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	w.Header().Add("strict-transport-security", "max-age=2629800")

	tmpl, err := template.New("index").Parse(indexHTML)
	if err != nil {
		glog.Errorln(err)
		return
	}

	rawASN := r.FormValue("asn")
	rawPrefix := r.FormValue("prefix")
	rawParseCIDR := r.FormValue("parsecidr")

	// If no query criteria submitted at all, simply serve initial empty query card
	if r.Method != http.MethodPost && rawASN == "" && rawPrefix == "" {
		tmpl.Execute(w, PageData{})
		return
	}

	normASN, err := normalizeASN(rawASN)
	if err != nil {
		ErrorHandler(w, r, http.StatusBadRequest, "Invalid ASN format", err)
		return
	}

	normPrefix, err := normalizePrefix(rawPrefix)
	if err != nil {
		ErrorHandler(w, r, http.StatusBadRequest, "Invalid IP Prefix format", err)
		return
	}

	input := inputROA{
		Asn:       normASN,
		Prefix:    normPrefix,
		ParseCIDR: rawParseCIDR,
	}

	if input.ParseCIDR != "" && input.Prefix != "" {
		_, n, err := net.ParseCIDR(input.Prefix)
		if err != nil {
			ErrorHandler(w, r, http.StatusBadRequest, "Failed to parse CIDR", err)
			return
		}
		input.Prefix = n.String()
	}

	inputStore, err := convInToStored(input)
	if err != nil {
		ErrorHandler(w, r, http.StatusBadRequest, "Invalid input prefix", err)
		return
	}

	var hasASN, hasPrefix bool
	if inputStore.Asn != "" {
		hasASN = true
	}

	if inputStore.Prefix != "" && inputStore.Subnet != 0 {
		hasPrefix = true
	}

	if !hasASN && !hasPrefix {
		ErrorHandler(w, r, http.StatusBadRequest, "Please provide either an ASN or an IP Prefix", errors.New("missing query criteria"))
		return
	}

	glog.V(2).Infoln(input)

	var query *bigquery.Query
	switch {
	case hasASN && !hasPrefix:
		query = client.Query(queryASN)

	case !hasASN && hasPrefix:
		query = client.Query(queryPrefix)
	case hasASN && hasPrefix:
		query = client.Query(queryBoth)
	}
	query.Parameters = []bigquery.QueryParameter{
		{
			Name:  "asn",
			Value: inputStore.Asn,
		},
		{
			Name:  "prefix",
			Value: inputStore.Prefix,
		},
		{
			Name:  "mask",
			Value: inputStore.Subnet,
		},
	}

	it, err := query.Read(ctx)
	if err != nil {
		ErrorHandler(w, r, 500, "Error with query", err)
		return
	}

	var resultsarr pb.ResultArr
	var pageResults []*RenderedROA

	for {
		var row []bigquery.Value
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			ErrorHandler(w, r, 500, "Error with query", err)
			return
		}

		var intime []time.Time
		var buf = row[5].([]bigquery.Value)

		for _, t := range buf {
			intime = append(intime, t.(time.Time))
		}

		// Sort observation timestamps ascending for accurate range grouping
		sort.Slice(intime, func(i, j int) bool {
			return intime[i].Before(intime[j])
		})

		var results = pb.ResultsFromDB{
			ASN:    row[0].(string),
			Prefix: row[1].(string),
			Mask:   int32(row[2].(int64)),
			Maxlen: int32(row[3].(int64)),
			Ta:     row[4].(string),
		}

		results.Fullprefix = fmt.Sprintf("%v/%v", results.Prefix, results.Mask)
		switch {
		case results.Maxlen != results.Mask:
			results.Fullprefixrange = fmt.Sprintf("%v/%v => %v",
				results.Prefix, results.Mask, results.Maxlen)
		case results.Maxlen == results.Mask:
			results.Fullprefixrange = fmt.Sprintf("%v/%v", results.Prefix, results.Mask)
		}

		var rfcTimes []string
		var unixTimes []int64

		for _, i := range intime {
			unixVal := i.Unix()
			rfcVal := i.Format(time.RFC3339)
			results.Unixtimearr = append(results.Unixtimearr, unixVal)
			results.RFC3339Timearr = append(results.RFC3339Timearr, rfcVal)

			rfcTimes = append(rfcTimes, rfcVal)
			unixTimes = append(unixTimes, unixVal)
		}

		resultsarr.Results = append(resultsarr.Results, &results)

		// Group consistent observation ranges (using a 26-hour gap threshold to allow for minor Cron shifts)
		dateRanges := computeAvailabilityRanges(intime, 26*time.Hour)

		pageResults = append(pageResults, &RenderedROA{
			ASN:             results.ASN,
			FullPrefixRange: results.Fullprefixrange,
			TrustAnchor:     results.Ta,
			DateRanges:      dateRanges,
			RFC3339Times:    rfcTimes,
			UnixTimes:       unixTimes,
		})
	}

	// Requirement 6: URL parameter '?json' bypasses HTML formatting
	if _, hasJSON := r.Form["json"]; hasJSON || r.URL.Query().Has("json") {
		opts := protojson.MarshalOptions{Multiline: true, Indent: "  "}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(opts.Format(&resultsarr)))
		return
	}

	// Render complete structured HTML results page
	pData := PageData{
		ASN:        rawASN,
		Prefix:     rawPrefix,
		ParseCIDR:  rawParseCIDR != "",
		HasResults: true,
		Results:    pageResults,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, pData)
}

// normalizeASN validates and prepends AS to raw autonomous system number strings
func normalizeASN(asn string) (string, error) {
	asn = strings.TrimSpace(asn)
	if asn == "" {
		return "", nil
	}

	// If purely numeric, prepend AS
	if _, err := strconv.Atoi(asn); err == nil {
		return "AS" + asn, nil
	}

	// If starts with AS (case-insensitive) and followed by digits
	upper := strings.ToUpper(asn)
	if strings.HasPrefix(upper, "AS") {
		if _, err := strconv.Atoi(upper[2:]); err == nil {
			return upper, nil
		}
	}

	return "", fmt.Errorf("invalid ASN format (expected numeric or AS####): %q", asn)
}

// normalizePrefix validates CIDR or automatically computes network for bare IP addresses
func normalizePrefix(prefix string) (string, error) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return "", nil
	}

	// If a bare IP Address is provided, assume /24 or /48 and normalize to base network
	if ip := net.ParseIP(prefix); ip != nil {
		var mask string
		if ip.To4() != nil {
			mask = "/24"
		} else {
			mask = "/48"
		}
		_, n, err := net.ParseCIDR(prefix + mask)
		if err != nil {
			return "", fmt.Errorf("failed to compute CIDR for bare IP %q: %w", prefix, err)
		}
		return n.String(), nil
	}

	// Otherwise, verify it's a valid CIDR netblock
	if _, _, err := net.ParseCIDR(prefix); err != nil {
		return "", fmt.Errorf("invalid IP Prefix or CIDR netblock %q: %w", prefix, err)
	}

	return prefix, nil
}

// convert input data into stored data
func convInToStored(i inputROA) (storedROA, error) {
	// shut up I know its not correct terminology
	ipandmask := strings.Split(i.Prefix, "/")

	var mask int
	var err error
	if len(ipandmask) == 2 {
		mask, err = strconv.Atoi(ipandmask[1])
		if err != nil {
			return storedROA{}, fmt.Errorf("invalid mask %q: %w", ipandmask[1], err)
		}
	} else if i.Prefix != "" {
		return storedROA{}, fmt.Errorf("invalid prefix format (expected ip/mask): %q", i.Prefix)
	}

	return storedROA{
		Asn:       i.Asn,
		Prefix:    ipandmask[0],
		MaxLength: i.MaxLength,
		Ta:        i.Ta,
		Subnet:    mask,
	}, nil
}

func verifyOIDCToken(ctx context.Context, r *http.Request) error {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return errors.New("missing Authorization header")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return errors.New("invalid Authorization header format")
	}
	token := parts[1]

	expectedAudience := os.Getenv("SCHEDULE_AUDIENCE")
	if expectedAudience == "" {
		expectedAudience = "https://" + r.Host + r.URL.Path
	}

	expectedSA := os.Getenv("SCHEDULE_SERVICE_ACCOUNT")
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")

	payload, err := idtoken.Validate(ctx, token, expectedAudience)
	if err != nil {
		return fmt.Errorf("token validation failed: %w", err)
	}

	email, ok := payload.Claims["email"].(string)
	if !ok {
		return errors.New("email claim missing in token")
	}

	if expectedSA != "" {
		if email != expectedSA {
			return fmt.Errorf("unauthorized service account: %s (expected %s)", email, expectedSA)
		}
	} else if projectID != "" {
		allowedSuffix := "@" + projectID + ".iam.gserviceaccount.com"
		if !strings.HasSuffix(email, allowedSuffix) {
			return fmt.Errorf("service account %s does not belong to project %s", email, projectID)
		}
	} else {
		return errors.New("neither SCHEDULE_SERVICE_ACCOUNT nor GOOGLE_CLOUD_PROJECT is set to verify caller identity")
	}

	return nil
}

func pullToDB(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Appengine-Cron") == "true" {
		// Trusted GAE Cron
	} else {
		if err := verifyOIDCToken(r.Context(), r); err != nil {
			TextErrorHandler(w, http.StatusForbidden, "Forbidden: OIDC verification failed", err)
			return
		}
	}

	// See if there has been an update within 50 mins by checking table metadata
	meta, err := client.Dataset("historical").Table("roas_arr").Metadata(context.Background())
	if err != nil {
		glog.Errorln("Cant get last edit time (table metadata): ", err)
	} else {
		lastIn := meta.LastModifiedTime
		if lastIn.Add(updateCooldown).After(time.Now()) {
			glog.V(2).Infoln("Record added in last 50 mins")
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "Skipped: already updated in last 50 mins")
			return
		}
	}

	glog.V(1).Infoln("starting update")

	origIn, err := downloadRARC()
	if err != nil {
		TextErrorHandler(w, 500, "Error parsing JSON", err)
		return
	}
	ctx := context.Background()

	schema, err := bigquery.InferSchema(storedROAWithTime{})
	if err != nil {
		TextErrorHandler(w, 500, "failed to infer schema", err)
		return
	}

	schema = schema.Relax()

	//query and dump to map
	var stored = make(map[string]struct{})

	currentQuery := client.Query(`SELECT asn, ta, prefix, mask, maxlen FROM public-routing-data-backup.historical.roas_arr`)
	job, err := currentQuery.Run(ctx)
	if err != nil {
		TextErrorHandler(w, 500, "Error running query", err)
		return
	}

	status, err := job.Wait(ctx)
	if err != nil {
		TextErrorHandler(w, 500, "Error waiting for query job", err)
		return
	}
	if err := status.Err(); err != nil {
		TextErrorHandler(w, 500, "Query job failed", err)
		return
	}

	it, err := job.Read(ctx)
	if err != nil {
		TextErrorHandler(w, 500, "Error reading query results", err)
		return
	}

	for {
		var row []bigquery.Value
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			TextErrorHandler(w, 500, "Error iterating query results", err)
			return
		}

		stored[xhashes.MD5(fmt.Sprint(pb.ResultsFromDB{
			ASN:    row[0].(string),
			Ta:     row[1].(string),
			Prefix: row[2].(string),
			Mask:   int32(row[3].(int64)), // google, you are
			Maxlen: int32(row[4].(int64)), // disgusting
		}))] = struct{}{}
	}

	now := []time.Time{time.Now()}
	var id int
	var in []*storedROAWithTime
	for _, i := range origIn.Roas {
		id++
		// shut up I know its not correct terminology
		ipandmask := strings.Split(i.Prefix, "/")
		if len(ipandmask) != 2 {
			glog.Error("Skipping invalid prefix (missing slash): ", i.Prefix)
			continue
		}
		mask, err := strconv.Atoi(ipandmask[1])
		if err != nil {
			glog.Error("Skipping invalid prefix (bad mask): ", i.Prefix, " err: ", err)
			continue
		}

		/*in = append(in, storedROA{
			Asn:       i.Asn,
			Prefix:    ipandmask[0],
			MaxLength: i.MaxLength,
			Ta:        i.Ta,
			Subnet:    mask,
		})*/

		in = append(in, &storedROAWithTime{i.Asn, ipandmask[0], i.MaxLength, i.Ta, mask, now})

		//go glog.V(2).Infoln(debug)
		//debug++

	}

	glog.V(2).Infoln("making buf table")
	// make buf table

	err = client.Dataset("historical").Table("buf").Delete(ctx)
	if err != nil {
		glog.Errorln("Error Deleting buf: ", err)
		err = nil
	}
	err = client.Dataset("historical").Table("buf").Create(ctx,
		&bigquery.TableMetadata{Schema: schema})
	if err != nil {
		TextErrorHandler(w, 500, "error creating buf table", err)
		return
	}

	tmpinserter := client.Dataset("historical").Table("buf").Inserter()

	var divided [][]*storedROAWithTime
	chunk := 950
	for i := 0; i < len(in); i += chunk {
		end := i + chunk
		if end > len(in) {
			end = len(in)
		}
		divided = append(divided, in[i:end])
	}
	for _, i := range divided {
		if len(i) == 0 {
			glog.Errorln("Divided array had len 0")
			break
		}
		err = tmpinserter.Put(ctx, i)
		if err != nil {
			TextErrorHandler(w, 500, "error putting updates into buf", err)
			return
		}
	}

	// now make one plus one equal 2
	// historical-roas.historical.roas_arr
	query := client.Query(fmt.Sprintf(
		`MERGE historical.roas_arr arr
 	 USING historical.buf b
	    ON 	%s
	  WHEN MATCHED THEN
 		UPDATE SET inserttimes = ARRAY_CONCAT(b.inserttimes, arr.inserttimes)
 	  WHEN NOT MATCHED BY TARGET THEN
		INSERT (asn, maxlen, prefix, ta, mask, inserttimes) VALUES (b.asn, b.maxlen, b.prefix, b.ta, b.mask, b.inserttimes)`, mergeOnCond))
	job, err = query.Run(ctx)
	if err != nil {
		TextErrorHandler(w, 500, "Error running MERGE query", err)
		return
	}
	status, err = job.Wait(ctx)
	if err != nil {
		TextErrorHandler(w, 500, "Error waiting for MERGE job", err)
		return
	}
	if err := status.Err(); err != nil {
		TextErrorHandler(w, 500, "MERGE job failed", err)
		return
	}

	_, err = job.Read(ctx)
	if err != nil {
		TextErrorHandler(w, 500, "Error reading MERGE results", err)
		return
	}

	glog.V(1).Infoln("done updating")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Update successful")
}

func downloadRARC() (*inputROAArr, error) {
	var form inputROAArr

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Get(roaURL)
	if err != nil {
		glog.Infof("failed attempting to download %q in downloadRARC, err: %v", roaURL, err)
		return &form, err
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return &form, fmt.Errorf("failed to read response body: %w", err)
	}
	jsonIn := buf.String()

	if resp.StatusCode != http.StatusOK {
		previewLen := 500
		if len(jsonIn) < previewLen {
			previewLen = len(jsonIn)
		}
		bodyPreview := jsonIn[:previewLen]
		return &form, fmt.Errorf("unexpected HTTP status %d from %s (body preview: %q)", resp.StatusCode, roaURL, bodyPreview)
	}

	err = json.Unmarshal([]byte(jsonIn), &form)
	if err != nil {
		previewLen := 500
		if len(jsonIn) < previewLen {
			previewLen = len(jsonIn)
		}
		bodyPreview := jsonIn[:previewLen]
		return &form, fmt.Errorf("JSON unmarshal failed: %w (body preview: %q)", err, bodyPreview)
	}

	return &form, nil
}

// ErrorHandler is a function to handle HTTP errors
// copied from imgsrvr, slightly different formatting
func ErrorHandler(resp http.ResponseWriter, req *http.Request, status int, alert string, err error) {
	glog.Errorln(err)
	resp.WriteHeader(status)
	glog.Error("artifical http error: ", status, alert)
	fmt.Fprintf(resp, "<html><title>Error!</title><body>You have found an error! This error is of type %v. Built in alert: \n'%v',\n Would you like a <a href='https://http.cat/%v'>cat</a> or a <a href='https://httpstatusdogs.com/%v'>dog?</a></body></html>",
		status, html.EscapeString(alert), status, status)
}

// TextErrorHandler handles HTTP errors for machine callers by returning plain text.
func TextErrorHandler(w http.ResponseWriter, status int, alert string, err error) {
	glog.Errorln(err)
	glog.Error("artifical http error: ", status, alert)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	if err != nil {
		fmt.Fprintf(w, "Error %d: %s: %v\n", status, alert, err)
	} else {
		fmt.Fprintf(w, "Error %d: %s\n", status, alert)
	}
}

/*
; modified to save storage

DROP TABLE IF EXISTS
  `public-routing-data-backup`.`historical`.`roas_arr`;
CREATE TABLE
  `public-routing-data-backup`.`historical`.`roas_arr`( asn STRING,
    prefix STRING,
    maxlen INT64,
    ta STRING,
    mask INT64,
    inserttimes ARRAY<TIMESTAMP>)
CLUSTER BY
  prefix,
  mask,
  asn;

*/
