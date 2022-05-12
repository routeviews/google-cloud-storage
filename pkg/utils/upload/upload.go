package uploadutils

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"cloud.google.com/go/storage"
)

// MD5FromGCS returns the MD5 encoding from the GCS object's metadata.
func MD5FromGCS(ctx context.Context, hd *storage.BucketHandle, path string) (string, error) {
	attrs, err := hd.Object(path).Attrs(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get attrs for obj: %v", err)
	}
	return hex.EncodeToString(attrs.MD5), nil
}

// MD5FromHTTP returns the MD5 encoding and raw bytes by fetching data from the
// provided URL.
func MD5FromHTTP(url string) (string, []byte, error) {
	cli := http.Client{
		Timeout: time.Minute,
	}
	resp, err := cli.Get(url)
	if err != nil {
		return "", nil, fmt.Errorf("failed to download %s: %v", url, err)
	}
	defer resp.Body.Close()

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}
	return fmt.Sprintf("%x", md5.Sum(buf)), buf, nil
}
