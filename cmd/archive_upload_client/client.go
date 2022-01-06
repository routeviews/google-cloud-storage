// Package main implements a golang client to upload RV archive to a Cloud Run RV service.
package main

import (
	"context"
	"crypto/md5"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"google.golang.org/api/idtoken"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	pb "github.com/routeviews/google-cloud-storage/proto/rv"
	grpcMetadata "google.golang.org/grpc/metadata"
)

var (
	server = flag.String("server", "localhost:9876", "The host:port of the gRPC server.")
	file   = flag.String("file", "", "A local File to transfer to cloud storage.")
)

func newConn(host string) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithAuthority(host))

	systemRoots, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}
	cred := credentials.NewTLS(&tls.Config{
		RootCAs: systemRoots,
	})
	opts = append(opts, grpc.WithTransportCredentials(cred))

	return grpc.Dial(host, opts...)
}

func upload(ctx context.Context, conn *grpc.ClientConn, p *pb.FileRequest, audience string) (*pb.FileResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Create an identity token.
	// With a global TokenSource tokens would be reused and auto-refreshed at need.
	// A given TokenSource is specific to the audience.
	tokenSource, err := idtoken.NewTokenSource(ctx, audience)
	if err != nil {
		return nil, fmt.Errorf("idtoken.NewTokenSource: %v", err)
	}
	token, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("TokenSource.Token: %v", err)
	}

	// Add token to gRPC Request.
	ctx = grpcMetadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token.AccessToken)

	// Send the request.
	client := pb.NewRVClient(conn)
	return client.FileUpload(ctx, p)
}

func makeReq(path string) (*pb.FileRequest, error) {
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
		Project:  pb.FileRequest_ROUTEVIEWS,
	}, nil
}

func main() {
	flag.Parse()
	if *file == "" {
		log.Fatal("a filename to transfer is required for operation")
	}

	conn, err := newConn(*server)
	if err != nil {
		log.Fatalf("fail to dial(%v): %v", *server, err)
	}
	defer conn.Close()
	ctx := context.Background()

	req, err := makeReq(*file)
	if err != nil {
		log.Fatalf("fail to makeReq(%v): %v", *file, err)
	}

	resp, err := upload(ctx, conn, req, strings.Split(*server, ":")[0])
	if err != nil {
		log.Fatalf("failed to upload file(%v): %v", *file, err)
	}

	fmt.Printf("Successfully uploaded file(%v) status: %v\n", *file, resp.GetStatus())
}
