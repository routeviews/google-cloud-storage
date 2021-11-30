package main

import (
	"context"
	"errors"
	"testing"

	"cloud.google.com/go/storage"
	pb "github.com/morrowc/rv/proto/rv"
)

// Setup the fake/mock server instance.
type fakeServer struct {
	fileErr   bool // If set, throw an error in FileStore.
	clientErr bool // if set, throw an error in CreateClient.
}

func (f fakeServer) FileStore(ctx context.Context, fn string, b []byte) error {
	if f.fileErr {
		return errors.New("failed in fileStore")
	}
	return nil
}

func (f fakeServer) CreateClient(ctx context.Context) (*storage.Client, error) {
	if f.clientErr {
		return nil, errors.New("failed in CreateClient")
	}
	return nil, nil
}

// TestFileUpload tests a full file-upload process request.
func TestFileUpload(t *testing.T) {
	tests := []struct {
		desc      string
		req       pb.FileRequest
		want      pb.FileResponse
		fileErr   bool
		clientErr bool
		wantErr   bool
	}{{}}

	for _, test := range tests {
		fs := fakeServer{
			fileErr:   test.fileErr,
			clientErr: test.clientErr,
		}
	}
}
