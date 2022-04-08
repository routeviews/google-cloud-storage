// Package mass_upload should upload content from a given ftp site
// to cloud storage. Verification of ftp site content against the
// cloud storage location before upload must be performed.
//
// Basic flow is:
//   1) start at the top of an FTP site.
//   2) download each file in turn, walking the remote directory tree.
//   3) calculate the md5() checksum for each file downloaded.
//   4) validate that the checksum matches the cloud-storage object's MD5 value.
//   5) if there is a mis-match, upload the ftp content to cloud-storage.
package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"github.com/golang/glog"
	"github.com/jlaffaye/ftp"
	"github.com/routeviews/google-cloud-storage/pkg/auth"
	pb "github.com/routeviews/google-cloud-storage/proto/rv"
	"google.golang.org/grpc"
)

const (
	dialTimeout = 5 * time.Second
	// Max message size set to 50mb.
	maxMsgSize = 512 * 1024 * 1024
	// Max ftp errors before exiting the process.
	maxFTPErrs = 50
	// Max grpc errors before processing stops. (NOTE: this is evaluated per thread)
	maxGrpcErrs = 50
	// Channel buffer size for the Walk() function to fill.
	maxWalk = 5000
)

var (
	// Google Cloud Storage bucket name to put content into.
	bucket = flag.String("bucket", "", "Bucket to mirror content into.")
	// Remote ftp archive URL to use as a starting point to read content from.
	archive = flag.String("archive", "", "Site URL to mirror content from: ftp://site/dir.")
	aUser   = flag.String("archive_user", "ftp", "Site userid to use with FTP.")
	aPasswd = flag.String("archive_pass", "mirror@", "Site password to use with this FTP.")

	// gRPC endpoint (https url) to upload replacement content to and credentials file, if necessary.
	grpcService   = flag.String("uploadURL", "rv-server-cgfq4yjmfa-uc.a.run.app:443", "Upload service host:port.")
	svcAccountKey = flag.String("saKey", "", "File location of service account key, if required.")
	threads       = flag.Int("threads", 10, "Number of ftp/cloud processing threads.")

	useTLS = flag.Bool("use_tls", true, "Enable TLS if true.")
)

type client struct {
	site    string
	user    string
	passwd  string
	gClient pb.RVClient
	bs      *storage.Client
	bh      *storage.BucketHandle
	fc      *ftp.ServerConn
	bucket  string
	// A buffered channel which will contain files to possibly download.
	ch chan *evalFile
	// A WaitGroup used to synchronize ending the reading jobs/processing.
	wg sync.WaitGroup
	// A mutex to protect the map for update processing.
	mu sync.Mutex
	// Metrics, collect copied vs not for exit reporting.
	metrics map[string]int
}

type evalFile struct {
	// name is a full filename path:
	//   /bgpdata/route-views4/bgpdata/2022.01/UPDATES/updates.20220109.1830.bz2
	name string
	// chksum is an md5 checksum
	chksum string
}

func connectFtp(site string) (*ftp.ServerConn, error) {
	conn, err := ftp.Dial(site, ftp.DialWithTimeout(dialTimeout))
	return conn, err
}

// newGRPC makes a new grpc (over https) connection for the upload service.
// host is the hostname to connect to, saPath is a path to a stored json service account key.
func newGRPC(ctx context.Context, host, saPath string) (*grpc.ClientConn, error) {
	if *useTLS {
		return auth.NewAuthConn(ctx, host, saPath)
	}
	return auth.InsecureConn(host)
}

func new(ctx context.Context, aUser, aPasswd, site, bucket, grpcService, saKey string, threads int) (*client, error) {
	f, err := connectFtp(site)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the ftp site(%v): %v", site, err)
	}

	// Login to cloud-storage, and get a bucket handle to the archive bucket.
	if err := f.Login(aUser, aPasswd); err != nil {
		return nil, fmt.Errorf("failed to login to site(%v) as u/p (%v/%v): %v",
			site, aUser, aPasswd, err)
	}

	c, errS := storage.NewClient(ctx)
	if errS != nil {
		return nil, fmt.Errorf("failed to create a new storage client: %v", errS)
	}
	// Get a BucketHandle, which enables access to the objects/etc.
	bh := c.Bucket(bucket)

	// Create a new upload service client.
	gc, err := newGRPC(ctx, grpcService, saKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client: %v", err)
	}

	wg := sync.WaitGroup{}
	wg.Add(threads)

	return &client{
		site:    site,
		user:    aUser,
		passwd:  aPasswd,
		gClient: pb.NewRVClient(gc),
		bs:      c,
		bh:      bh,
		fc:      f,
		bucket:  bucket,
		ch:      make(chan *evalFile, maxWalk),
		wg:      wg,
		mu:      sync.Mutex{},
		metrics: map[string]int{"sync": 0, "skip": 0, "error": 0},
	}, nil
}

// close politely closes the handles to cloud-storage and the ftp archive.
func (c *client) close() {
	c.fc.Quit()
	if err := c.bs.Close(); err != nil {
		glog.Fatalf("failed to close the cloud-storage client: %v", err)
	}
}

