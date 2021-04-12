all: proto

proto: *.proto
	protoc rv.proto --go_out=plugins=grpc:.
