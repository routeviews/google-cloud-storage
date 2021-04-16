from concurrent import futures
from textwrap import indent

import grpc

import rv_pb2_grpc
import rv_pb2


class Servicer(rv_pb2_grpc.RVServicer):
    def FileUpload(self, request, context):
        msg = f"DEBUG::ECHO::\n{indent(str(request), '    ')}\n"
        print(msg)
        return rv_pb2.FileResponse(status=2, error_message=msg)


def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    rv_pb2_grpc.add_RVServicer_to_server(Servicer(), server)
    server.add_insecure_port('[::]:50051')
    server.start()
    server.wait_for_termination()


if __name__ == '__main__':
    serve()
