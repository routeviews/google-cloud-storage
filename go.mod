module server/server.go

go 1.16

require (
	cloud.google.com/go/bigquery v1.8.0
	cloud.google.com/go/storage v1.18.2
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/fsouza/fake-gcs-server v1.31.1 // indirect
	github.com/google/go-cmp v0.5.6
	github.com/morrowc/rv/proto/rv v0.0.0-00010101000000-000000000000
	github.com/osrg/gobgp v2.0.0+incompatible
	github.com/sirupsen/logrus v1.8.1
	google.golang.org/api v0.62.0
	google.golang.org/grpc v1.40.1
)

replace github.com/morrowc/rv/proto/rv => ./proto
