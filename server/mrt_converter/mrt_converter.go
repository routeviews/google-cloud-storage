package converter

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/osrg/gobgp/pkg/packet/bgp"
	"github.com/osrg/gobgp/pkg/packet/mrt"
	log "github.com/sirupsen/logrus"
)

// attributePayload represents path attribute data to be saved in BigQuery. It
// contains an attribute type and JSON of the BGP attribute.
type attributePayload struct {
	AttrType bgp.BGPAttrType
	Payload  string
}

type update struct {
	Collector string
	SeenAt    time.Time
	PeerAS    uint32

	// Data inside BGP updates.
	Announced  []string
	Withdrawn  []string
	Attributes []*attributePayload
}

// extractCollector extracts the collector name from the input file path.
func extractCollector(filename string) (string, error) {
	if filename == "" {
		return "", fmt.Errorf("empty file path")
	}
	if filename[0] != '/' {
		return "", fmt.Errorf("path must starts with '/'")
	}
	if !strings.Contains(filename, "bgpdata") {
		return "", fmt.Errorf("file %s is not a valid RouteViews archive path", filename)
	}

	dirs := strings.Split(filename, "/")
	if dirs[1] == "bgpdata" {
		return "route-views2", nil
	}
	return dirs[1], nil
}

func translateAttrs(attrs []bgp.PathAttributeInterface) []*attributePayload {
	if len(attrs) == 0 {
		return nil
	}
	var res []*attributePayload
	for _, attr := range attrs {
		p, err := json.Marshal(attr)
		if err != nil {
			log.Error(err)
		}
		res = append(res, &attributePayload{
			AttrType: attr.GetType(),
			Payload:  string(p),
		})
	}
	return res
}

func translatePrefixes(prefixes []*bgp.IPAddrPrefix) []string {
	var res []string
	for _, p := range prefixes {
		res = append(res, p.String())
	}
	return res
}

// parseUpdate converts a pair of MRT header and message into a BigQuery
// compatible update. A BGP4MP_ET message will be treated as a BGP4MP message,
// and the microsecond field will be ignored.
func parseUpdate(collector string, h *mrt.MRTHeader, buf []byte) (*update, error) {
	if h == nil {
		return nil, fmt.Errorf("header cannot be nil")
	}
	// Force GoBGP to parse BGP4MP_ET message. We do not need the extended
	// timestamp.
	if h.Type == mrt.BGP4MP_ET {
		if len(buf) < 4 {
			return nil, fmt.Errorf("bad extended timestamp: %v", buf)
		}
		h.Type = mrt.BGP4MP
		h.Len -= 4
		buf = buf[4:]
	}

	msg, err := mrt.ParseMRTBody(h, buf)
	if err != nil {
		return nil, fmt.Errorf("failed to parse body: %v", err)
	}

	mrtMsg := msg.Body.(*mrt.BGP4MPMessage)
	bgpUpdate := mrtMsg.BGPMessage.Body.(*bgp.BGPUpdate)

	return &update{
		SeenAt:    h.GetTime(),
		PeerAS:    mrtMsg.PeerAS,
		Collector: collector,

		Announced:  translatePrefixes(bgpUpdate.NLRI),
		Withdrawn:  translatePrefixes(bgpUpdate.WithdrawnRoutes),
		Attributes: translateAttrs(bgpUpdate.PathAttributes),
	}, nil
}
