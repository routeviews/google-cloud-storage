// Package main is the grpc server which accepts file storage requests,
// Files are stored in cloud-storage, they may be converted and added to
// BigQuery tables as well, depending upon the request.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/morrowc/rv/proto/rv"
	pb "github.com/morrowc/rv/proto/rv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	port   = flag.Int("port", 9876, "Port on which gRPC connections will come.")
	apiKey = flag.String("apikey", "", "API Key to use in cloud storage operations.")

	// TODO(morrowc): find a method to define the TLS certificate to be used.
)

type RV struct {
	apiKey string
	rv.UnimplementedRVServer
}

// newRV creates and returns a proper RV object.
func newRV(key string) (RV, error) {
	return RV{
		apiKey: key,
	}, nil
}

func (r RV) FileUpload(ctx context.Context, req *pb.FileRequest) (*pb.FileResponse, error) {
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

	r, err := newRV(*apiKey)
	s := grpc.NewServer()
	pb.RegisterRVServer(s, r)

	// Register the reflection service on gRPC server.
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to listen&&serve: %v", err)
	}

}
