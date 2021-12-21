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
  $ gcloud run deploy  rv-server --image us-docker.pkg.dev/public-routing-data-backup/cloudrun/rv-server:latest
  ```

4. Reserve an IPv4/6 address for the load balancer (DO THIS ONCE)
   Names: cloud-backup-vip-v4 && cloud-backup-vip-v6

5. Setup certificates for the VIP/name mapping (DO THIS ONCE)
   Name: storage-archive.rarc.net

6. Setup loadbalancer config (DO THIS ONCE)
