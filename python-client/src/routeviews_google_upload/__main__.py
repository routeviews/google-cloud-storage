import argparse
import logging
from pkg_resources import get_distribution

import uologging

from routeviews_google_upload.client import Client


logger = logging.getLogger(__name__)

def main():
    args = parse_args()
    if args.syslog:
        uologging.init_syslog_logging()
    try:
        run(args)
    except Exception as e:
        logger.exception(e)
        raise e


def run(args):
    client = Client(args.dest, args.key_file)
    if args.override_filename:
        client.upload(args.file, args.to_sql, args.override_filename)
    else:
        client.upload(args.file, args.to_sql)


def parse_args():
    parser = argparse.ArgumentParser()
    # This tool runs in two modes -- (1) either provide a '--dest' target gRPC server, or (2) provide the '--server' flag.
    parser.add_argument(
        '--dest',
        default='grpc.routeviews.org',
        help="The gRPC server where to send the file (use 'localhost:50051' for local development)"
    )
    parser.add_argument(
        '--file',
        required=True,
        help='The file to be sent.'
    )
    parser.add_argument(
        '--override-filename',
        help='''Override the filename in the destination gRPC server. 
                (omit to simply use the name/path provided by the --file argument).'''
    )
    parser.add_argument(
        '--key-file',
        help='If the destination gRPC server required authentication, provide an appropriate Service Account Key file (JSON).'
    )
    parser.add_argument(
        '--to-sql', 
        action='store_true', 
        help='Convert to sql (e.g. for uploading to BigQuery).'
    )
    parser.add_argument(
        '--version', 
        action='version', 
        version=get_distribution('routeviews_google_upload').version
    )
    parser.add_argument(
        '--syslog',
        action='store_true',
        help='Send log messages to syslog (in addition to the console).',
    )
    return parser.parse_args()


if __name__ == '__main__':
    main()
