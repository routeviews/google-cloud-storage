// Package main implements a golang client to upload RV archive to a Cloud Run RV service.
package main

import (
	"context"
	"crypto/md5"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"google.golang.org/grpc"

	"github.com/routeviews/google-cloud-storage/pkg/auth"
	pb "github.com/routeviews/google-cloud-storage/proto/rv"
)

const (
	// Max message size set to 50mb.
	maxMsgSize = 512 * 1024 * 1024
)

var (
	server  = flag.String("server", "localhost:9876", "The host:port of the gRPC server.")
	file    = flag.String("file", "", "A local File to transfer to cloud storage.")
	saKey   = flag.String("sa_key", "", "Service account private key.")
	project = flag.String("project", "", "Determines which project this file belongs to.")
	useTLS  = flag.Bool("use_tls", true, "Enable TLS if true.")
)

func newConn(ctx context.Context, host string, saPath string) (*grpc.ClientConn, error) {
	if *useTLS {
		return auth.NewAuthConn(ctx, host, saPath)
	}
	return auth.InsecureConn(host)
}

func upload(ctx context.Context, conn *grpc.ClientConn, p *pb.FileRequest) (*pb.FileResponse, error) {
	client := pb.NewRVClient(conn)
	return client.FileUpload(ctx, p)
}

func makeReq(path string, proj pb.FileRequest_Project) (*pb.FileRequest, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	raw, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return &pb.FileRequest{
		Filename: path,
		Content:  raw,
		Md5Sum:   fmt.Sprintf("%x", md5.Sum(raw)),
		Project:  proj,
	}, nil
}

func main() {
	flag.Parse()
	if *file == "" {
		log.Fatal("a filename to transfer is required for operation")
	}

	ctx := context.Background()
	conn, err := newConn(ctx, *server, *saKey)
	if err != nil {
		log.Fatalf("fail to dial(%v): %v", *server, err)
	}
	defer conn.Close()

	req, err := makeReq(*file, pb.FileRequest_Project(pb.FileRequest_Project_value[*project]))
	if err != nil {
		log.Fatalf("fail to makeReq(%v): %v", *file, err)
	}

	resp, err := upload(ctx, conn, req)
	if err != nil {
		log.Fatalf("failed to upload file(%v): %v", *file, err)
	}

	fmt.Printf("Successfully uploaded file(%v) status: %v\n", *file, resp.GetStatus())
}
