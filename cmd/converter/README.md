## RouteViews Archive Converter

### Deployment

1.  Build the image from the root directory. (TODO: use docker-decompose.yaml)
    -   In root directory, `docker build -f cmd/converter/Dockerfile . -t
        [IMAGE_URL]`
    -   Don't forget to upload it to the registries: `docker push -t
        [IMAGE_URL]`
2.  Deploy the image by setting the output bucket `BIGQUERY_BUCKET` for
    converted updates.
    -   Example: `gcloud run deploy [SERVICE] --image [IMAGE_URL] --cpu 2
        --memory 4Gi --update-env-vars BIGQUERY_BUCKET=[BUCKET]`
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
