# Convert all archives

A CLI tool to convert all archives from src bucket/directory to the dst bucket. It will stop on the first failed conversion.

## Usage
  ```shell
  $  go run cmd/convert_all/main.go --src_bucket=routeviews-archives \
                                    --dst_bucket=routeviews-bigquery \
                                    --root_dir=bgpdata \
                                    --num_workers=4
  ```