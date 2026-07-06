# ROA Proxy Server

This is a simple HTTP server designed to act as a backend for an nginx reverse-proxy setup.
It fetches ROA (Route Origin Authorization) data from a configured target URL and returns it to the requester.

## Usage

To run the server:

```bash
go run roa_proxy/main.go [flags]
```

### Flags

*   `-port`: The port to listen on (default: `8080`)
*   `-target-url`: The URL to fetch ROA data from (default: `https://hosted-routinator.rarc.net/json`)
*   `-timeout`: Timeout for fetching data from the target URL (default: `30s`)

## Testing

To run the tests:

```bash
go test -v ./roa_proxy/...
```
