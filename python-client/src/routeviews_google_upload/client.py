from hashlib import md5
import logging
import os

import grpc
import google.auth.transport.requests
import google.auth.transport.grpc
import google.auth.transport.requests
import google.oauth2.service_account

from routeviews_google_upload import rv_pb2_grpc
from routeviews_google_upload import rv_pb2


logger = logging.getLogger(__name__)


MAX_MESSAGE_SIZE = 2000000000  # 2 Gigabytes
grpc_max_size_options = [
    ('grpc.max_send_message_length', MAX_MESSAGE_SIZE),
    ('grpc.max_receive_message_length', MAX_MESSAGE_SIZE),
]


def read_bytes(file_path):
    with open(file_path, "rb") as f:
        return f.read()


def setup_secure_channel(server, service_account_file):
    # Set up credentials
    id_credentials = google.oauth2.service_account.IDTokenCredentials.from_service_account_file(
        service_account_file,
        target_audience=f'https://{server}')

    # Create an authorized channel, per: https://github.com/salrashid123/grpc_google_id_tokens/blob/f09517fca10fa4b457204ec863502a917efb2a00/python/grpc_client.py
    return google.auth.transport.grpc.secure_authorized_channel(
        target=server,
        credentials=id_credentials,
        request=google.auth.transport.requests.Request(),
        ssl_credentials=grpc.ssl_channel_credentials(),
        options=grpc_max_size_options + [
            ('grpc.ssl_target_name_override', server,)
        ]
    )


def generate_FileRequest(file_path: str, to_sql: bool, filename: str = None):
    content = read_bytes(file_path)
    filename = filename if filename else file_path
    payload = rv_pb2.FileRequest()    
    payload.filename = filename.lstrip(os.sep)
    payload.project = rv_pb2._FILEREQUEST_PROJECT.values_by_name['ROUTEVIEWS'].number
    payload.convert_sql = to_sql
    payload.content = content
    payload.md5sum = md5(content).hexdigest()
    return payload


class Client:
    def __init__(self, grpc_server: str, service_account_file: str = None):
        """A client to upload Route Views MRT and UPDATE files to a gRPC server.

        Args:
            grpc_server (str): The FQDN (name) of the server (e.g. grpc.routeviews.org).
            service_account_file (str, optional): If the gRPC server requires authorization, you'll need to 
                provide the relevant Google "Service Account Key" (JSON) file path. Defaults to None.
        """
        if service_account_file:
            self._channel = setup_secure_channel(
                grpc_server, 
                service_account_file,
            )
        else:
            self._channel = grpc.insecure_channel(
                grpc_server, 
                options=grpc_max_size_options,
            )

    @property
    def channel(self):
        return self._channel

    def upload(self, file_path: str, to_sql: bool = False, filename: str = None):
        """Upload a file.

        Args:
            file_path (str): The file to be uploaded. Note: this `file_path` will 
                be used as the `filename` in the gRPC request.
            to_sql (bool, optional): Indicate in the upload that this file should 
                be converted to SQL (for BigQuery). Defaults to False.
        """
        payload = generate_FileRequest(file_path, to_sql, filename)
        grpc_client = rv_pb2_grpc.RVStub(self.channel)
        response = grpc_client.FileUpload(payload)
        logger.info("Upload Status: " + str(response.status))
        if response.error_message:
            logger.error(f'Error uploading {file_path} --- Error message: {response.error_message}')
