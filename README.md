# Collect and store RouteViews data, from a Cloud Service

## Overview

[Routeviews](http://www.routeviews.org)(RV) provides data collected from various
route collectors. The data comes from BGP peers to the route collectors, historical
data from at least 2004 is available. Review the RV [website](http://www.routeviews.org/routeviews/index.php/archive/) 
for specifics on data formats / timing.

This project's goal is to provide the RV data through a standard cloud storage
mechanism uploading/archiving the data to at least:

  * [Google Cloud Storage (GCS)](https://cloud.google.com/storage)
  * [Google BigQuery](https://cloud.google.com/bigquery)

Archiving to cloud-storage may be accomplished through a simple signaling method
from the RV archive server(s) to a service running in Google Cloud which is provided
the file path/name and file content upon local archive completion at RV.

Storing the data into bigquery should be enabled at the time of cloud-storage write
as well, after converting the RIB or Update data from MRT to JSON matching the bigquery
data model.
  (NOTE: possibly [AVRO](http://avro.apache.org) is better for this than JSON. AVRO
  golang code from [linkedin/github](https://github.com/linkedin/goavro))

## Requirements

> A metadata service must be built to track the state of each file in process.

Initially a CLI client for the RV upload part of the solution which can be run with simple
command-line options such as:

```shell
$ routeviews-google-upload --file <filepath> --dest https://thing.com
```

> This CLI tool is implemented! 
> See more details in the [routeviews_google_upload PyPI package](https://badge.fury.io/py/routeviews-google-upload) 

The CLI tool should package up the file content, path and a MD5  checksum of the content
in a Google Protobuf, and send that data over a [gRPC](https://gRPC.io) connection to a
cloud service. An upload event should be idempotent, meaning uploading the same file
multiple times should not negatively impact the archive.

The cloud portion of the gRPC service should be served behind a load balancer in order
to provide a resilient and scalable service. The load-balanced service will accept the gRPC
request, upload the raw file content to cloud storage, and parse the file to JSON (or AVRO?)
and store the result in a cloud storage location adjacent to the raw file. Once stored, the
data should be loaded into the BigQuery instance and a reply to the CLI caller should be sent.

The server must provide either affirmation that the files were handled properly, or an error
with appropriate status information about the fate of the file, conversion and bigquery uplaod.

## Metadata
The following metadata should be retrievable during and after the gRPC request has been made. 
Many of these attributes may be included from the standard set of metadata API 
(https://cloud.google.com/storage/docs/metadata) attributes. Some of these attributes overlap with 
FileRequest spec. 
It's unclear where this metadata will sit in relation to the actual data, or if the metadata will 
be searchable/filterable.

 1. name: `status`  
    type: `string`  
    description: `The current status of the file transfer, e.g. "None", "In Progress", "Done"` 
 2. name: `filename`  
    type: `string`  
    description: `The filepath used in the FileRequest gRPC call.`  
 3. name: `content-type`  
    type: `string`  
    description: `The IANA media type of the file, e.g. application/octet-stream, or application/MRT?`  
 4. name: `content-encoding`  
    type: `string`  
    description: `The encoding of file, e.g. bzip2`
 5. name: `updated`  
    type: `timestamp`  
    description: `The last date and time the file or metadata was update.`
 6. name: `project`  
    type: `string`
    description: `The project used in the FileRequest gRPC call`  
 7. name: `collector ID`  
    type: `string`  
    description: `An identifier that can be associated with a collector, e.g. route-views.amsix`
 8. name: `MRT_type`  
    type: `enum` 
    description: `The type of the MRT file, e.g. RIB or UPDATE`

# For Developers

With the gRPC interface defined, we actually have to instantiate a server and client that implement this interface.

For simplicity of maintaining RouteViews infrastructure, the client is implemented in Python (similar to other 
RouteViews automation solutions).
The client is implemented and owned by the University of Oregon Network and Telecom Services team, who are the primary 
maintainers of RouteViews.

> See more details in the [routeviews_google_upload project README](routeviews_google_upload/README.md)

The server is implemented in Go and maintained by Chris Morrow at Google.
