# Should be run from the parent directory, e.g. docker build -f cmd/converter/Dockerfile . -t [IMAGE_URL]
FROM golang:1.17-buster as builder
 
WORKDIR /app
 
COPY . ./

RUN go mod download
 
RUN go build -v -o converter cmd/converter/main.go
 
FROM debian:buster-slim
RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/converter /app/converter

CMD ["/app/converter"]