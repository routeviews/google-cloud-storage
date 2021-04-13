# TODO Items to conclude discussion and/or outline direction.

Put items into the list (ideally with an owner) that need
research to be done and a decision made for progress on meeting
requirements listed in the README document.
   
   * Decide what parts of the MRT content to store in bigquery
      1. This may take longer than just storing the data and making that available.

   * To load data into BigQuery one of the following data format conversions must happen:

      1. MRT -> [JSON](https://jsonlines.org) - json lines, newline separated json data elements
      2. MRT -> [AVRO](https://avro.apache.org) - avro provides a compressed binary format

   * Finding a golang MRT reader is also a required work item,

   * Determine where or how to store process metadata
      1. [cloudsql](https://cloud.google.com/sql) - seems heavyweight?
      2. [firebase real-time db](https://firebase.google.com/docs/database) - is this [firestore](https://cloud.google.com/firestore)?
      3. [cloud bigtable](https://cloud.google.com/bigtable)
