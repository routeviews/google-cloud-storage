// package synchronizer uploads missing MRT archives to the gRPC archive server.
// It is only compatible with RouteViews files currently.
package synchronizer

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/jlaffaye/ftp"

	"cloud.google.com/go/storage"
	uploadutils "github.com/routeviews/google-cloud-storage/pkg/utils/upload"

	pb "github.com/routeviews/google-cloud-storage/proto/rv"
	log "github.com/sirupsen/logrus"
)

type Synchronizer struct {
	retryCount    uint64
	retryInterval time.Duration
	httpURLRoot   string

	expectedCollectorCount int

	ftpServer string
	ftpUser   string
	ftpPass   string

	bh *storage.BucketHandle
	gc pb.RVClient
}

type Config struct {
	FTPServer string
	FTPUser   string
	FTPPass   string

	GCSCli          *storage.Client
	UploadServerCli pb.RVClient
	ArchiveBucket   string
	HTTPRoot        string
}

func New(cfg *Config) (*Synchronizer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil config")
	}
	if cfg.FTPServer == "" || cfg.FTPUser == "" {
		return nil, fmt.Errorf("invalid FTP creds")
	}
	if cfg.GCSCli == nil {
		return nil, fmt.Errorf("invalid GCS client")
	}
	if cfg.UploadServerCli == nil {
		return nil, fmt.Errorf("invalid gRPC connection")
	}
	if cfg.ArchiveBucket == "" {
		return nil, fmt.Errorf("the corresponding archive bucket is not specified")
	}
	hr := cfg.HTTPRoot
	if !strings.HasPrefix(hr, "http") {
		return nil, fmt.Errorf("%s does not have the HTTP prefix", hr)
	}
	if hr[len(hr)-1] != '/' {
		hr = hr + "/"
	}

	return &Synchronizer{
		ftpServer: cfg.FTPServer,
		ftpUser:   cfg.FTPUser,
		ftpPass:   cfg.FTPPass,

		gc: cfg.UploadServerCli,
		bh: cfg.GCSCli.Bucket(cfg.ArchiveBucket),

		// TODO: Hardcoded value. Parameterize this with a config.
		expectedCollectorCount: 34,

		retryCount:    5,
		retryInterval: 10 * time.Second,
		httpURLRoot:   hr,
	}, nil
}

func (s *Synchronizer) uploadFromFTP(ctx context.Context, f string) (bool, error) {
	gcsMD5, err := uploadutils.MD5FromGCS(ctx, s.bh, f)
	if err == nil && gcsMD5 != "" {
		// Skip if file exists.
		return true, nil
	}

	ftpMD5, content, err := uploadutils.MD5FromHTTP(s.httpURLRoot + f)
	if err != nil {
		return false, fmt.Errorf("file %s cannot be downloaded from %s: %v", f, s.httpURLRoot, err)
	}
	log.Infof("Writing %s", f)
	if resp, err := s.gc.FileUpload(ctx, &pb.FileRequest{
		Filename: f,
		Content:  content,
		Md5Sum:   ftpMD5,
		Project:  pb.FileRequest_ROUTEVIEWS,
	}); err != nil {
		return false, fmt.Errorf("FileUpload: err %v, resp %s", err, resp.String())
	}
	return false, nil
}

func (s *Synchronizer) uploadFilesFromFTP(ctx context.Context, files []string) {
	var lastUploaded string
	uploadCount := 0
	for _, f := range files {
		err := backoff.Retry(func() error {
			skipped, err := s.uploadFromFTP(ctx, f)
			if err != nil {
				return fmt.Errorf("uploadFromFTP(%s): %v", f, err)
			}
			if !skipped {
				uploadCount++
			}
			return nil
		}, backoff.WithMaxRetries(backoff.NewConstantBackOff(s.retryInterval), s.retryCount))
		if err != nil {
			log.Error(err)
			return
		}

		// Check if any archive is missing (two consecutive archives are over
		// 15 minutes apart). Structured log for potential metric collection.
		latest, err := timeFromFilename(f)
		if err != nil {
			log.Errorf("failed to parse filename %s", f)
			continue
		}
		if lastUploaded != "" {
			prev, err := timeFromFilename(lastUploaded)
			if err != nil {
				log.Errorf("failed to parse filename %s", f)
				continue
			}
			diff := latest.Sub(prev)
			if diff > 15*time.Minute {
				log.WithFields(log.Fields{
					"prev": lastUploaded,
					"next": f,
					"diff": diff.Minutes(),
				}).Warn("archive missing")
			}
		}
		lastUploaded = f
	}
	log.Infof("Uploaded %d files to dir.", uploadCount)
}

