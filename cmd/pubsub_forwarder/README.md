## PubSub Forwarder - A Cloud Run service

### Deployment

-   Build the image from the root directory.
    -   In root directory, 
    ```shell
    $   docker build -f cmd/pubsub_forwarder/Dockerfile . -t us-docker.pkg.dev/public-routing-data-backup/cloudrun/pubsub-forwarder:latest
    ```
    -   Don't forget to upload it to the registries: 
    ```shell
    $   docker push us-docker.pkg.dev/public-routing-data-backup/cloudrun/pubsub-forwarder:latest
    ```
-   Deploy the image by setting the output bucket `BIGQUERY_BUCKET` for
    converted updates.
    -   Example:
    ```shell
    $   gcloud run deploy pubsub-forwarder \
        --image us-docker.pkg.dev/public-routing-data-backup/cloudrun/pubsub-forwarder:latest \
        --no-allow-unauthenticated \
        --update-env-vars PROJECT=public-routing-data-backup,CLOUD_TASK_LOCATION=us-central1,CLOUD_TASK_QUEUE=conversion-queue
    ```
-  **[Only need once]** Create a Cloud task queue: 
    ```shell
    $   gcloud tasks queues create conversion-queue \
        --routing-override=service:rv-converter \
        --max-concurrent-dispatches=1000 \
        --max-attempts=50
    ```

    To update the queue: 
    ```shell
    $   gcloud tasks queues update conversion-queue \
        --routing-override=service:rv-converter \
        --max-concurrent-dispatches=1000 \
        --max-attempts=50
    ```

-  **[Only need once]** Hook up a PubSub channel with the Cloud Run service
    through PubSub (see
    [instructions](https://cloud.google.com/run/docs/triggering/pubsub-push)).
    Make sure you enable authentication.
-  **[Only need once]** Hook up a PubSub channel with the archive source
    bucket (see
    [instructions](https://cloud.google.com/storage/docs/pubsub-notifications)).
