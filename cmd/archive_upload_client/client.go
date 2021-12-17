// Package main implements a golang client to speak to the RV cloud storage relay server.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	pb "github.com/routeviews/google-cloud-storage/proto/rv"
	"google.golang.org/grpc"
)

var (
	server = flag.String("server", "localhost:9876", "The host:port of the gRPC server.")
	file   = flag.String("file", "", "A File to transfer to cloud storage.")
)

func main() {
	flag.Parse()
	if *file == "" {
		log.Fatal("a filename to transfer is required for operation")
	}

	// Create the gRPC Connection.
	var opts []grpc.DialOption
	// TODO(morrowc): Fix the TLS process here, do not be insecure.
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(*server, opts...)
	if err != nil {
		log.Fatalf("fail to dial(%v): %v", *server, err)
	}
	defer conn.Close()
	ctx := context.Background()

	client := pb.NewRVClient(conn)

	r := pb.FileRequest{
		Filename: *file,
	}

	resp, err := client.FileUpload(ctx, &r)
	if err != nil {
		log.Fatalf("failed to upload file(%v): %v", *file, err)
	}

	fmt.Printf("Successfully uploaded file(%v) status: %v\n", *file, resp.GetStatus())
}
