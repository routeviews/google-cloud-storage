package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/gidoBOSSftw5731/log"
	"google.golang.org/api/iterator"

	"cloud.google.com/go/bigquery"
)

// Sample out:
/*
2020/07/20 02:24:18 [debug]     AS54054
2020/07/20 02:24:18 [debug]     204.194.22.0
2020/07/20 02:24:18 [debug]     23
2020/07/20 02:24:18 [debug]     24
2020/07/20 02:24:18 [debug]     arin
2020/07/20 02:24:18 [debug]     [2020-07-18 10:10:02.339643 +0000 UTC 2020-07-18 13:00:05.34143 +0000 UTC 2020-07-18 14:00:04.630536 +0000 UTC 2020-07-18 15:00:04.028664 +0000 UTC 2020-07-20 03:15:51.462001 +0000 UTC]
*/

// google cloud credentials file
type Creds struct {
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url"`
	AuthURI                 string `json:"auth_uri"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	ClientX509CertURL       string `json:"client_x509_cert_url"`
	PrivateKey              string `json:"private_key"`
	PrivateKeyID            string `json:"private_key_id"`
	ProjectID               string `json:"project_id"`
	TokenURI                string `json:"token_uri"`
	Type                    string `json:"type"`
}

var (
	client *bigquery.Client
	gcreds Creds
)

func main() {
	log.SetCallDepth(4)
	gcredsPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if gcredsPath == "" {
		gcredsPath = "./Historical-ROAs-02210e643954.json"
	}
	gc, err := ioutil.ReadFile(gcredsPath)
	if err != nil {
		log.Fatalln(err)
	}

	err = json.Unmarshal(gc, &gcreds)
	if err != nil {
		log.Fatalln(err)
	}

	// open bigquery connection
	client, err = bigquery.NewClient(context.Background(), gcreds.ProjectID)
	if err != nil {
		log.Fatalln(err)
	}

	query := client.Query(`SELECT asn, prefix, mask, maxlen, ta, inserttimes FROM historical-roas.historical.roas_arr
	WHERE asn = @asn`)
	query.Parameters = []bigquery.QueryParameter{
		{
			Name:  "asn",
			Value: "AS54054",
		},
	}

	ctx := context.Background()

	job, err := query.Run(ctx)
	if err != nil {
		ErrorHandler(500, "Error with query", err)
		return
	}

	status, err := job.Wait(ctx)
	if err := status.Err(); err != nil {
		ErrorHandler(500, "Error with query", err)
		return
	}

	it, err := job.Read(ctx)
	if err != nil {
		ErrorHandler(500, "Error with query", err)
		return
	}

	for {
		var row []bigquery.Value
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			ErrorHandler(500, "Error with query", err)
			continue
		}
		for _, i := range row {
			log.Debugf("%+v\n", i)
		}
	}
}

func ErrorHandler(status int, alert string, err error) {
	log.Errorln(err)

}
