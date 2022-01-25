## RouteViews Archive Converter

### Deployment

1.  Build the image from the root directory. (TODO: use docker-decompose.yaml)
    -   In root directory, 
    ```shell
    $   docker build -f cmd/converter/Dockerfile . -t us-docker.pkg.dev/public-routing-data-backup/cloudrun/rv-converter:latest
    ```
    -   Don't forget to upload it to the registries: 
    ```shell
    docker push
        us-docker.pkg.dev/public-routing-data-backup/cloudrun/rv-converter:latest`
    ```
2.  Deploy the image by setting the output bucket `BIGQUERY_BUCKET` for
    converted updates.
    -   Example:
    ```shell
    $   gcloud run deploy rv-converter \ 
        --image us-docker.pkg.dev/public-routing-data-backup/cloudrun/rv-converter:latest \
        --cpu 2 --memory 4Gi \
        --concurrency 2 \
        --update-env-vars BIGQUERY_BUCKET=routeviews-bigquery
    ```
3.  **[Only need once]** Hook up a PubSub channel with the Cloud Run service
    through PubSub (see
    [instructions](https://cloud.google.com/run/docs/triggering/pubsub-push)).
    -   Acknowledgement deadline is set to 300s to prevent too many retry
        messages.
4.  **[Only need once]** Hook up a PubSub channel with the archive source
    bucket (see
    [instructions](https://cloud.google.com/storage/docs/pubsub-notifications)).
5.  **[Only need once]** Set up recurrent data transfer in BigQuery (see
    [instructions](https://cloud.google.com/bigquery-transfer/docs/cloud-storage-transfer))
6.  **[Only need once]** Set up log-based alerts (TBD).
