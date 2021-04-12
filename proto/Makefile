all: proto

proto: *.proto
	protoc --proto_path=. --go_out=. --go_opt=paths=source_relative rv.proto
