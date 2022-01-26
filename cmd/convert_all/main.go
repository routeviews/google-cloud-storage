package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"

	"cloud.google.com/go/storage"
	"github.com/golang/glog"
	"google.golang.org/api/idtoken"
	"google.golang.org/api/iterator"
)

var (
	host       = flag.String("host", "", "HTTP URL of the converter address.")
	saKey      = flag.String("sa_key", "", "Service account private key.")
	srcBucket  = flag.String("src_bucket", "routeviews-archives", "GCS bucket that saves all MRT archives.")
	rootDir    = flag.String("root_dir", "", "The directory that the converter should traverse from the source bucket. Empty means the root of the bucket.")
	numWorkers = flag.Int("num_workers", 4, "Number of concurrent workers to perform conversions.")
)

const taskMsgFormat = `{
	"message": {
	  "attributes": {
		"eventType": "OBJECT_METADATA_UPDATE",
		"objectId": "%s",
		"bucketId": "%s"
	  }
	}
  }`

type conMgr struct {
	ConJobs chan string
}

func conReq(cli *http.Client, host, obj, bkt string) error {
	resp, err := cli.Post(host, "application/json", bytes.NewBuffer(
		[]byte(fmt.Sprintf(taskMsgFormat, obj, bkt))))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	msg, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if string(msg) != "" {
		return fmt.Errorf(string(msg))
	}
	return nil
}

func newConMgr(ctx context.Context, cli *http.Client, host, srcBkt string, w int) *conMgr {
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

				fmt.Printf("Converting gs://%s/%s\n", srcBkt, obj)
				if err := conReq(cli, host, obj, srcBkt); err != nil {
					glog.Fatal(err)
				}
				fmt.Printf("gs://%s/%s converted\n", srcBkt, obj)
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

	mgr := newConMgr(ctx, hc, *host, *srcBucket, *numWorkers)
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
