import sys
import os

from hashlib import md5

import grpc

# The next 3 lines enable gRPC to operate even when we are calling this from the context of a python package (e.g.
# installed via setup.py, e.g. into the user global site-package directory).
import routeviews_google_upload
this_package_path = os.path.dirname(routeviews_google_upload.__file__)
sys.path.append(this_package_path)
from routeviews_google_upload import rv_pb2
from routeviews_google_upload import rv_pb2_grpc


def read_bytes(file_path):
    with open(file_path, "rb") as f:
        return f.read()


def upload(grpc_server, file_path, to_sql=False):
    # 1. Read the provided file.
    content = read_bytes(file_path)

    # 2. Package that file in a gRPC protobuf.
    payload = rv_pb2.FileRequest()
    payload.filename = file_path
    payload.project = 1  # TODO should this be an argument rather than hardcoded?
    payload.convert_sql = to_sql
    payload.content = content
    payload.md5sum = md5(content).hexdigest()

    # 3. Send to a gRPC endpoint.
    with grpc.insecure_channel(grpc_server) as channel:
        client = rv_pb2_grpc.RVStub(channel)
        response = client.FileUpload(payload)
        print("Status: " + str(response.status))
        if response.error_message:
            print("Error Message: " + response.error_message)
