module github.com/routeviews/google-cloud-storage

go 1.16

require (
	cloud.google.com/go/storage v1.18.2
	github.com/dsnet/compress v0.0.1
	github.com/fsouza/fake-gcs-server v1.31.1
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/google/go-cmp v0.5.6
	github.com/osrg/gobgp v2.0.0+incompatible
	github.com/routeviews/google-cloud-storage/proto/rv v0.0.0-00010101000000-000000000000
	github.com/sirupsen/logrus v1.8.1
	golang.org/x/net v0.0.0-20211015210444-4f30a5c0130f // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/grpc v1.40.1
	google.golang.org/protobuf v1.27.1
)

replace github.com/routeviews/google-cloud-storage/proto/rv => ./proto
