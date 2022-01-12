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
	"log"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/jlaffaye/ftp"
)

const (
	dialTimeout = 5 * time.Second
)

var (
	// Google Cloud Storage bucket name to put content into.
	bucket = flag.String("bucket", "", "Bucket to mirror content into.")
	// Remote ftp archive URL to use as a starting point to read content from.
	archive = flag.String("archive", "", "Site URL to mirror content from.")
	aUser   = flag.String("archive_user", "ftp", "Site userid to use with FTP.")
	aPasswd = flag.String("archive_pass", "mirror@", "Site password to use with this FTP.")
)

type client struct {
	bs     *storage.Client
	bh     *storage.BucketHandle
	fc     *ftp.ServerConn
	bucket string
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

func new(ctx context.Context, aUser, aPasswd, site, bucket string) (*client, error) {
	f, err := connectFtp(site)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the ftp site(%v): %v", site, err)
	}

	// Change to the top level directory of the archive.
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

	return &client{
		bs:     c,
		bh:     bh,
		fc:     f,
		bucket: bucket,
	}, nil
}

func (c *client) close() {
	c.fc.Quit()
	if err := c.bs.Close(); err != nil {
		log.Fatalf("failed to close the cloud-storage client: %v", err)
	}
}

func (c *client) getMD5cloud(ctx context.Context, path string) (string, error) {
	attrs, err := c.bh.Object(path).Attrs(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get attrs for obj: %v", err)
	}
	return hex.EncodeToString(attrs.MD5), nil
}

func (c *client) getMD5ftp(path string) (string, error) {
	r, err := c.fc.Retr(path)
	if err != nil {
		return "", fmt.Errorf("failed to RETR the path: %v", err)
	}
	defer r.Close()

	buf, err := ioutil.ReadAll(r)
	return fmt.Sprintf("%x", md5.Sum(buf)), nil
}

func main() {
	flag.Parse()
	if *bucket == "" || *archive == "" {
		log.Fatal("set archive and bucket, or there is nothing to do")
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

	// Open the ftp connection.
	// NOTE: Consider spawning N goroutines as fetch processors for the
	//       pathnames which are output from Walk().
	ctx := context.Background()
	c, err := new(ctx, *aUser, *aPasswd, site, *bucket)
	if err != nil {
		log.Fatalf("failed to create the client: %v", err)
	}

	// Move this Walk activity to a dedicated go-routine,
	// feed collected filenames to a channel of evalFile, evaluate the channel
	// with a set of go-routines collecting information from cloud-storage.
	// to verify whether or not the file must be uploaded to storage.
	w := c.fc.Walk(dir)
	// Walk the directory tree, stat/evaluate files, else continue walking.
	i := 0
	for {
		if !w.Next() {
			fmt.Println("Next returned false")
			return
		}
		e := w.Stat()
		if e.Type == ftp.EntryTypeFile && strings.HasPrefix(e.Name, "updates") {
			fmt.Printf("Found file: %v = sz: %d\n", w.Path(), e.Size)
			i++
			if i >= 10 {
				fmt.Println("Got to 10000")
				break
			}
		}
	}

	// Request one object: (via ObjectHandle that allows access to attrs)
	//   /bgpdata/2022.01/UPDATES/updates.20220101.0000.bz2
	// reported md5: 2723e683865e5d22e12504e3fbede7c6
	// routeviews-archives/bgpdata/2022.01/UPDATES/updates.20220101.0000.bz2
	cksum, err := c.getMD5cloud(ctx, "bgpdata/2022.01/UPDATES/updates.20220101.0000.bz2")
	fmt.Printf("MD5: %s\nKNN: 2723e683865e5d22e12504e3fbede7c6\n", cksum)
	fSum, err := c.getMD5ftp("bgpdata/2022.01/UPDATES/updates.20220101.0000.bz2")
	if err != nil {
		log.Fatalf("failed to get md5 sum over ftp: %v", err)
	}
	fmt.Printf("FTP: %s\n", fSum)

	// Download the file from FTP, compare MD5.
}