// ftpWalk walks a defined directory, sending each file
// which matches a known pattern (updates) to a channel for further evaluation.
func (c *client) ftpWalk(dir string) {
	// Start the walk activity.
	w := c.fc.Walk(dir)

	// Walk the directory tree, stat/evaluate files, else continue walking.
	for w.Next() {
		e := w.Stat()
		// The only check prior to sending the file for collection is that is a file.
		if e.Type == ftp.EntryTypeFile {
			// Add the file to the channel, for evaluation and potential copy.
			glog.Infof("Sending file for eval: %s", w.Path())
			c.ch <- &evalFile{name: strings.TrimLeft(w.Path(), "/")}
		}
	}
	if w.Err() != nil {
		glog.Errorf("Next returned false, closing channel and returning: %v", w.Err())
		glog.Errorf("Current working directory: %s", w.Path())
		// Close the channel, the readChannel threads should continue to drain this channel.
		close(c.ch)
		return
	}
}

func (c *client) metric(k string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metrics[k]++
}

// readChannel reads FTP file results from a channel, collects and compares MD5 checksums
// and uploads files to cloud-storage if mismatches occur.
func (c *client) readChannel(ctx context.Context) {
	defer c.wg.Done()
	ftpErrs := 0
	grpcErrs := 0

	// Open a new, bespoke FTP connection, so overlapping
	// command/data channel problems are avoided.
	f, err := connectFtp(c.site)
	if err != nil {
		glog.Errorf("failed to open new FTP connection: %v", err)
		return
	}
	if err := f.Login(c.user, c.passwd); err != nil {
		glog.Errorf("failed to login to FTP site: %v", err)
		return
	}

	for {
		ef, ok := <-c.ch
		// Exit if ok is false, this is the 'channel closed' signal.
		if !ok {
			glog.Error("Channel closed, exiting readChannel.")
			break
		}

		fn := strings.TrimLeft(ef.name, "/")
		// If the file doesn't have UPDATES/updates do not process further.
		if !strings.Contains(fn, "UPDATES/updates") {
			continue
		}

		csSum, err := c.md5FromGCS(ctx, fn)
		if err != nil {
			csSum = ""
		}

		fSum, fc, err := c.md5FromFTP(ef.name, f)
		if err != nil {
			if ftpErrs < maxFTPErrs {
				glog.Infof("error getting md5(%s): %v", ef.name, err)
				ftpErrs++
				continue
			}
			// Enough failures have happened, exit and restart.
			glog.Fatalf("failed to get ftp md5 for file(%s): %v", ef.name, err)
		}

		if csSum == fSum {
			c.metric("skip")
			continue
		}

		glog.Infof("Archiving file(%s) size(%d) to cloud.", ef.name, len(fc))
		req := pb.FileRequest{
			Filename: ef.name,
			Content:  fc,
			Md5Sum:   fSum,
			Project:  pb.FileRequest_ROUTEVIEWS,
		}
		resp, err := c.gClient.FileUpload(ctx, &req)
		if err != nil {
			glog.Errorf("failed uploading(%s) to grpcService: %v", ef.name, err)
			c.metric("error")
			if grpcErrs >= maxGrpcErrs {
				return
			}
			grpcErrs++
			continue
		}
		c.metric("sync")
		glog.Infof("File upload status: %s", resp.GetStatus())
	}
}

func (c *client) md5FromGCS(ctx context.Context, path string) (string, error) {
	attrs, err := c.bh.Object(path).Attrs(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get attrs for obj: %v", err)
	}
	return hex.EncodeToString(attrs.MD5), nil
}

func (c *client) md5FromFTP(path string, fc *ftp.ServerConn) (string, []byte, error) {
	r, err := fc.Retr(path)
	if err != nil {
		return "", nil, fmt.Errorf("failed to RETR the path: %v", err)
	}
	defer r.Close()

	buf, err := ioutil.ReadAll(r)
	return fmt.Sprintf("%x", md5.Sum(buf)), buf, nil
}

func main() {
	flag.Parse()
	if *bucket == "" || *archive == "" {
		glog.Fatal("set archive and bucket, or there is nothing to do")
	}

	// Clean up the archive (ftp://blah.org/floop/) to be a host/directory.
	var site, dir string
	site = *archive
	if strings.HasPrefix(site, "ftp://") {
		site = strings.TrimLeft(site, "ftp://")
	}
	if strings.HasSuffix(site, "/") {
		site = strings.TrimRight(site, "/")
	}
	parts := strings.Split(site, "/")
	site = parts[0] + ":21"
	dir = strings.Join(parts[1:], "/")
	dir = "/" + dir

	// Create a client, and start processing.
	// NOTE: Consider spawning N goroutines as fetch processors for the
	//       pathnames which are output from Walk().
	ctx := context.Background()
	c, err := new(ctx, *aUser, *aPasswd, site, *bucket, *grpcService, *svcAccountKey, *threads)
	if err != nil {
		glog.Fatalf("failed to create the client: %v", err)
	}

	// Start the readChannel threads.
	for i := 0; i < *threads; i++ {
		go c.readChannel(ctx)
	}

	// Start the FTP walk, then read from the channel and evaluate each file.
	go c.ftpWalk(dir)

	// Wait on all readChannel routines to finish.
	c.wg.Wait()

	// All operations ended, close the external services.
	glog.Info("Ending transmission/comparison.")
	c.close()
	fmt.Println("Metrics for file sync activity:")
	for k, v := range c.metrics {
		fmt.Printf("%s: %d\n", k, v)
	}
}
