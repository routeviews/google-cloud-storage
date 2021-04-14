import argparse
from proto.rv_pb2 import FileRequest
from proto.rv_pb2_grpc import RVStub

def read_file(file_path):
    pass

def main(args):
    # 1. Read the provided file.
    read_file(args.file)

    # 2. Package that file in a gRPC protobuf.
    fr = FileRequest()
    fr.filename = "Testing"

    # 3. Send to a gRPC endpoint.
    channel = grpc.insecure_channel('localhost:50051')
    stub = RVStub(channel)
    response = stub.FileUpload(message=fr)
    request(args.destination, )


    exit(0)

def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument('--file', help='The file to be sent to the Google Cloud.')
    return parser.parse_args()

if __name__ is '__main__':
    args = parse_args()
    main(args)