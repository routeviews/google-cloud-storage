# Should be run from the parent directory, e.g. docker build -f cmd/forwarder/Dockerfile . -t [IMAGE_URL]
FROM golang:1.17-buster as builder
 
WORKDIR /app
 
COPY . ./

RUN go mod download
 
RUN go build -v -o forwarder cmd/pubsub_forwarder/main.go
 
FROM debian:buster-slim
RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/forwarder /app/forwarder

CMD ["/app/forwarder"]