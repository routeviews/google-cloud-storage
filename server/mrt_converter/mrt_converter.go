package converter

import (
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
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

// update represents a MRT message with a BGP update. It will be written as
// JSON, which will then be picked up by BigQuery.
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
// The path is assumed to be a GCS object (i.e. not started with '/').
func extractCollector(filename string) (string, error) {
	if filename == "" {
		return "", fmt.Errorf("empty file path")
	}

	if strings.HasPrefix(filename, "/") {
		return "", fmt.Errorf("path should not start with '/'")
	}
	// TODO: Handle other object filenames when we import other sources.
	if !strings.Contains(filename, "bgpdata") {
		return "", fmt.Errorf("file %s is not a valid RouteViews archive path", filename)
	}
	dirs := strings.Split(filename, "/")
	if dirs[0] == "bgpdata" {
		return "route-views2", nil
	}
	return dirs[0], nil
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
		SeenAt:     h.GetTime(),
		PeerAS:     mrtMsg.PeerAS,
		Collector:  collector,
		Announced:  translatePrefixes(bgpUpdate.NLRI),
		Withdrawn:  translatePrefixes(bgpUpdate.WithdrawnRoutes),
		Attributes: translateAttrs(bgpUpdate.PathAttributes),
	}, nil
}

type bzReaderFunc func(_ io.Reader) io.Reader

// convert translates the bzip'ed MRT raw bytes into a BigQuery compatible
// format and write to GCS.
func convert(collector string, src []byte, dst io.Writer, bzip2Reader bzReaderFunc) error {
	br := bzip2Reader(bytes.NewReader(src))
	gw := gzip.NewWriter(dst)
	defer gw.Close()

	for {
		buf := make([]byte, mrt.MRT_COMMON_HEADER_LEN)
		_, err := io.ReadFull(br, buf)
		if err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("failed to read MRT header: %v", err)
		}

		h := &mrt.MRTHeader{}
		err = h.DecodeFromBytes(buf)
		if err != nil {
			return fmt.Errorf("(*mrt.MRTHeader).DecodeFromBytes: %v", err)
		}

		buf = make([]byte, h.Len)
		_, err = io.ReadFull(br, buf)
		if err != nil {
			return fmt.Errorf("failed to read MRT body: %v", err)
		}

		// We only parse updates at the moment.
		if (h.Type != mrt.BGP4MP && h.Type != mrt.BGP4MP_ET) ||
			(h.SubType != uint16(mrt.MESSAGE_AS4) && h.SubType != uint16(mrt.MESSAGE)) {
			log.WithFields(log.Fields{"type": h.Type, "subType": h.SubType}).Warn("unsupported message types")
			continue
		}

		update, err := parseUpdate(collector, h, buf)
		if err != nil {
			log.WithError(err).Warn("failed to parse update")
			continue
		}

		b, err := json.Marshal(update)
		if err != nil {
			return fmt.Errorf("json.Marshal: %v", err)
		}
		// Write as JSONL.
		if _, err := gw.Write(append(b, '\n')); err != nil {
			return fmt.Errorf("writer.Write: %v", err)
		}
	}
	return nil
}

// convertedExists checks if a converted archive already exists at the
// destination.
func convertedExists(ctx context.Context, gcsCli *storage.Client, object, bucket string) (bool, error) {
	r, err := gcsCli.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return false, nil
		}
		return false, fmt.Errorf("gs://%s/%s is not found", bucket, object)
	}
	defer r.Close()
	return true, nil
}

// ProcessMRTArchive converts an MRT dump into updates on GCS, which will later
// be picked up by BigQuery automatically. ProcessMRTDump only supports dumps
// of updates.
func ProcessMRTArchive(ctx context.Context, gcsCli *storage.Client, filename, bucket string, content []byte) error {
	return processMRTArchive(ctx, gcsCli, filename, bucket, content, bzip2.NewReader)
}

func processMRTArchive(ctx context.Context, gcsCli *storage.Client, filename, bucket string, content []byte, br bzReaderFunc) error {
	if len(content) == 0 {
		return fmt.Errorf("gs://%s/%s: content length is zero", bucket, filename)
	}
	outObject := strings.Replace(filename, filepath.Ext(filename), ".gz", 1)
	if found, err := convertedExists(ctx, gcsCli, outObject, bucket); err != nil {
		return fmt.Errorf("convertedExists: %v", err)
	} else if found {
		log.Warnf("converted archive gs://%s/%s already exists.", bucket, outObject)
		return nil
	}

	collector, err := extractCollector(filename)
	if err != nil {
		return fmt.Errorf("extractCollector(%s): %v", filename, err)
	}

	buf := bytes.NewBuffer(nil)
	err = convert(collector, content, buf, br)
	if err != nil {
		return fmt.Errorf("parser.ParseUpdateMRT: %v", err)
	}

	// Only write messages if the whole conversion is done.
	dst := gcsCli.Bucket(bucket).Object(outObject).NewWriter(ctx)
	dst.Write(buf.Bytes())
	defer dst.Close()
	return nil
}
