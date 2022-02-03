package main

import (
	"context"
	"flag"
	"net/http"
	"os"

	"cloud.google.com/go/storage"
	"github.com/golang/glog"

	datatransfer "cloud.google.com/go/bigquery/datatransfer/apiv1"
	bqtransfer "github.com/routeviews/google-cloud-storage/pkg/bq_transfer"
)

var (
	project  = flag.String("project", "public-routing-data-backup", "Project that contains public routing data.")
	location = flag.String("location", "US", "Location of the bigquery dataset.")
	dataset  = flag.String("dataset", "historical_routing_data", "Dataset that stores all routing updates.")
	table    = flag.String("table", "updates", "Table that stores all routing updates.")
	bucket   = flag.String("bucket", "routeviews-bigquery", "GCS bucket that saves all MRT archives.")
)

func main() {
	flag.Parse()

	ctx := context.Background()

	sc, err := storage.NewClient(ctx)
	if err != nil {
		glog.Exit(err)
	}
	defer sc.Close()

	dc, err := datatransfer.NewClient(ctx)
	if err != nil {
		glog.Exit(err)
	}
	defer dc.Close()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		glog.Infof("Defaulting to port %s", port)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if err := bqtransfer.Transfer(ctx, sc, dc, &bqtransfer.TransferParams{
			Project:  *project,
			Location: *location,
			Dataset:  *dataset,
			Table:    *table,
			Bucket:   *bucket,
		}); err != nil {
			glog.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}
	})
	glog.Infof("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		glog.Fatal(err)
	}

	// TODO: update current month of transfer to every 15 minutes.
}
