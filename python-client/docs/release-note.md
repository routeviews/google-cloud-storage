We began tracking release notes as of version 0.1.6.

## 0.2.0

* Provide a `--key-file` CLI argument, to enable Authentication against the Google Cloud Run backend.
* Removed the `--server` CLI flag. Instead, added a CLI tool for the debug-echo server: `routeviews-google-upload-test-server`
* Provide an `--override-filename` CLI argument, to enable overriding the filename in the destination gRPC server.

## 0.1.6

* Provide `routeviews-google-upload` CLI tool.
* This client works against a local debug-echo server.
* A local debug-echo server is baked into the tool, via the `--server` CLI flag.
