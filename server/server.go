// Package main is the grpc server which accepts file storage requests,
// Files are stored in cloud-storage, they may be converted and added to
// BigQuery tables as well, depending upon the request.
package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"

	"cloud.google.com/go/storage"
	"github.com/morrowc/rv/proto/rv"
	pb "github.com/morrowc/rv/proto/rv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	// https://cloud.google.com/storage/docs/reference/libraries#client-libraries-install-go
	// TODO(morrowc): Sort out organization privilege problems to create a service account key.
	// Be sure to have the JSON authentication bits in env(GOOGLE_APPLICATION_CREDENTIALS)
	projectID = "1071922449970"
)

var (
	port   = flag.Int("port", 9876, "Port on which gRPC connections will come.")
	apiKey = flag.String("apikey", "", "API Key to use in cloud storage operations.")
	bucket = flag.String("bucket", "archive-routeviews", "Cloud storage bucket name.")

	// TODO(morrowc): find a method to define the TLS certificate to be used.
)

type RV struct {
	apiKey string
	bucket string
	sc     *storage.Client
	rv.UnimplementedRVServer
}

// newRV creates and returns a proper RV object.
func newRV(key string, bucket string) (RV, error) {
	c, err := storage.NewClient(context.Background())
	if err != nil {
		return RV{}, fmt.Errorf("failed to create storage client: %v", err)
	}

	return RV{
		apiKey: key,
		bucket: bucket,
		sc:     c,
	}, nil
}

// Store the file to cloud storage.
func (r RV) handleRPKIRarc(ctx context.Context, resp *pb.FileResponse, fn string, c []byte) (*pb.FileResponse, error) {
	// Store the file content to the
	wc := r.sc.Bucket(r.bucket).Object(fn).NewWriter(ctx)
	if _, err := io.Copy(wc, bytes.NewReader(c)); err != nil {
		resp.Status = pb.FileResponse_FAIL
		return resp, fmt.Errorf("failed copying content to destination: %s/%s: %v", r.bucket, fn, err)
	}
	resp.Status = pb.FileResponse_SUCCESS
	return resp, nil
}

// FileUpload collects a file and handles it according to the appropriate rules.
//  FileRequeasts must have:
//    filename
//    checksum
//    content
//    project
//
// If any of these is missing the requset is invalid.
//
func (r RV) FileUpload(ctx context.Context, req *pb.FileRequest) (*pb.FileResponse, error) {
	resp := &pb.FileResponse{}

	fn := req.GetFilename()
	content := req.GetContent()
	proj := req.GetProject()
	sum := req.GetMd5Sum()
	if len(content) < 1 || proj == pb.FileRequest_UNKNOWN || len(fn) < 1 {
		resp.Status = pb.FileResponse_FAIL
		return nil, errors.New("base requirements for FileRequest unmet")
	}

	// validate that content checksum matches the requseted checksum.
	ts := md5.Sum(content)
	tsString := hex.EncodeToString(ts[:])
	if tsString != sum {
		resp.Status = pb.FileResponse_FAIL
		return nil, fmt.Errorf("checksum failure req(%q) != calc(%q)", sum, tsString)
	}

	// Process the content based upon project requirements.
	switch {
	case proj == pb.FileRequest_ROUTEVIEWS:
	case proj == pb.FileRequest_RIPE_RIS:
	case proj == pb.FileRequest_RPKI_RARC:
		// Simply store the file.
		return r.handleRPKIRarc(ctx, resp, fn, content)
	}

	return nil, fmt.Errorf("not Implemented storing: %v", req.GetFilename())
}

func main() {
	flag.Parse()

	// Validate that required flags are set.
	if *apiKey == "" {
		log.Fatal("apiKey must be defined")
	}

	// Start the listener.
	// NOTE: this listens on all IP Addresses, caution when testing.
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen(): %v", err)
	}

	r, err := newRV(*apiKey, *bucket)
	s := grpc.NewServer()
	pb.RegisterRVServer(s, r)

	// Register the reflection service on gRPC server.
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to listen&&serve: %v", err)
	}

}
