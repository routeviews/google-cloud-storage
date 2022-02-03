// Package bqtransfer provides utilities to transfer large amound of converted
// routing archives to BigQuery using Data Transfer Service.
package bqtransfer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/types/known/structpb"

	datatransfer "cloud.google.com/go/bigquery/datatransfer/apiv1"
	dpb "google.golang.org/genproto/googleapis/cloud/bigquery/datatransfer/v1"
)

var queryInterval = time.Second

type TransferParams struct {
	Project  string
	Location string
	Dataset  string
	Table    string
	Bucket   string
}

// fetchMonthDirs traverse all directories and finds the month directories (i.e.
// the directories ended with YYYY.MM only contain a month of data.)
func fetchMonthDirs(ctx context.Context, cli *storage.Client, bkt string) ([]string, error) {
	queries := []*storage.Query{{Delimiter: "/"}}
	var res []string
	for len(queries) > 0 {
		query := queries[0]
		queries = queries[1:]
		it := cli.Bucket(bkt).Objects(ctx, query)

		for {
			attrs, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, err
			}
			dirs := strings.Split(attrs.Prefix, "/")
			// Minus 2 because the last character of the path is a slash and
			// thus the last element will be "".
			if len(dirs) < 2 {
				fmt.Println(query.Prefix, attrs.Prefix)
				glog.Errorf("prefix shorter than 2 levels: %s", attrs.Prefix)
				continue
			}
			parent := dirs[len(dirs)-2]
			// Check if the value is in a YYYY.MM format.
			if _, err := time.Parse("2006.01", parent); err == nil {
				res = append(res, attrs.Prefix)
			} else if attrs.Prefix != "" {
				queries = append(queries, &storage.Query{Prefix: attrs.Prefix, Delimiter: "/"})
			}
		}
	}
	return res, nil
}

// fetchCoveredDirs finds the GCS prefixes that are already covered.
func fetchCoveredDirs(ctx context.Context, cli *datatransfer.Client, project string) (map[string]string, error) {
	req := &dpb.ListTransferConfigsRequest{
		Parent: fmt.Sprintf("projects/%s", project),
	}
	res := make(map[string]string)
	it := cli.ListTransferConfigs(ctx, req)
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		res[resp.GetParams().GetFields()["data_path_template"].GetStringValue()] = resp.GetName()
	}

	return res, nil
}

func makeTransferConfig(cfg *TransferParams, dir, pattern string) *dpb.TransferConfig {
	return &dpb.TransferConfig{
		DisplayName:  dir,
		DataSourceId: "google_cloud_storage",
		Destination: &dpb.TransferConfig_DestinationDatasetId{
			DestinationDatasetId: cfg.Dataset,
		},
		Schedule: "every day 00:00",
		EmailPreferences: &dpb.EmailPreferences{
			// TODO: flip this to true when transfers stablize.
			EnableFailureEmail: false,
		},
		Params: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"destination_table_name_template": structpb.NewStringValue(cfg.Table),
				"data_path_template":              structpb.NewStringValue(pattern),
				"file_format":                     structpb.NewStringValue("JSON"),
				"max_bad_records":                 structpb.NewStringValue("0"),
				"skip_leading_rows":               structpb.NewStringValue("0"),
				"write_disposition":               structpb.NewStringValue("APPEND"),
			},
		},
	}
}

func createTransferRuns(ctx context.Context, cli *datatransfer.Client, dirs []string, covered map[string]string, cfg *TransferParams) error {
	for _, dir := range dirs {
		pattern := fmt.Sprintf("gs://%s/%s*/*.gz", cfg.Bucket, dir)
		if cid, ok := covered[pattern]; ok {
			glog.Warningf("Skipped: config %s is covering %s", cid, pattern)
			continue
		}
		// Create a new transfer config if not found.
		err := backoff.Retry(func() error {
			resp, err := cli.CreateTransferConfig(ctx, &dpb.CreateTransferConfigRequest{
				Parent:         fmt.Sprintf("projects/%s/locations/%s", cfg.Project, cfg.Location),
				TransferConfig: makeTransferConfig(cfg, dir, pattern),
			})
			if err != nil {
				glog.Warningf("attempt to transfer %s failed: %v", pattern, err)
				return err
			}
			glog.Infof("Created transfer config %s for %s\n", resp.Name, pattern)
			return nil
		}, backoff.WithMaxRetries(backoff.NewConstantBackOff(30*time.Second), 3))
		if err != nil {
			return err
		}

		// Avoid exceeding DTS API access quota.
		time.Sleep(queryInterval)
	}
	return nil
}

// Transfer transfers all archives from a bucket to a BigQuery table.
func Transfer(ctx context.Context, sc *storage.Client, dc *datatransfer.Client, params *TransferParams) error {
	monthPrefixes, err := fetchMonthDirs(ctx, sc, params.Bucket)
	if err != nil {
		return fmt.Errorf("fetchMonthDirs(%s): %v", params.Bucket, err)
	}

	covered, err := fetchCoveredDirs(ctx, dc, params.Project)
	if err != nil {
		return fmt.Errorf("fetchCoveredDirs(%s): %v", params.Project, err)
	}
	glog.Infof("Found %d month dirs, %d covered dirs", len(monthPrefixes), len(covered))

	return createTransferRuns(ctx, dc, monthPrefixes, covered, params)
}
