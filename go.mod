module server/server.go

go 1.16

require (
	cloud.google.com/go/storage v1.10.0
	github.com/googleapis/gax-go v1.0.3 // indirect
	github.com/morrowc/rv/proto/rv v0.0.0-00010101000000-000000000000
	google.golang.org/api v0.44.0 // indirect
	google.golang.org/grpc v1.37.0
	google.golang.org/protobuf v1.26.0
)

replace github.com/morrowc/rv/proto/rv => ./proto
