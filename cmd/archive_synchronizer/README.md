# Archive synchronizer

This binary synchronizes archives that are missing from a given time period.
It does the following:
- Scan the archive server through FTP;
- Identify files that are within the specified time range & missing on GCS;
- Download & upload the files through HTTP.
Launch it as a Cloud Run service and triggers it with Cloud Scheduler (Cloud
Cron).

## Usage (local)
  ```shell
  $  FTP_SERVER=<FTP server addr with port> FTP_USERNAME=<FTP username> FTP_PASSWORD=<FTP password> \
     go run cmd/archive_synchronizer/sync.go --http_server=false
  ```

### Deploy as a Cloud Cron job

1.  Build the image from the root directory. (TODO: use docker-decompose.yaml)
    -   In root directory, 
    ```shell
    $   docker build -f cmd/archive_synchronizer/Dockerfile . -t us-docker.pkg.dev/public-routing-data-backup/cloudrun/archive-sync:latest
    ```
    -   Don't forget to upload it to the registries: 
    ```shell
    docker push \
        us-docker.pkg.dev/public-routing-data-backup/cloudrun/archive-sync:latest
    ```
2.  Deploy the image by setting FTP configuration as env params.
    -   Example:
    ```shell
    $   gcloud run deploy archive-sync \
        --image us-docker.pkg.dev/public-routing-data-backup/cloudrun/archive-sync:latest \
        --cpu 2 --memory 4Gi \
        --no-allow-unauthenticated \
        --concurrency 2 \
        --max-instances 1 \
        --timeout=1h \
        --update-env-vars FTP_SERVER=<FTP server addr with port>,FTP_USERNAME=<FTP username>,FTP_PASSWORD=<FTP password>
    ```

3. Add a new Cron job with Cloud Scheduler that points to the Cloud Run URL.