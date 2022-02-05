# transfer_all: Transfer all JSON BGP updates to bigquery.

Transfer routing data archives (in JSON format) to BigQuery. As Bigquery has
a 10,000 files limit on each load job, this tool will create a load config for
each month & collector which automatically reloads at midnight.

## Usage (local)
  ```shell
  $  go run cmd/utils/transfer_all/main.go --project=public-routing-data-archive \
                                     --location=US \
                                     --bucket=routeviews-bigquery \
                                     --dataset=public_routing_data \
                                     --table=updates &

  $  curl localhost:8080 # trigger transfer
  ```

### Deploy to Cloud Run

1.  Build the image from the root directory. 
    -   In root directory, 
    ```shell
    $   docker build -f cmd/utils/transfer_all/Dockerfile . -t us-docker.pkg.dev/public-routing-data-backup/cloudrun/transfer-all:latest
    ```
    -   Don't forget to upload it to the registries: 
    ```shell
    docker push \
        us-docker.pkg.dev/public-routing-data-backup/cloudrun/transfer-all:latest
    ```
2.  Deploy the image:
    -   Example:
    ```shell
    $   gcloud run deploy transfer-all \
        --image us-docker.pkg.dev/public-routing-data-backup/cloudrun/transfer-all:latest \
        --concurrency=1 --max-instances=2 \
        --no-allow-unauthenticated \
        --timeout=3600
    ```
3.  **[Only need once]** Use Cloud Scheduler to trigger the cloud run service every hour.
    - Remember to add approriate permissions to the caller.
    - The serivce/personal account that runs this server must have roles `BigQuery Data Viewer`, `BigQuery Data Editor` and `Bigquery Job User`.