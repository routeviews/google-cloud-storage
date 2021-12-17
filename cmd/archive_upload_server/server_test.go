package main

import (
	"context"
	"testing"

	"github.com/fsouza/fake-gcs-server/fakestorage"
	"github.com/google/go-cmp/cmp"
	pb "github.com/routeviews/google-cloud-storage/proto/rv"
	"google.golang.org/protobuf/testing/protocmp"
)

// TestFileUpload tests a full file-upload process request.
func TestFileUpload(t *testing.T) {
	tests := []struct {
		desc    string
		req     *pb.FileRequest
		bucket  string
		want    *pb.FileResponse
		wantErr bool
	}{{
		desc:   "RARC: Failure - bad checksum",
		bucket: "foo",
		req: &pb.FileRequest{
			Filename:   "bar",
			Md5Sum:     "abcdefg123456",
			Content:    []byte("Foo Bar Baz"),
			ConvertSql: false,
			Project:    pb.FileRequest_RPKI_RARC,
		},
		wantErr: true,
	}, {
		desc:   "RARC: Success",
		bucket: "foo",
		req: &pb.FileRequest{
			Filename:   "bar",
			Md5Sum:     "50e3903156f5d2dac6c9f89626d48c75",
			Content:    []byte("Foo Bar Baz"),
			ConvertSql: false,
			Project:    pb.FileRequest_RPKI_RARC,
		},
		want: &pb.FileResponse{
			Status:       pb.FileResponse_SUCCESS,
			ErrorMessage: "",
		},
	}}

	ctx := context.Background()
	for _, test := range tests {
		fs, err := newRVServer(test.bucket, fakestorage.NewServer(nil).Client())
		if err != nil {
			t.Fatalf("[%v]: failed initialzing server: %v", test.desc, err)
		}
		got, err := fs.FileUpload(ctx, test.req)
		switch {
		case err != nil && !test.wantErr:
			t.Errorf("[%v]: got error when not expoecting one: %v", test.desc, err)
		case err == nil && test.wantErr:
			t.Errorf("[%v]: did not get error when expoecting one", test.desc)
		case err == nil:
			if diff := cmp.Diff(got, test.want, protocmp.Transform()); diff != "" {
				t.Errorf("[%v] got/want mismatch:\n%v\n", test.desc, diff)
			}
		}
	}
}
