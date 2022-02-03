# Should be run from the parent directory, e.g. docker build -f cmd/converter/Dockerfile . -t [IMAGE_URL]
FROM golang:1.17-buster as builder
 
WORKDIR /app
 
COPY . ./

RUN go mod download
 
RUN go build -v -o transfer_all cmd/utils/transfer_all/main.go
 
FROM debian:buster-slim
RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/transfer_all /app/transfer_all

CMD ["/app/transfer_all", "--logtostderr=1"]