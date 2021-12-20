# RouteViews Archive Server

Collects files to archive from remote service owners.

## Deployment

Deployed to GCP CloudRun, from the project's Docker image store.

1. Build a current image from repository root:
  ```shell
  $ docker build -f cmd/archive_upload_server/Dockerfile . \
                 --target rv-server
  ```

2. Push the docker image to the registry:
  ```shell
  $ docker push rv-server
  ```

3. Verify the loadbalanced path is in place from internet -> port.
