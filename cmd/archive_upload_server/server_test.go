package main

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/fsouza/fake-gcs-server/fakestorage"
	"github.com/google/go-cmp/cmp"
	converter "github.com/routeviews/google-cloud-storage/pkg/mrt_converter"
	pb "github.com/routeviews/google-cloud-storage/proto/rv"
	"google.golang.org/protobuf/testing/protocmp"
	"gopkg.in/yaml.v2"
)

func createConf(t *testing.T, conf *config) string {
	t.Helper()
	raw, err := yaml.Marshal(conf)
	if err != nil {
		t.Fatal(err)
	}
	return tempFile(t, "conf.yaml", raw)
}

func tempFile(t *testing.T, fn string, data []byte) string {
	t.Helper()
	file, err := ioutil.TempFile(t.TempDir(), fn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return file.Name()
}

// TestFileUpload tests a full file-upload process request.
func TestFileUpload(t *testing.T) {
	tests := []struct {
		desc    string
		req     *pb.FileRequest
		conf    *config
		want    *pb.FileResponse
		wantErr bool
	}{{
		desc: "RARC: Failure - bad checksum",
		conf: &config{
			Buckets: map[string]string{
				pb.FileRequest_ROUTEVIEWS.String():     "foo",
				pb.FileRequest_ROUTEVIEWS_RIB.String(): "baz",
			},
		},
		req: &pb.FileRequest{
			Filename: "bar",
			Md5Sum:   "abcdefg123456",
			Content:  []byte("Foo Bar Baz"),
			Project:  pb.FileRequest_RPKI_RARC,
		},
		wantErr: true,
	}, {
		desc: "RARC: Success",
		conf: &config{
			Buckets: map[string]string{
				pb.FileRequest_RPKI_RARC.String(): "foo",
			},
		},
		req: &pb.FileRequest{
			Filename: "bar",
			Md5Sum:   "50e3903156f5d2dac6c9f89626d48c75",
			Content:  []byte("Foo Bar Baz"),
			Project:  pb.FileRequest_RPKI_RARC,
		},
		want: &pb.FileResponse{
			Status:       pb.FileResponse_SUCCESS,
			ErrorMessage: "",
		},
	}, {
		desc: "Routeviews: Success",
		conf: &config{
			Buckets: map[string]string{
				pb.FileRequest_ROUTEVIEWS.String():     "foo",
				pb.FileRequest_ROUTEVIEWS_RIB.String(): "baz",
			},
		},
		req: &pb.FileRequest{
			Filename: "bar",
			Md5Sum:   "50e3903156f5d2dac6c9f89626d48c75",
			Content:  []byte("Foo Bar Baz"),
			Project:  pb.FileRequest_ROUTEVIEWS,
		},
		want: &pb.FileResponse{
			Status:       pb.FileResponse_SUCCESS,
			ErrorMessage: "",
		},
	}, {
		desc: "Routeviews: sent to RV RIB bucket",
		conf: &config{
			Buckets: map[string]string{
				pb.FileRequest_ROUTEVIEWS.String():     "foo",
				pb.FileRequest_ROUTEVIEWS_RIB.String(): "baz",
			},
		},
		req: &pb.FileRequest{
			Filename: "bar",
			Md5Sum:   "50e3903156f5d2dac6c9f89626d48c75",
			Content:  []byte("Foo Bar Baz"),
			Project:  pb.FileRequest_ROUTEVIEWS_RIB,
		},
		want: &pb.FileResponse{
			Status:       pb.FileResponse_SUCCESS,
			ErrorMessage: "",
		},
	}}

	ctx := context.Background()
	for _, test := range tests {
		srv := fakestorage.NewServer(nil)
		cli := srv.Client()
		srv.CreateBucket("baz")
		srv.CreateBucket("foo")

		fs, err := newRVServer(context.Background(), createConf(t, test.conf), cli)
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

		// Check validity of uploaded files.
		if test.wantErr {
			return
		}
		wantBkt := test.conf.Buckets[test.req.GetProject().String()]
		obj, err := srv.GetObject(wantBkt, test.req.Filename)
		if err != nil {
			t.Fatal(err)
		}
		if gotProj := obj.ObjectAttrs.Metadata[converter.ProjectMetadataKey]; gotProj != test.req.Project.String() {
			t.Errorf("got metadata %s=%s; want %s", converter.ProjectMetadataKey, gotProj, test.req.Project.String())
		}
	}
}

func TestBadConfig(t *testing.T) {
	tests := []struct {
		desc string
		data []byte
	}{
		{
			desc: "Failure - non-existent bucket",
			data: func() []byte {
				raw, _ := yaml.Marshal(&config{
					Buckets: map[string]string{
						pb.FileRequest_RPKI_RARC.String(): "foo",
					},
				})
				return raw
			}(),
		},
		{
			desc: "Failure - bad yaml config",
			data: []byte(`b:a
			b:a`),
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			_, err := newRVServer(context.Background(),
				tempFile(t, "conf.yaml", test.data),
				fakestorage.NewServer(nil).Client())
			if err == nil {
				t.Error("newRVServer: nil err; want non-nil err")
			}
		})
	}
}
