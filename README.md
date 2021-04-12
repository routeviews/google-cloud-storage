# Collect and store RouteViews data, from a Cloud Service

## Overview

[Routeviews](http://www.routeviews.org)(RV) provides data collected from various
route servers. The data comes from BGP peers to the route servers, historical
data from at least 2004 is available. Review the RV website for specifics on
data formats / timing.

This project's goal is to provide the RV data through a standard cloud storage
mechanism uploading/archiving the data to at least:

  * [google cloud storage](https://cloud.google.com/storage)
  * [google big query](https://cloud.google.com/bigquery)

Archiving to cloud-storage may be accomplished through a simple signaling method
from the RV archive server(s) to a service running in Google Cloud which is provided
the file path/name and file content upon local archive completion at RV.

Storing the data into bigquery should be enabled at the time of cloud-storage write
as well, after converting the RIB or Update data from MRT to JSON matching the bigquery
data model.
  (NOTE: possibly [AVRO](http://avro.apache.org) is better for this than JSON. AVRO
  golang code from [linkedin/github](https://github.com/linkedin/goavro))

## Requirements

<A metadata service must be built to track the state of each file in process>

Initially a CLI client for the RV upload part of the solution which can be run with simple
command-line options such as:

```shell
$ upload-to-cloudz -f <filepath> -d https://thing.com
```

The CLI tool should package up the file content, path and a sha256 checksum of the content
in a Google Protobuf, and send that data over a [gRPC](https://grpc.io) connection to a
cloud service. An upload event should be idempotent, meaning uploading the same file
multiple times should not negatively impact the archive.

The cloud portion of the gRPC service should be served behind a load balancer in order
to provide a resilient and scalable service. The load-balanced service will accept the gRPC
request, upload the raw file content to cloud storage, and parse the file to JSON (or AVRO?)
and store the result in a cloud storage location adjacent to the raw file. Once stored, the
data should be loaded into the BigQuery instance and a reply to the CLI caller should be sent.

The server must provide either affirmation that the files were handled properly, or an error
with appropriate status information about the fate of the file, conversion and bigquery uplaod.

## Work Items

Items to build, research, test or evaluate in creating the services outlined above:

1. metadata storage system - a resilent service which can serve and store data about
   each file uploaded and at which stage of processing the file has progressed.
2. Cloud Storage bucket (https://storage.cloud.google.com/archive-routeviews/helowurld.txt)
   NOTE: the referenced path is not public, it should be.
3. BigQuery schema for the data to be loaded
4. CLI Client to read a file, package that file in a protobuf and send to a gRPC endpoint.
5. Server infrastructure to accept the gRPC request, process the included file and provide status.
6. Metrics to collect on the server portion of the processing (prometheus metrics)

