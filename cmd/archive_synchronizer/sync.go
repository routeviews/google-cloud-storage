package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/storage"
	"github.com/routeviews/google-cloud-storage/pkg/auth"
	"github.com/routeviews/google-cloud-storage/pkg/synchronizer"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	pb "github.com/routeviews/google-cloud-storage/proto/rv"
)

var (
	uploadServer = flag.String("upload_server", "rv-server-cgfq4yjmfa-uc.a.run.app:443", "Upload service host:port.")
	bucket       = flag.String("bucket", "routeviews-archives", "GCS bucket that stores these archives.")

	minLapse = flag.Duration("min_lapse", time.Hour,
		"Minimum time to wait before uploading a missing archive.")
	maxLapse = flag.Duration("max_lapse", 15*24*time.Hour,
		"Maximum time that we look back for a missing archive.")

	saPath = flag.String("sa_key", "", "Service account key for the upload server.")
	useTLS = flag.Bool("use_tls", true, "Enable TLS if true. Disable TLS if testing with a local instance.")

	runHTTP = flag.Bool("http_server", true, `If true, this will be run as an
	 HTTP server, and users can trigger sync by accessing path '/'. Otherwise,
	 it will run a one-off synchronization.`)
)

func main() {
	flag.Parse()

	ctx := context.Background()

	// Setup gRPC connection.
	var gc *grpc.ClientConn
	var err error
	if *useTLS {
		gc, err = auth.NewAuthConn(ctx, *uploadServer, *saPath)
	} else {
		gc, err = auth.InsecureConn(*uploadServer)
	}
	if err != nil {
		log.Fatalf("failed to establish gRPC connection: %v", err)
	}
	defer gc.Close()

	// Setup GCS client.
	sc, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("storage.NewClient: %v", err)
	}
	defer sc.Close()

	sr, err := synchronizer.New(&synchronizer.Config{
		FTPServer: os.Getenv("FTP_SERVER"),
		FTPUser:   os.Getenv("FTP_USERNAME"),
		FTPPass:   os.Getenv("FTP_PASSWORD"),

		GCSCli:          sc,
		UploadServerCli: pb.NewRVClient(gc),
		ArchiveBucket:   *bucket,
		HTTPRoot:        "http://routeviews.org",
	})
	if err != nil {
		log.Fatal(err)
	}

	if !*runHTTP {
		now := time.Now()
		start := now.Add(-*maxLapse)
		end := now.Add(-*minLapse)
		log.Infof("Start synchronization from %s to %s",
			start.Format(time.RFC3339), end.Format(time.RFC3339))
		if err := sr.Sync(ctx, start, end); err != nil {
			log.Fatal(err)
		}
		return
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Infof("Defaulting to port %s", port)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		start := now.Add(-*maxLapse)
		end := now.Add(-*minLapse)
		log.Infof("Start synchronization from %s to %s",
			start.Format(time.RFC3339), end.Format(time.RFC3339))
		if err := sr.Sync(ctx, start, end); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}
	})
	log.Infof("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
