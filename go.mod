module github.com/routeviews/google-cloud-storage

go 1.16

require (
	cloud.google.com/go/bigquery v1.50.0
	cloud.google.com/go/cloudtasks v1.10.0
	cloud.google.com/go/storage v1.29.0
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/cockroachdb/apd v1.1.0 // indirect
	github.com/dsnet/compress v0.0.1
	github.com/fsouza/fake-gcs-server v1.31.1
	github.com/gidoBOSSftw5731/log v0.0.0-20210527210830-1611311b4b64
	github.com/gofrs/uuid v4.4.0+incompatible // indirect
	github.com/golang/glog v1.2.4
	github.com/google/go-cmp v0.6.0
	github.com/jackc/fake v0.0.0-20150926172116-812a484cc733 // indirect
	github.com/jackc/pgx v3.6.2+incompatible
	github.com/jlaffaye/ftp v0.0.0-20211117213618-11820403398b
	github.com/lib/pq v1.10.9 // indirect
	github.com/osrg/gobgp v0.0.0-20211201041502-6248c576b118
	github.com/routeviews/google-cloud-storage/proto/rv v0.0.0-00010101000000-000000000000
	github.com/shomali11/util v0.0.0-20220717175126-f0771b70947f
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/sirupsen/logrus v1.8.1
	golang.org/x/oauth2 v0.7.0
	google.golang.org/api v0.114.0
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1
	google.golang.org/grpc v1.56.3
	google.golang.org/protobuf v1.33.0
	gopkg.in/yaml.v2 v2.4.0
)

replace github.com/routeviews/google-cloud-storage/proto/rv => ./proto
