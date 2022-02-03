package main

import (
	"context"
	"flag"
	"net/http"
	"path/filepath"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/golang/glog"
	"google.golang.org/api/idtoken"
	"google.golang.org/api/iterator"

	converter "github.com/routeviews/google-cloud-storage/pkg/mrt_converter"
	pb "github.com/routeviews/google-cloud-storage/proto/rv"
)

var (
	host       = flag.String("host", "", "HTTP URL of the converter address.")
	saKey      = flag.String("sa_key", "", "Service account private key.")
	srcBucket  = flag.String("src_bucket", "routeviews-archives", "GCS bucket that saves all raws MRT archives.")
	dstBucket  = flag.String("dst_bucket", "routeviews-bigquery", "GCS bucket that saves all converted MRT archives.")
	rootDir    = flag.String("root_dir", "", "The directory that the converter should traverse from the source bucket. Empty means the root of the bucket.")
	numWorkers = flag.Int("num_workers", 4, "Number of concurrent workers to perform conversions.")
)

const defaultDataSource = pb.FileRequest_ROUTEVIEWS

type conMgr struct {
	ConJobs chan string
}

func newConMgr(ctx context.Context, cli *http.Client, sc *storage.Client, host, srcBkt, dstBkt string, w int) *conMgr {
	m := &conMgr{
		ConJobs: make(chan string),
	}

	for i := 0; i < w; i++ {
		go func() {
			for {
				obj, ok := <-m.ConJobs
				if !ok {
					return
				}

				dstObject := strings.Replace(obj, filepath.Ext(obj), ".gz", 1)
				if found, err := converter.ObjExists(ctx, sc, dstObject, dstBkt); err != nil {
					// Will start conversion if we can't fetch the object.
					glog.Errorf("ObjExists: %v", err)
				} else if found {
					glog.Infof("Skipped: converted archive gs://%s/%s already exists.", srcBkt, dstObject)
					continue
				}
				gcsObj := sc.Bucket(srcBkt).Object(obj)
				attrs, err := gcsObj.Attrs(ctx)
				if err != nil {
					glog.Errorf("failed to get metadata of gs://%s/%s: %v", srcBkt, obj, err)
					continue
				}
				dataSource := attrs.Metadata[converter.ProjectMetadataKey]
				if dataSource == "" {
					glog.Warningf("gs://%s/%s doesn't have project metadata; set to %s", srcBkt, obj, defaultDataSource.String())
					dataSource = defaultDataSource.String()
				}

				// Reset metadata to trigger conversion.
				_, err = gcsObj.Update(ctx, storage.ObjectAttrsToUpdate{Metadata: map[string]string{
					converter.ProjectMetadataKey: dataSource,
				}})
				if err != nil {
					glog.Errorf("failed to update metadata of gs://%s/%s: %v", srcBkt, obj, err)
					continue
				}
				glog.Infof("Conversion request sent: gs://%s/%s", srcBkt, obj)
			}
		}()
	}
	return m
}

func main() {
	flag.Parse()
	ctx := context.Background()

	sc, err := storage.NewClient(ctx)
	if err != nil {
		glog.Exit(err)
	}

	hc, err := idtoken.NewClient(ctx, *host, idtoken.WithCredentialsFile(*saKey))
	if err != nil {
		glog.Exit(err)
	}

	mgr := newConMgr(ctx, hc, sc, *host, *srcBucket, *dstBucket, *numWorkers)
	query := &storage.Query{Prefix: *rootDir}
	it := sc.Bucket(*srcBucket).Objects(ctx, query)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			glog.Fatal(err)
		}
		mgr.ConJobs <- attrs.Name
	}
}
