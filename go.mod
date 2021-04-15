module server/server.go

go 1.16

require (
	google.golang.org/grpc v1.37.0
	google.golang.org/protobuf v1.26.0
)

replace github.com/morrowc/rv/proto/rv => ./proto/rv