func (s *Synchronizer) initFTP() (*ftp.ServerConn, error) {
	fc, err := ftp.Dial(s.ftpServer,
		ftp.DialWithTimeout(10*time.Second))
	if err != nil {
		return nil, fmt.Errorf("ftp.Dial: %v", err)
	}
	if err := fc.Login(s.ftpUser, s.ftpPass); err != nil {
		return nil, fmt.Errorf("ftp login: %v", err)
	}
	return fc, nil
}

// Sync syncs MRT archives with the GCS backup. It uses FTP to find all
// missing files from the given time span, check if any of them are missing,
// download the missing ones through HTTP (better stability), and upload them
// through the gRPC upload server. Files are uploaded in order of time per
// collector.
func (s *Synchronizer) Sync(ctx context.Context, start, end time.Time) error {
	if !start.Before(end) {
		return fmt.Errorf("start time %s should be before end time %s", start, end)
	}

	var dir map[string][]string
	months := spannedMonths(start, end)
	err := backoff.Retry(func() error {
		ftpConn, err := s.initFTP()
		if err != nil {
			log.Warningf("initFTP: %v", err)
			return err
		}

		dir, err = s.compileDir(ftpConn, start, end, months)
		if err != nil {
			log.Warningf("compileDir: %v", err)
			return fmt.Errorf("compileDir: %v", err)
		}
		return nil
	}, backoff.WithMaxRetries(backoff.NewConstantBackOff(s.retryInterval), s.retryCount))
	if err != nil {
		return err
	}

	log.Infof("Found %d roots to be sync'ed.", len(dir))

	// Synchronize directories.
	var wg sync.WaitGroup
	for r, fs := range dir {
		wg.Add(1)
		go func(root string, files []string) {
			defer wg.Done()
			s.uploadFilesFromFTP(ctx, files)
		}(r, fs)
	}
	wg.Wait()

	return nil
}

func (s *Synchronizer) compileDir(ftpConn *ftp.ServerConn, start, end time.Time, months []string) (map[string][]string, error) {
	entries, err := ftpConn.List("/")
	if err != nil {
		return nil, fmt.Errorf("ftp LIST: %v", err)
	}
	res := make(map[string][]string)
	for _, e := range entries {
		if e.Type == ftp.EntryTypeFolder {
			if e.Name == "bgpdata" {
				res[""] = nil
			} else {
				res[e.Name] = nil
			}
		}
	}
	for root := range res {
		for _, m := range months {
			monthDir := filepath.Join(root, "bgpdata", m, "UPDATES")
			files, err := ftpConn.List(monthDir)
			if err == nil && len(files) != 0 {
				for _, f := range files {
					ts, err := timeFromFilename(f.Name)
					if err != nil {
						log.Warningf("failed to parse archive %s", f.Name)
						continue
					}
					if ts.After(start) && ts.Before(end) {
						res[root] = append(res[root], filepath.Join(root, "bgpdata", m, "UPDATES", f.Name))
					}
				}
			} else if err != nil {
				log.Warningf("ftp listing %s: %v", monthDir, err)
			}
		}
	}

	// Clean up files under each directory. Sort files by their creation times
	// in an ascending order.
	res, total := prepareDir(res)
	if len(res) < s.expectedCollectorCount {
		return nil, fmt.Errorf("missing collectors: want %d, got %d",
			s.expectedCollectorCount, len(res))
	}
	log.Infof("Checking %d files from %d collectors.", total, len(res))

	return res, nil
}

func prepareDir(dir map[string][]string) (map[string][]string, int) {
	total := 0
	res := make(map[string][]string)
	for rootDir, files := range dir {
		if len(files) == 0 {
			continue
		}
		res[rootDir] = append(res[rootDir], files...)
		sort.Strings(res[rootDir])
		total += len(files)
	}
	return res, total
}

func timeFromFilename(name string) (time.Time, error) {
	_, fn := filepath.Split(name)

	vals := strings.Split(fn, ".")
	if len(vals) != 4 {
		return time.Time{}, fmt.Errorf("bad filename %s", name)
	}
	// RouteViews timestamps are in UTC.
	ts, err := time.Parse("20060102.1504 MST", fmt.Sprintf("%s.%s UTC", vals[1], vals[2]))
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse timestamp %s: %v", name, err)
	}
	return ts, nil
}

func rvMonth(now time.Time) string {
	return now.Format("2006.01")
}

// spannedMonths generate a list of months that the time range spans. The
// returned list may not be sorted in  time.
func spannedMonths(start, end time.Time) []string {
	months := make(map[string]bool)
	for curr := start; curr.Before(end); curr = curr.Add(24 * time.Hour) {
		months[rvMonth(curr)] = true
	}
	var res []string
	for m := range months {
		res = append(res, m)
	}
	return res
}
