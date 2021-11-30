package converter

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/osrg/gobgp/pkg/packet/bgp"
	"github.com/osrg/gobgp/pkg/packet/mrt"
	log "github.com/sirupsen/logrus"
)

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
			if diff := cmp.Diff(test.want, got, cmp.AllowUnexported(bgp.PrefixDefault{}, bgp.IPAddrPrefix{})); diff != "" {
				t.Errorf("parseUpdate diff: (-got +want)\n%s", diff)
			}
		})
	}
}

func TestExtractCollector(t *testing.T) {
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
			path:    "/route-views.sg/2021.11/UPDATES/updates.20211101.0000.bz2",
			wantErr: true,
		},
		{
			desc:    "bad file path - missing preceding slash",
			path:    "route-views.sg/bgpdata/2021.11/UPDATES/updates.20211101.0000.bz2",
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			got, err := extractCollector(test.path)
			if gotErr := err != nil; test.wantErr != gotErr || got != test.want {
				t.Errorf("extractCollector(%s) = '%s', %v; want '%s', wantErr = %v", test.path, got, err, test.want, test.wantErr)
			}
		})
	}
}
