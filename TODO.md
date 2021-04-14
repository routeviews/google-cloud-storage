# TODO Items to conclude discussion and/or outline direction.

## Work Items

Items to build, research, test or evaluate in creating the services outlined above:

1. metadata storage system - a resilent service which can serve and store data about
   each file uploaded and at which stage of processing the file has progressed.
   * Determine where or how to store process metadata
      1. [cloudsql](https://cloud.google.com/sql) - seems heavyweight?
      2. [firebase real-time db](https://firebase.google.com/docs/database) - is this [firestore](https://cloud.google.com/firestore)?
      3. [cloud bigtable](https://cloud.google.com/bigtable)


2. Cloud Storage bucket (https://storage.cloud.google.com/archive-routeviews/helowurld.txt)
   NOTE: the referenced path is not public, it should be.
   * To load data into BigQuery one of the following data format conversions must happen:
      1. MRT -> [JSON](https://jsonlines.org) - json lines, newline separated json data elements
      2. MRT -> [AVRO](https://avro.apache.org) - avro provides a compressed binary format

3. BigQuery schema for the data to be loaded
   * Decide what parts of the MRT content to store in bigquery
      1. This may take longer than just storing the data and making that available.

4. CLI Client to read a file, package that file in a protobuf and send to a gRPC endpoint.

5. Server infrastructure to accept the gRPC request, process the included file and provide status.

6. Metrics to collect on the server portion of the processing (prometheus metrics)

7. Finding a golang MRT reader is also a required work item,
