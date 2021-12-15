package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/dsnet/compress/bzip2" // Test-only.
	"github.com/fsouza/fake-gcs-server/fakestorage"
	"github.com/google/go-cmp/cmp"
	"github.com/osrg/gobgp/pkg/packet/bgp"
	"github.com/osrg/gobgp/pkg/packet/mrt"
)

const pubsubMsgFormat = `{
		"message": {
		  "publishTime": "2021-12-13T06:12:21.277Z",
		  "messageId": "3510957425154221",
		  "message_id": "3510957425154221",
		  "attributes": {
			"eventType": "%s",
			"notificationConfig": "projects/_/%s/routeviews-test-archives/notificationConfigs/1",
			"objectGeneration": "1639375940318976",
			"objectId": "%s",
			"bucketId": "%s",
			"eventTime": "2021-12-13T06:12:20.390497Z",
			"payloadFormat": "JSON_API_V1"
		  },
		  "publish_time": "2021-12-13T06:12:21.277Z"
		},
		"subscription": "projects/fake-project/subscriptions/gcs-upload"
	  }`

func makeFakeMsgFormat(reason, object, bucket string) string {
	return fmt.Sprintf(pubsubMsgFormat, reason, bucket, object, bucket)
}

func TestNewServer(t *testing.T) {
	ctx := context.Background()
	t.Run("dest bucket not specified", func(t *testing.T) {
		s, err := newServer(ctx, fakestorage.NewServer(nil).Client(), "")
		if err == nil || s != nil {
			t.Errorf("newServer(''): %v, %v; want nil server, non-nil err", s, err)
		}
	})
	t.Run("GCS is not available", func(t *testing.T) {
		cancelledCtx, cancel := context.WithCancel(ctx)
		// Cancelled immediately.
		cancel()
		t.Log(cancelledCtx.Err(), os.Getenv("STORAGE_EMULATOR_HOST"))
		s, err := newServer(cancelledCtx, nil, "test-bucket")
		if err == nil || s != nil {
			t.Errorf("newServer('test-bucket'): %v, %v; want nil server, non-nil err", s, err)
		}
	})
	t.Run("success", func(t *testing.T) {
		gcs := fakestorage.NewServer(nil)
		t.Cleanup(gcs.Stop)
		s, err := newServer(ctx, fakestorage.NewServer(nil).Client(), "test-bucket")
		if err != nil || s == nil {
			t.Errorf("newServer('test-bucket'): %v, %v; want non-nil server, nil err", s, err)
		}
	})
}

// makeFakeCompressedMRT make a fake, bzip2-compressed MRT archive in bytes.
func makeFakeCompressedMRT(t *testing.T, body mrt.Body) []byte {
	m, err := mrt.NewMRTMessage(uint32(time.Now().Unix()), mrt.BGP4MP, mrt.MESSAGE_AS4, body)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := m.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	buf := bytes.NewBuffer(nil)
	bw, err := bzip2.NewWriter(buf, nil)
	if err != nil {
		t.Fatal(err)
	}
	bw.Write(raw)
	bw.Close()
	return buf.Bytes()
}

func TestArchiveUploadHandlerErrors(t *testing.T) {
	tests := []struct {
		desc        string
		pubsubMsg   string
		fakeobjects []fakestorage.Object
		dstObjects  []string
	}{
		{
			desc:      "bad message",
			pubsubMsg: "bad JSON pubsub message",
		},
		{
			desc:      "skipped because it's not a OBJECT_FINALIZE",
			pubsubMsg: makeFakeMsgFormat("OBJECT_DELETE", "route-views4/bgpdata/updates/2021.12/updates.20211212.0015.bz2", "src-bucket"),
		},
		{
			desc:      "object doesn't exist",
			pubsubMsg: makeFakeMsgFormat("OBJECT_FINALIZE", "route-views4/bgpdata/updates/2021.12/updates.20211212.0015.bz2", "src-bucket"),
		},
		{
			desc:      "object can't be parsed",
			pubsubMsg: makeFakeMsgFormat("OBJECT_FINALIZE", "route-views4/bgpdata/updates/2021.12/updates.20211212.0015.bz2", "src-bucket"),
			fakeobjects: []fakestorage.Object{
				{
					ObjectAttrs: fakestorage.ObjectAttrs{
						BucketName: "src-bucket",
						Name:       "route-views4/bgpdata/updates/2021.12/updates.20211212.0015.bz2",
					},
					Content: []byte{1, 2, 3, 4},
				},
			},
		},
		{
			desc:      "object can't be parsed",
			pubsubMsg: makeFakeMsgFormat("OBJECT_FINALIZE", "route-views4/bgpdata/updates/2021.12/updates.20211212.0015.bz2", "src-bucket"),
			fakeobjects: []fakestorage.Object{
				{
					ObjectAttrs: fakestorage.ObjectAttrs{
						BucketName: "src-bucket",
						Name:       "route-views4/bgpdata/updates/2021.12/updates.20211212.0015.bz2",
					},
					Content: makeFakeCompressedMRT(t, mrt.NewBGP4MPMessage(100000, 6447, 0, "1.0.0.0", "2.0.0.0", true, bgp.NewBGPUpdateMessage(nil, nil, []*bgp.IPAddrPrefix{
						bgp.NewIPAddrPrefix(24, "10.0.0.0"),
						bgp.NewIPAddrPrefix(24, "20.0.0.0"),
					}))),
				},
			},
			dstObjects: []string{
				"gs://dst-bucket/route-views4/bgpdata/updates/2021.12/updates.20211212.0015.gz",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			fakegcs := fakestorage.NewServer(test.fakeobjects)
			// Create "dst-bucket" if it doesn't exist.
			fakegcs.CreateBucketWithOpts(fakestorage.CreateBucketOpts{Name: "dst-bucket"})
			t.Cleanup(fakegcs.Stop)
			server := &server{
				gcsCli:    fakegcs.Client(),
				dstBucket: "dst-bucket",
			}

			// Setup fake HTTP request.
			req, err := http.NewRequest("GET", "/", bytes.NewBuffer([]byte(test.pubsubMsg)))
			if err != nil {
				t.Fatal(err)
			}
			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(server.archiveUploadHandler)
			handler.ServeHTTP(rr, req)

			// Verify content in dst-bucket.
			objs, _, err := fakegcs.ListObjectsWithOptions("dst-bucket", fakestorage.ListOptions{})
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, obj := range objs {
				o, err := fakegcs.GetObject(obj.BucketName, obj.Name)
				if err != nil {
					t.Fatal(err)
				}
				// Check if written data can be decompressed and parsed.
				gr, _ := gzip.NewReader(bytes.NewReader(o.Content))
				de, _ := ioutil.ReadAll(gr)
				if err := json.Unmarshal(de, &(struct{}{})); err != nil {
					t.Fatalf("failed to decode written data: %v", err)
				}

				got = append(got, fmt.Sprintf("gs://%s/%s", obj.BucketName, obj.Name))
			}
			if len(got) == 0 {
				got = nil
			}
			if diff := cmp.Diff(test.dstObjects, got); diff != "" {
				t.Errorf("diff found in dest bucket:\n%s", diff)
			}
		})
	}
}
