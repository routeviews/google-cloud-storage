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

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"

	pb "github.com/routeviews/google-cloud-storage/proto/rv"
)

const (
	// Max message size set to 50mb.
	maxMsgSize = 50 * 1024 * 1024
)

var (
	server = flag.String("server", "localhost:9876", "The host:port of the gRPC server.")
	file   = flag.String("file", "", "A local File to transfer to cloud storage.")
	saKey  = flag.String("sa_key", "", "Service account private key.")
)

func newConn(ctx context.Context, host string, saPath string) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption

	var idTokenSource oauth2.TokenSource
	var err error
	audience := "https://" + strings.Split(host, ":")[0]
	if saPath == "" {
		idTokenSource, err = idtoken.NewTokenSource(ctx, audience)
		if err != nil {
			if err.Error() != `idtoken: credential must be service_account, found "authorized_user"` {
				return nil, fmt.Errorf("idtoken.NewTokenSource: %v", err)
			}
			gts, err := google.DefaultTokenSource(ctx)
			if err != nil {
				return nil, fmt.Errorf("attempt to use Application Default Credentials failed: %v", err)
			}
			idTokenSource = gts
		}
	} else {
		idTokenSource, err = idtoken.NewTokenSource(ctx, audience, idtoken.WithCredentialsFile(saPath))
		if err != nil {
			return nil, fmt.Errorf("unable to create TokenSource: %v", err)
		}
	}

	opts = append(opts, grpc.WithAuthority(host))

	systemRoots, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}
	cred := credentials.NewTLS(&tls.Config{
		RootCAs: systemRoots,
	})

	opts = append(opts,
		[]grpc.DialOption{
			grpc.WithTransportCredentials(cred),
			grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(maxMsgSize)),
			grpc.WithPerRPCCredentials(oauth.TokenSource{idTokenSource}),
		}...,
	)

	return grpc.Dial(host, opts...)
}

func upload(ctx context.Context, conn *grpc.ClientConn, p *pb.FileRequest) (*pb.FileResponse, error) {
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

	ctx := context.Background()
	conn, err := newConn(ctx, *server, *saKey)
	if err != nil {
		log.Fatalf("fail to dial(%v): %v", *server, err)
	}
	defer conn.Close()

	req, err := makeReq(*file)
	if err != nil {
		log.Fatalf("fail to makeReq(%v): %v", *file, err)
	}

	resp, err := upload(ctx, conn, req)
	if err != nil {
		log.Fatalf("failed to upload file(%v): %v", *file, err)
	}

	fmt.Printf("Successfully uploaded file(%v) status: %v\n", *file, resp.GetStatus())
}
