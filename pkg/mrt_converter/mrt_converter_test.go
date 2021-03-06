package converter

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/osrg/gobgp/pkg/packet/bgp"
	"github.com/osrg/gobgp/pkg/packet/mrt"

	"github.com/fsouza/fake-gcs-server/fakestorage"

	pb "github.com/routeviews/google-cloud-storage/proto/rv"
	log "github.com/sirupsen/logrus"
)

// Fake MRT BGP4MP messages.
var (
	// 4-octet ASNs show up as peer and in AS_Path attribute.
	fakeAS4Ann = mrt.NewBGP4MPMessage(100000, 6447, 0, "1.0.0.0", "2.0.0.0", true, bgp.NewBGPUpdateMessage(nil, []bgp.PathAttributeInterface{
		bgp.NewPathAttributeAsPath([]bgp.AsPathParamInterface{&bgp.As4PathParam{Type: bgp.BGP_ASPATH_ATTR_TYPE_SEQ, Num: 1, AS: []uint32{100000}}}),
	}, []*bgp.IPAddrPrefix{
		bgp.NewIPAddrPrefix(24, "10.0.0.0"),
		bgp.NewIPAddrPrefix(24, "20.0.0.0"),
	}))
	fakeAnn = mrt.NewBGP4MPMessage(15169, 6447, 0, "1.0.0.0", "2.0.0.0", false, bgp.NewBGPUpdateMessage(nil, []bgp.PathAttributeInterface{
		bgp.NewPathAttributeAs4Path([]*bgp.As4PathParam{{Type: bgp.BGP_ASPATH_ATTR_TYPE_SEQ, Num: 1, AS: []uint32{100000}}}),
		bgp.NewPathAttributeAsPath([]bgp.AsPathParamInterface{&bgp.AsPathParam{Type: bgp.BGP_ASPATH_ATTR_TYPE_SEQ, Num: 1, AS: []uint16{23456}}}),
	}, []*bgp.IPAddrPrefix{
		bgp.NewIPAddrPrefix(24, "30.0.0.0"),
		bgp.NewIPAddrPrefix(24, "40.0.0.0"),
	}))
	fakeAS4Withdrawal = mrt.NewBGP4MPMessage(100000, 6447, 0, "1.0.0.0", "2.0.0.0", true, bgp.NewBGPUpdateMessage([]*bgp.IPAddrPrefix{
		bgp.NewIPAddrPrefix(24, "30.0.0.0"),
		bgp.NewIPAddrPrefix(24, "40.0.0.0"),
	}, nil, nil))
	fakeAS4UnknownAttr = mrt.NewBGP4MPMessage(100000, 6447, 0, "1.0.0.0", "2.0.0.0", true, bgp.NewBGPUpdateMessage(nil, []bgp.PathAttributeInterface{
		&bgp.PathAttributeUnknown{PathAttribute: bgp.PathAttribute{
			Flags:  bgp.BGP_ATTR_FLAG_OPTIONAL & bgp.BGP_ATTR_FLAG_TRANSITIVE,
			Type:   0xFF,
			Length: 0,
		}, Value: nil},
	}, []*bgp.IPAddrPrefix{
		bgp.NewIPAddrPrefix(24, "10.0.0.0"),
		bgp.NewIPAddrPrefix(24, "20.0.0.0"),
	}))

	gobgpCmpOpts = cmp.AllowUnexported(bgp.IPAddrPrefix{}, bgp.PrefixDefault{})
	fakeBzip     = func(r io.Reader) io.Reader { return r }
)

var (
	fourOctetASPath = &attributePayload{
		AttrType: bgp.BGP_ATTR_TYPE_AS_PATH,
		Payload: marshalAttr(bgp.NewPathAttributeAsPath([]bgp.AsPathParamInterface{
			&bgp.As4PathParam{Type: bgp.BGP_ASPATH_ATTR_TYPE_SEQ, Num: 1, AS: []uint32{100000}}})),
	}
	twoOctetASPath = &attributePayload{
		AttrType: bgp.BGP_ATTR_TYPE_AS_PATH,
		Payload: marshalAttr(bgp.NewPathAttributeAsPath([]bgp.AsPathParamInterface{
			&bgp.AsPathParam{Type: bgp.BGP_ASPATH_ATTR_TYPE_SEQ, Num: 1, AS: []uint16{23456}}})),
	}
	twoOctetAS4Path = &attributePayload{
		AttrType: bgp.BGP_ATTR_TYPE_AS4_PATH,
		Payload: marshalAttr(bgp.NewPathAttributeAs4Path([]*bgp.As4PathParam{
			{Type: bgp.BGP_ASPATH_ATTR_TYPE_SEQ, Num: 1, AS: []uint32{100000}}})),
	}
)

