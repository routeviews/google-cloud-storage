package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	converter "github.com/routeviews/google-cloud-storage/pkg/mrt_converter"
	log "github.com/sirupsen/logrus"

	"cloud.google.com/go/storage"
)

type server struct {
	gcsCli    *storage.Client
	dstBucket string
}

func newServer(ctx context.Context, cli *storage.Client, dstBucket string) (*server, error) {
	if dstBucket == "" {
		return nil, fmt.Errorf("destination bucket is not specified")
	}
	if cli == nil {
		return nil, fmt.Errorf("nil GCS client")
	}
	return &server{
		gcsCli:    cli,
		dstBucket: dstBucket,
	}, nil
}

type gcsPubSubEvent struct {
	Message struct {
		Attributes struct {
			Bucket    string `json:"bucketId"`
			Object    string `json:"objectId"`
			EventType string `json:"eventType"`
		} `json:"attributes,omitempty"`
		MessageID string `json:"messageId"`
	} `json:"message"`
	Subscription string `json:"subscription"`
}

// archiveUploadHandler handles any new object changes from the archive bucket.
// It will not return an HTTP error because all errrors are fatal and should
// not be retried.
func (s *server) archiveUploadHandler(w http.ResponseWriter, r *http.Request) {
	var msg gcsPubSubEvent
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Errorf("ioutil.ReadAll: %v", err)
		return
	}

	if err := json.Unmarshal(body, &msg); err != nil {
		log.Infof("json.Unmarshal: %v", err)
		return
	}

	// The archive server will set metadata of project source after the object
	// is created, so we will look for metadata update messages instead of
	// object creations.
	if msg.Message.Attributes.EventType != "OBJECT_METADATA_UPDATE" {
		log.Infof("Skipped non-'OBJECT_METADATA_UPDATE' msg: id %s, type %s", msg.Message.MessageID, msg.Message.Attributes.EventType)
		return
	}

	log.WithFields(log.Fields{
		"bucket":    msg.Message.Attributes.Bucket,
		"object":    msg.Message.Attributes.Object,
		"messageID": msg.Message.MessageID,
	}).Info("Converting archive")
	err = converter.ProcessMRTArchive(r.Context(), s.gcsCli, &converter.Config{
		SrcBucket: msg.Message.Attributes.Bucket,
		SrcObject: msg.Message.Attributes.Object,
		DstBucket: s.dstBucket,
	})
	if err != nil {
		log.WithFields(log.Fields{
			"dstBucket": s.dstBucket,
			"object":    msg.Message.Attributes.Object,
		}).Errorf("converter.ProcessMRTArchive: %v", err)
		w.Write([]byte(fmt.Sprintf("converter.ProcessMRTArchive: %v", err)))
		return
	}
	log.WithFields(log.Fields{
		"bucket":    msg.Message.Attributes.Bucket,
		"dstBucket": s.dstBucket,
		"object":    msg.Message.Attributes.Object,
		"messageID": msg.Message.MessageID,
	}).Info("Archive converted")
}

func main() {
	ctx := context.Background()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Infof("Defaulting to port %s", port)
	}
	cli, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("storage.NewClient: %v", err)
	}

	srvr, err := newServer(ctx, cli, os.Getenv("BIGQUERY_BUCKET"))
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", srvr.archiveUploadHandler)
	log.Printf("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
