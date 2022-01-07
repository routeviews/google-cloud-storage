# RouteViews Archive Client

gRPC client in Go to upload RouteViews archives.

## Usage

1. If you have already installed gcloud SDK ran `gcloud auth login` on the machine:
  ```shell
  $ go run client.go --file [filename] --server [host:port]
  ```

2. If you do not have gcloud SDK on the machine, download the service account key file and:
  ```shell
  $ go run client.go --file [filename] --server [host:port] --sa_key [path/to/key.json]
  ```

  Alternatively, you can point the environment variable "GOOGLE_APPLICATION_CREDENTIALS"
  to the key file and then skip the "sa_key" flag:

  ```shell
  $ export GOOGLE_APPLICATION_CREDENTIALS=[path/to/key.json]
  $ go run client.go --file [filename] --server [host:port]
  ```