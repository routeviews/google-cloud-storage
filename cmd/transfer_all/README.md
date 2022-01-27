# transfer_all: Transfer all JSON BGP updates to bigquery.

Transfer routing data archives (in JSON format) to BigQuery. As Bigquery has
a 10,000 files limit on each load job, this tool will create a load config for
each month & collector which automatically reloads at midnight.

## Usage
  ```shell
  $  go run cmd/transfer_all/main.go --project=public-routing-data-archive \
                                     --location=US \
                                     --bucket=routeviews-bigquery \
                                     --dataset=public_routing_data \
                                     --table=updates
  ```