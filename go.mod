module github.com/routeviews/google-cloud-storage

go 1.23

require (
	cloud.google.com/go/bigquery v1.50.0
	cloud.google.com/go/cloudtasks v1.10.0
	cloud.google.com/go/storage v1.29.0
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/dsnet/compress v0.0.1
	github.com/fsouza/fake-gcs-server v1.31.1
	github.com/gidoBOSSftw5731/log v0.0.0-20210527210830-1611311b4b64
	github.com/golang/glog v1.2.4
	github.com/google/go-cmp v0.6.0
	github.com/jackc/pgx v3.6.2+incompatible
	github.com/jlaffaye/ftp v0.0.0-20211117213618-11820403398b
	github.com/osrg/gobgp v0.0.0-20211201041502-6248c576b118
	github.com/routeviews/google-cloud-storage/proto/rv v0.0.0-00010101000000-000000000000
	github.com/shomali11/util v0.0.0-20220717175126-f0771b70947f
	github.com/sirupsen/logrus v1.8.1
	golang.org/x/oauth2 v0.27.0
	google.golang.org/api v0.114.0
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1
	google.golang.org/grpc v1.56.3
	google.golang.org/protobuf v1.33.0
	gopkg.in/yaml.v2 v2.4.0
)

require (
	cloud.google.com/go v0.110.0 // indirect
	cloud.google.com/go/compute/metadata v0.3.0 // indirect
	cloud.google.com/go/iam v0.13.0 // indirect
	github.com/andybalholm/brotli v1.0.4 // indirect
	github.com/apache/arrow/go/v11 v11.0.0 // indirect
	github.com/apache/thrift v0.16.0 // indirect
	github.com/cockroachdb/apd v1.1.0 // indirect
	github.com/felixge/httpsnoop v1.0.1 // indirect
	github.com/goccy/go-json v0.9.11 // indirect
	github.com/gofrs/uuid v4.4.0+incompatible // indirect
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/flatbuffers v2.0.8+incompatible // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.2.3 // indirect
	github.com/googleapis/gax-go/v2 v2.7.1 // indirect
	github.com/gorilla/handlers v1.5.1 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/jackc/fake v0.0.0-20150926172116-812a484cc733 // indirect
	github.com/klauspost/asmfmt v1.3.2 // indirect
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/klauspost/cpuid/v2 v2.0.9 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/minio/asm2plan9s v0.0.0-20200509001527-cdd76441f9d8 // indirect
	github.com/minio/c2goasm v0.0.0-20190812172519-36a3d3bbc4f3 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	go.opencensus.io v0.24.0 // indirect
	golang.org/x/crypto v0.0.0-20211108221036-ceb1ce70b4fa // indirect
	golang.org/x/mod v0.9.0 // indirect
	golang.org/x/net v0.9.0 // indirect
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	golang.org/x/tools v0.7.0 // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	google.golang.org/appengine v1.6.7 // indirect
)

replace github.com/routeviews/google-cloud-storage/proto/rv => ./proto
