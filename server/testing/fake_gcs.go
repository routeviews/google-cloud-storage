// Package fakegcs provides a in-memory GCS. This package should be test-only.
package fakegcs

import (
	"context"
	"fmt"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	log "github.com/sirupsen/logrus"
)

// SetupFakeGCS creates a fake Google Cloud Storage and returns a *storage.Client pointed at it.
func SetupFakeGCS(ctx context.Context, t *testing.T, handler func(rw http.ResponseWriter, r *http.Request)) *storage.Client {
	t.Helper()
	fakeGCS := httptest.NewServer(http.HandlerFunc(handler))

	if err := os.Setenv("STORAGE_EMULATOR_HOST", fakeGCS.Listener.Addr().String()); err != nil {
		t.Fatalf("unable to set STORAGE_EMULATOR_HOST env var: %v", err)
	}
	t.Cleanup(func() { fakeGCS.Close() })

	sc, err := storage.NewClient(ctx, option.WithEndpoint(fakeGCS.URL), option.WithoutAuthentication())
	if err != nil {
		t.Fatalf("Creating storage client failed: %v", err)
	}
	t.Cleanup(func() { sc.Close() })

	return sc
}

func writeError(rw http.ResponseWriter, status int, err error) {
	log.Error(err)
	rw.Write([]byte(err.Error()))
	rw.WriteHeader(status)
}

// WriteInterceptor records the last GCS write.
type WriteInterceptor struct {
	// Below are data of the last write.
	Path    string
	Object  string
	Content []byte
}

// Handler records information of the last write into WriteInterceptor.
func (i *WriteInterceptor) Handler(rw http.ResponseWriter, r *http.Request) {
	i.Path = r.URL.Path
	if len(r.URL.Query()["name"]) != 1 {
		writeError(rw, http.StatusBadRequest, fmt.Errorf("invalid param 'name': %v", r.URL.Query()["name"]))
		return
	}
	i.Object = r.URL.Query()["name"][0]

	// Obtain written data: GCS uses multipart HTTP messages to upload
	// data; the first part contains metadata, and the second part
	// contains object data.
	_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		writeError(rw, http.StatusBadRequest, err)
		return
	}
	reader := multipart.NewReader(r.Body, params["boundary"])
	reader.NextPart()
	p, err := reader.NextPart()
	if err != nil {
		writeError(rw, http.StatusBadRequest, err)
		return
	}
	data, err := ioutil.ReadAll(p)
	if err != nil {
		writeError(rw, http.StatusBadRequest, err)
		return
	}
	i.Content = data
}
