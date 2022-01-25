package main

import (
	"context"
	"flag"

	"cloud.google.com/go/storage"
	"github.com/golang/glog"
	converter "github.com/routeviews/google-cloud-storage/pkg/mrt_converter"
	"google.golang.org/api/iterator"
)

var (
	srcBucket  = flag.String("src_bucket", "routeviews-archives", "GCS bucket that saves all MRT archives.")
	dstBucket  = flag.String("dst_bucket", "routeviews-bigquery", "GCS bucket that saves all parsed BGP updates.")
	rootDir    = flag.String("root_dir", "", "The directory that the converter should traverse from the source bucket. Empty means the root of the bucket.")
	numWorkers = flag.Int("num_workers", 4, "Number of concurrent workers to perform conversions.")
)

type conMgr struct {
	ConJobs chan string
}

func newConMgr(ctx context.Context, cli *storage.Client, srcBkt, dstBkt string, w int) *conMgr {
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

				glog.Infof("Converting gs://%s%s", srcBkt, obj)
				err := converter.ProcessMRTArchive(ctx, cli, &converter.Config{
					SrcBucket: srcBkt,
					DstBucket: dstBkt,
					SrcObject: obj,
				})
				if err != nil {
					glog.Fatalf("cannot convert gs://%s%s: %v", srcBkt, obj, err)
				}
				glog.Infof("gs://%s%s converted", srcBkt, obj)
			}
		}()
	}
	return m
}

func main() {
	flag.Parse()
	ctx := context.Background()

	cli, err := storage.NewClient(ctx)
	if err != nil {
		glog.Exit(err)
	}

	mgr := newConMgr(ctx, cli, *srcBucket, *dstBucket, *numWorkers)
	query := &storage.Query{Prefix: *rootDir}
	it := cli.Bucket(*srcBucket).Objects(ctx, query)
	objCount := 0
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		objCount++
		if err != nil {
			glog.Fatal(err)
		}
		mgr.ConJobs <- attrs.Name
	}
}
