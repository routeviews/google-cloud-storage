all: proto

go: proto_go

py: proto_py

proto: *.proto
	protoc --proto_path=. --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative rv.proto
	python3 -m grpc_tools.protoc --proto_path=. rv.proto --python_out=. --grpc_python_out=.

proto_go: *.proto
	protoc --proto_path=. --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative rv.proto

proto_py: *.proto
	python3 -m grpc_tools.protoc --proto_path=. rv.proto --python_out=. --grpc_python_out=.
	touch __init__.py