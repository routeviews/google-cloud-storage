module github.com/routeviews/google-cloud-storage

go 1.16

require (
	cloud.google.com/go/bigquery v1.8.0
	cloud.google.com/go/storage v1.18.2
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/dsnet/compress v0.0.1
	github.com/fsouza/fake-gcs-server v1.31.1
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/google/go-cmp v0.5.6
	github.com/jlaffaye/ftp v0.0.0-20211117213618-11820403398b
	github.com/osrg/gobgp v0.0.0-20211201041502-6248c576b118
	github.com/routeviews/google-cloud-storage/proto/rv v0.0.0-00010101000000-000000000000
	github.com/sirupsen/logrus v1.8.1
	golang.org/x/net v0.0.0-20211015210444-4f30a5c0130f // indirect
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/api v0.62.0
	google.golang.org/genproto v0.0.0-20211203200212-54befc351ae9
	google.golang.org/grpc v1.40.1
	google.golang.org/protobuf v1.27.1
)

replace github.com/routeviews/google-cloud-storage/proto/rv => ./proto