func encodeMRTMessage(t *testing.T, msg *mrt.MRTMessage) []byte {
	t.Helper()
	raw, err := msg.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	// Insert fake microseconds if this is BGP4MP_ET.
	if msg.Header.Type == mrt.BGP4MP_ET {
		body := raw[mrt.MRT_COMMON_HEADER_LEN:]
		msg.Header.Len = uint32(len(body) + 4)
		header, err := msg.Header.Serialize()
		if err != nil {
			t.Fatal(err)
		}
		raw = append(header, append([]byte{1, 2, 3, 4}, body...)...)
	}
	return raw
}

// encodeBGP4MP encodes a BGP4MP message, a type of MRT message, into bytes.
// The common header is not included here.
func encodeBGP4MP(t *testing.T, msg *mrt.BGP4MPMessage) []byte {
	t.Helper()
	raw, err := msg.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func fakeMRTHeader(t *testing.T, timestamp time.Time, mrtType mrt.MRTType, subType mrt.MRTSubTyper, l int) *mrt.MRTHeader {
	t.Helper()
	h, err := mrt.NewMRTHeader(uint32(timestamp.Unix()), mrtType, subType, uint32(l))
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func marshalAttr(attr bgp.PathAttributeInterface) string {
	raw, err := json.Marshal(attr)
	if err != nil {
		log.Fatal(err)
	}
	return string(raw)
}

func fakeMRTMessage(t *testing.T, timestamp time.Time, mrtType mrt.MRTType, subType mrt.MRTSubTyper, body mrt.Body) *mrt.MRTMessage {
	t.Helper()
	m, err := mrt.NewMRTMessage(uint32(timestamp.Unix()), mrtType, subType, body)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestParseUpdate(t *testing.T) {
	fakeTime := time.Unix(time.Now().Unix(), 0)
	tests := []struct {
		desc string
		// Inputs of parseUpdate().
		collector string
		header    *mrt.MRTHeader
		body      []byte

		want    *update
		wantErr bool
	}{
		{
			desc:      "parse BGP4MP message with AS4 announcement",
			collector: "route-views3",
			header:    fakeMRTHeader(t, fakeTime, mrt.BGP4MP, mrt.MESSAGE_AS4, len(encodeBGP4MP(t, fakeAS4Ann))),
			body:      encodeBGP4MP(t, fakeAS4Ann),
			want: &update{
				Collector:  "route-views3",
				SeenAt:     fakeTime,
				PeerAS:     100000, // 4-octet ASN as peer.
				Announced:  []string{"10.0.0.0/24", "20.0.0.0/24"},
				Attributes: []*attributePayload{fourOctetASPath},
			},
		},
		{
			desc:      "parse BGP4MP message with non-AS4 announcement",
			collector: "route-views3",
			header:    fakeMRTHeader(t, fakeTime, mrt.BGP4MP, mrt.MESSAGE, len(encodeBGP4MP(t, fakeAnn))),
			body:      encodeBGP4MP(t, fakeAnn),
			want: &update{
				Collector:  "route-views3",
				SeenAt:     fakeTime,
				PeerAS:     15169,
				Announced:  []string{"30.0.0.0/24", "40.0.0.0/24"},
				Attributes: []*attributePayload{twoOctetAS4Path, twoOctetASPath},
			},
		},
		{
			desc:      "parse BGP4MP message with AS4 withdrawl",
			collector: "route-views3",
			header:    fakeMRTHeader(t, fakeTime, mrt.BGP4MP, mrt.MESSAGE_AS4, len(encodeBGP4MP(t, fakeAS4Withdrawal))),
			body:      encodeBGP4MP(t, fakeAS4Withdrawal),
			want: &update{
				Collector:  "route-views3",
				SeenAt:     fakeTime,
				PeerAS:     100000,
				Withdrawn:  []string{"30.0.0.0/24", "40.0.0.0/24"},
				Attributes: nil,
			},
		},
		{
			desc:      "parse BGP4MP_ET message",
			collector: "route-views3",
			header:    fakeMRTHeader(t, fakeTime, mrt.BGP4MP_ET, mrt.MESSAGE_AS4, len(encodeBGP4MP(t, fakeAS4Ann))),
			// Add fake microseconds for extened timestamp field.
			body: append([]byte{1, 2, 3, 4}, encodeBGP4MP(t, fakeAS4Ann)...),
			want: &update{
				Collector:  "route-views3",
				SeenAt:     fakeTime,
				PeerAS:     100000, // 4-octet ASN as peer.
				Announced:  []string{"10.0.0.0/24", "20.0.0.0/24"},
				Attributes: []*attributePayload{fourOctetASPath},
			},
		},
		{
			desc:      "bad MRT body of BGP4MP",
			collector: "route-views3",
			header:    fakeMRTHeader(t, fakeTime, mrt.BGP4MP, mrt.MESSAGE_AS4, len(encodeBGP4MP(t, fakeAS4Ann))),
			body:      []byte{},
			wantErr:   true,
		},
		{
			desc:      "bad MRT body of BGP4MP_ET",
			collector: "route-views3",
			header:    fakeMRTHeader(t, fakeTime, mrt.BGP4MP_ET, mrt.MESSAGE_AS4, len(encodeBGP4MP(t, fakeAS4Ann))),
			body:      []byte{1, 2, 3},
			wantErr:   true,
		},
		{
			desc:      "bad MRT header",
			collector: "route-views3",
			body:      encodeBGP4MP(t, fakeAS4Ann),
			wantErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			got, err := parseUpdate(test.collector, test.header, test.body)
			if gotErr := err != nil; test.wantErr != gotErr {
				t.Errorf("parseUpdate() = err %v; wantErr = %v", err, test.wantErr)
			}
			if diff := cmp.Diff(test.want, got, gobgpCmpOpts); diff != "" {
				t.Errorf("parseUpdate diff: (-got +want)\n%s", diff)
			}
		})
	}
}

func TestRouteViewsCollectorFromPath(t *testing.T) {
	tests := []struct {
		desc    string
		path    string
		want    string
		wantErr bool
	}{
		{
			desc: "route-views2 archive",
			path: "/bgpdata/2021.11/UPDATES/updates.20211101.0000.bz2",
			want: "route-views2",
		},
		{
			desc: "non route-views2 archive",
			path: "/route-views.sg/bgpdata/2021.11/UPDATES/updates.20211101.0000.bz2",
			want: "route-views.sg",
		},
		{
			desc:    "bad file path - empty string",
			wantErr: true,
		},
		{
			desc:    "bad file path - invalid RouteViews path",
			path:    "route-views.sg/2021.11/UPDATES/updates.20211101.0000.bz2",
			wantErr: true,
		},
		{
			desc: "valid path - no preceding slash",
			path: "route-views.sg/bgpdata/2021.11/UPDATES/updates.20211101.0000.bz2",
			want: "route-views.sg",
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			got, err := routeViewsCollectorFromPath(test.path)
			if gotErr := err != nil; test.wantErr != gotErr || got != test.want {
				t.Errorf("routeViewsCollectorFromPath(%s) = '%s', %v; want '%s', wantErr = %v", test.path, got, err, test.want, test.wantErr)
			}
		})
	}
}

func concatMsgs(msgs ...[]byte) []byte {
	var res []byte
	for _, msg := range msgs {
		res = append(res, msg...)
	}
	return res
}

func decompressed(t *testing.T, src io.Reader) []byte {
	t.Helper()
	r, err := gzip.NewReader(src)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func makeResponse(t *testing.T, updates []*update) []byte {
	t.Helper()
	var res []byte
	for _, u := range updates {
		updateJSON, err := json.Marshal(u)
		if err != nil {
			t.Fatal(err)
		}
		res = append(res, append(updateJSON, '\n')...)
	}
	return res
}

func TestConvertMRT(t *testing.T) {
	fakeTime := time.Now()
	unextended := time.Unix(fakeTime.Unix(), 0)
	tests := []struct {
		desc      string
		collector string
		archive   []byte
		want      []*update
	}{
		{
			desc:      "convert an archive with one AS4 update",
			collector: "route-views2",
			archive:   encodeMRTMessage(t, fakeMRTMessage(t, fakeTime, mrt.BGP4MP, mrt.MESSAGE_AS4, fakeAS4Ann)),
			want: []*update{{
				Collector:  "route-views2",
				SeenAt:     unextended,
				PeerAS:     100000, // 4-octet ASN as peer.
				Announced:  []string{"10.0.0.0/24", "20.0.0.0/24"},
				Attributes: []*attributePayload{fourOctetASPath},
			}},
		},
		{
			desc:      "convert an archive with non-AS4 update",
			collector: "route-views3",
			archive:   encodeMRTMessage(t, fakeMRTMessage(t, fakeTime, mrt.BGP4MP, mrt.MESSAGE, fakeAnn)),
			want: []*update{{
				Collector:  "route-views3",
				SeenAt:     unextended,
				PeerAS:     15169,
				Announced:  []string{"30.0.0.0/24", "40.0.0.0/24"},
				Attributes: []*attributePayload{twoOctetAS4Path, twoOctetASPath},
			}},
		},
		{
			desc:      "convert an archive with withdrawal",
			collector: "route-views3",
			archive:   encodeMRTMessage(t, fakeMRTMessage(t, fakeTime, mrt.BGP4MP, mrt.MESSAGE_AS4, fakeAS4Withdrawal)),
			want: []*update{{
				Collector:  "route-views3",
				SeenAt:     unextended,
				PeerAS:     100000,
				Withdrawn:  []string{"30.0.0.0/24", "40.0.0.0/24"},
				Attributes: nil,
			}},
		},
		{
			desc:      "convert an archive with BGP4_ET",
			collector: "route-views3",
			archive:   encodeMRTMessage(t, fakeMRTMessage(t, fakeTime, mrt.BGP4MP_ET, mrt.MESSAGE_AS4, fakeAS4Withdrawal)),
			want: []*update{{
				Collector:  "route-views3",
				SeenAt:     unextended,
				PeerAS:     100000,
				Withdrawn:  []string{"30.0.0.0/24", "40.0.0.0/24"},
				Attributes: nil,
			}},
		}, {
			desc:      "convert an archive with multiple updates",
			collector: "route-views3",
			archive: concatMsgs(
				encodeMRTMessage(t, fakeMRTMessage(t, fakeTime, mrt.BGP4MP, mrt.MESSAGE_AS4, fakeAS4Ann)),
				encodeMRTMessage(t, fakeMRTMessage(t, fakeTime, mrt.BGP4MP, mrt.MESSAGE, fakeAnn)),
				encodeMRTMessage(t, fakeMRTMessage(t, fakeTime, mrt.BGP4MP_ET, mrt.MESSAGE_AS4, fakeAS4Withdrawal)),
			),
			want: []*update{{
				Collector:  "route-views3",
				SeenAt:     unextended,
				PeerAS:     100000, // 4-octet ASN as peer.
				Announced:  []string{"10.0.0.0/24", "20.0.0.0/24"},
				Attributes: []*attributePayload{fourOctetASPath},
			}, {
				Collector:  "route-views3",
				SeenAt:     unextended,
				PeerAS:     15169,
				Announced:  []string{"30.0.0.0/24", "40.0.0.0/24"},
				Attributes: []*attributePayload{twoOctetAS4Path, twoOctetASPath},
			}, {
				Collector:  "route-views3",
				SeenAt:     unextended,
				PeerAS:     100000,
				Withdrawn:  []string{"30.0.0.0/24", "40.0.0.0/24"},
				Attributes: nil,
			}},
		}, {
			desc:      "incomplete message - bad header",
			collector: "route-views3",
			archive: concatMsgs(
				encodeMRTMessage(t, fakeMRTMessage(t, fakeTime, mrt.BGP4MP, mrt.MESSAGE, fakeAnn)),
				encodeMRTMessage(t, fakeMRTMessage(t, fakeTime, mrt.BGP4MP_ET, mrt.MESSAGE_AS4, fakeAS4Withdrawal))[:10],
			),
			want: []*update{{
				Collector:  "route-views3",
				SeenAt:     unextended,
				PeerAS:     15169,
				Announced:  []string{"30.0.0.0/24", "40.0.0.0/24"},
				Attributes: []*attributePayload{twoOctetAS4Path, twoOctetASPath},
			}},
		}, {
			desc:      "incomplete message - bad body",
			collector: "route-views3",
			archive: concatMsgs(
				encodeMRTMessage(t, fakeMRTMessage(t, fakeTime, mrt.BGP4MP_ET, mrt.MESSAGE_AS4, fakeAS4Withdrawal))[:mrt.MRT_COMMON_HEADER_LEN+1],
			),
		}, {
			desc:      "parsing failed during converion",
			collector: "route-views3",
			// Full MRT message & no nlri, withdrawal or attributes, but one
			// value inside the BGP payload is wrong.
			archive: concatMsgs([]byte{97, 157, 202, 61, 0, 16, 0, 4, 0, 0, 0, 43, 0, 1, 134, 160, 0, 0,
				25, 47, 0, 0, 0, 1, 1, 0, 0, 0, 2, 0, 0, 0, 255, 255, 255, 255, 255, 255, 255,
				255, 255, 255, 255, 255, 255, 255, 255, 255,
				0, 23, 2, 100, 0, 0, 0}, // Wrong withdrawn routes length (100).
				encodeMRTMessage(t, fakeMRTMessage(t, fakeTime, mrt.BGP4MP, mrt.MESSAGE_AS4, fakeAS4Withdrawal))),

			want: []*update{{
				Collector:  "route-views3",
				SeenAt:     unextended,
				PeerAS:     100000,
				Withdrawn:  []string{"30.0.0.0/24", "40.0.0.0/24"},
				Attributes: nil,
			}},
		}, {
			desc:      "ignore unrecognized types of messages",
			collector: "route-views3",
			archive: concatMsgs(
				encodeMRTMessage(t, fakeMRTMessage(t, fakeTime, mrt.BGP4MP, mrt.MESSAGE_AS4, fakeAS4Withdrawal)),
				encodeMRTMessage(t, fakeMRTMessage(t, fakeTime, mrt.BGP4MP, mrt.STATE_CHANGE,
					mrt.NewBGP4MPStateChange(15169, 6447, 0, "1.0.0.0", "2.0.0.0", true, mrt.CONNECT, mrt.ACTIVE))),
			),
			want: []*update{{
				Collector:  "route-views3",
				SeenAt:     unextended,
				PeerAS:     100000,
				Withdrawn:  []string{"30.0.0.0/24", "40.0.0.0/24"},
				Attributes: nil,
			}},
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			// ignoreBzip will disable bzip2 decompression in tests. Golang doesn't have a
			// bzip2 encoder, and it will be difficult to create test data compressed by
			// bzip2, so we disable bzip2 in tests.
			buf := bytes.NewBuffer(nil)
			convert(test.collector, bytes.NewBuffer(test.archive), buf, fakeBzip)

			// Decompress written data.
			got := decompressed(t, buf)
			want := makeResponse(t, test.want)
			if string(want) != string(got) {
				t.Errorf("convert() outputs mismatched:\nwant: %s\ngot: %s", string(want), string(got))
			}
		})
	}
}

type badWriter struct {
	err error
}

func (w *badWriter) Write([]byte) (int, error) {
	return 0, w.err
}

func TestConvertMRTErrors(t *testing.T) {
	t.Run("bad writer", func(t *testing.T) {
		dst := &badWriter{err: fmt.Errorf("GCS not available")}
		err := convertNext(bytes.NewReader(encodeMRTMessage(t, fakeMRTMessage(t, time.Now(), mrt.BGP4MP, mrt.MESSAGE_AS4, fakeAS4Withdrawal))), dst, "routeviews.sg")
		if err == nil {
			t.Error("convert() => nil err; want non-nil err")
		}
	})
}

func TestProcessMRTArchive(t *testing.T) {
	ctx := context.Background()

	fakeTime := time.Unix(time.Now().Unix(), 0)
	fakeMRT := encodeMRTMessage(t, fakeMRTMessage(t, fakeTime, mrt.BGP4MP, mrt.MESSAGE_AS4, fakeAS4Ann))

	dstBucket := "test-dst-bucket"
	srcBucket := "test-src-bucket"
	srcObject := "bgpdata/2021.11/UPDATES/updates.20211101.0000.bz2"
	wantObject := "bgpdata/2021.11/UPDATES/updates.20211101.0000.gz"
	fakegcs := fakestorage.NewServer([]fakestorage.Object{{
		ObjectAttrs: fakestorage.ObjectAttrs{
			BucketName: srcBucket,
			Name:       srcObject,
			Metadata: map[string]string{
				ProjectMetadataKey: pb.FileRequest_ROUTEVIEWS.String(),
			},
		},
		Content: fakeMRT,
	}})
	fakegcs.CreateBucketWithOpts(fakestorage.CreateBucketOpts{
		Name: dstBucket,
	})
	t.Cleanup(fakegcs.Stop)
	fakeCli := fakegcs.Client()

	err := processMRTArchive(ctx, fakeCli, &Config{
		SrcBucket: srcBucket,
		DstBucket: dstBucket,
		SrcObject: srcObject,
	}, fakeBzip)
	if err != nil {
		t.Error(err)
	}
	wantUpdates := []*update{{
		Collector:  "route-views2",
		SeenAt:     fakeTime,
		PeerAS:     100000,
		Announced:  []string{"10.0.0.0/24", "20.0.0.0/24"},
		Attributes: []*attributePayload{fourOctetASPath},
	}}

	// Check if converted archive is expected.
	gotObj, err := fakegcs.GetObject(dstBucket, wantObject)
	if err != nil {
		t.Fatalf("fakegcs.GetObject(%s, %s): %v", dstBucket, wantObject, err)
	}
	want := makeResponse(t, wantUpdates)
	if got := decompressed(t, bytes.NewBuffer(gotObj.Content)); string(want) != string(got) {
		t.Errorf("ProcessMRTArchive() outputs mismatched:\nwant: %s\ngot: %s", string(want), string(got))
	}

	// Converted archive already exists; conversion should be skipped.
	err = processMRTArchive(ctx, fakeCli, &Config{
		SrcBucket: srcBucket,
		DstBucket: dstBucket,
		SrcObject: srcObject,
	}, fakeBzip)
	if err != nil {
		t.Errorf("processMRTArchive: %v; want nil err", err)
	}
	gotObj, err = fakegcs.GetObject(dstBucket, wantObject)
	if err != nil {
		t.Fatalf("fakegcs.GetObject(%s, %s): %v", dstBucket, wantObject, err)
	}
	if got := decompressed(t, bytes.NewBuffer(gotObj.Content)); string(want) != string(got) {
		t.Errorf("ProcessMRTArchive() outputs mismatched:\nwant: %s\ngot: %s", string(want), string(got))
	}
}

func TestProcessMRTArchiveErrors(t *testing.T) {
	tests := []struct {
		desc     string
		filename string
		metadata map[string]string
		content  []byte
	}{
		{
			desc:     "bad filename",
			filename: "/routeviews",
			metadata: map[string]string{ProjectMetadataKey: pb.FileRequest_ROUTEVIEWS.String()},
			content:  encodeMRTMessage(t, fakeMRTMessage(t, time.Now(), mrt.BGP4MP_ET, mrt.MESSAGE_AS4, fakeAS4Withdrawal)),
		},
		{
			desc:     "unrecognized project type",
			filename: "route-views.sg/bgpdata/2021.11/UPDATES/updates.20211101.0000.bz2",
			content:  encodeMRTMessage(t, fakeMRTMessage(t, time.Now(), mrt.BGP4MP_ET, mrt.MESSAGE_AS4, fakeAS4Withdrawal)),
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ctx := context.Background()
			fakegcs := fakestorage.NewServer([]fakestorage.Object{{
				ObjectAttrs: fakestorage.ObjectAttrs{
					BucketName: "src-bucket",
					Name:       test.filename,
					Metadata:   test.metadata,
				},
				Content: test.content,
			}})
			fakegcs.CreateBucketWithOpts(fakestorage.CreateBucketOpts{
				Name: "test-bucket",
			})
			t.Cleanup(fakegcs.Stop)
			err := processMRTArchive(ctx, fakegcs.Client(), &Config{
				SrcBucket: "src-bucket",
				SrcObject: test.filename,
				DstBucket: "test-bucket",
			}, fakeBzip)
			if err == nil {
				t.Error("ProcessMRTArchive() = nil err; want non-nil err")
			}
		})
	}
}
