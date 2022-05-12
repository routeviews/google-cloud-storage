# Should be run from the parent directory:
#   docker build -f cmd/archive_synchronizer/Dockerfile . -t archive-sync
FROM golang:1.17-buster as builder

# Set a working directory to hold built binaries.
WORKDIR /app

COPY . ./

# Download all required golang modules.
RUN go mod download ...

# Build the server binary.
RUN go build -v -o sync cmd/archive_synchronizer/sync.go

# Declare the base image, and update it.
FROM debian:buster-slim

RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Copy the built binary into the docker image.
COPY --from=builder /app/sync /app/sync

CMD ["/app/sync", "--bucket=routeviews-archives"]
