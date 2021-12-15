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

func (m *server) readAll(ctx context.Context, object, bucket string) ([]byte, error) {
	r, err := m.gcsCli.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("NewReader(gs://%s/%s): %v", bucket, object, err)
	}
	content, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadAll: %v", err)
	}
	return content, nil
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

	if msg.Message.Attributes.EventType != "OBJECT_FINALIZE" {
		log.Infof("Skipped non-'OBJECT_FINALIZE' msg: id %s, type %s", msg.Message.MessageID, msg.Message.Attributes.EventType)
		return
	}

	log.WithFields(log.Fields{
		"bucket":    msg.Message.Attributes.Bucket,
		"object":    msg.Message.Attributes.Object,
		"messageID": msg.Message.MessageID,
	}).Info("Converting archive")
	content, err := s.readAll(r.Context(), msg.Message.Attributes.Object, msg.Message.Attributes.Bucket)
	if err != nil {
		log.WithFields(log.Fields{
			"bucket":    msg.Message.Attributes.Bucket,
			"object":    msg.Message.Attributes.Object,
			"messageID": msg.Message.MessageID,
		}).Errorf("m.readAll: %v", err)
		return
	}
	err = converter.ProcessMRTArchive(r.Context(), s.gcsCli, msg.Message.Attributes.Object, s.dstBucket, content)
	if err != nil {
		log.WithFields(log.Fields{
			"dstBucket": s.dstBucket,
			"object":    msg.Message.Attributes.Object,
		}).Errorf("converter.ProcessMRTArchive: %v", err)
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
