# Convert all archives

A CLI tool to convert all archives from src bucket/directory to the dst bucket. It will send conversion requests to the cloud run endpoint in a controlled manner, and it will stop at the first failed conversion.

## Usage
  ```shell
  $  go run cmd/convert_all/main.go --src_bucket=routeviews-archives \
                                    --root_dir=bgpdata \
                                    --host=[Cloud Run URL] \
                                    --sa_key=[Path to service account key] \
                                    --num_workers=4
  ```