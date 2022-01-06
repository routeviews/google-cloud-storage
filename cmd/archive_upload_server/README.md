# RouteViews Archive Server

Collects files to archive from remote service owners.

## Deployment

Deployed to GCP CloudRun, from the project's Docker image store.

1. Build a current image from repository root:
  ```shell
  $ docker build -f cmd/archive_upload_server/Dockerfile . \
                 --tag us-docker.pkg.dev/public-routing-data-backup/cloudrun/rv-server:latest
  ```

2. Push the docker image to the registry:
  ```shell
  $ docker push us-docker.pkg.dev/public-routing-data-backup/cloudrun/rv-server:latest
  ```

3. Have cloud run, run the job:
  ```shell
  $ gcloud run deploy  rv-server --image us-docker.pkg.dev/public-routing-data-backup/cloudrun/rv-server:latest --use-http2 --no-allow-unauthenticated
  ```

4. Verify the loadbalanced path is in place from internet -> port.
