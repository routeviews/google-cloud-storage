import argparse
from pkg_resources import get_distribution
import sys

from routeviews_google_upload import client
from routeviews_google_upload import echo_server


def main():
    args = parse_args()
    run(args)


def run(args):
    if args.server:
        echo_server.serve()
    else:
        client.upload(args.dest, args.file, args.to_sql)


def parse_args():
    parser = argparse.ArgumentParser()
    # This tool runs in two modes -- (1) either provide a '--dest' target gRPC server, or (2) provide the '--server' flag.
    client_or_server = parser.add_mutually_exclusive_group(required=True)
    client_or_server.add_argument(
        '--dest',
        help="The gRPC server where to send the file (use 'localhost:50051' for local development)")
    client_or_server.add_argument(
        '--server', 
        action='store_true',
        help='Run a local "DEBUG::ECHO" server (for debugging purposes only).'
    )
    parser.add_argument(
        '--file', 
        help='The file to be sent. (Required when running as a client)'
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
    args = parser.parse_args()
    # When running as a client, a file must be provided.
    if args.dest and not args.file:
        parser.print_usage()
        print(f'Must also provide the `--file` argument when targeting {args.dest}.')
        sys.exit(-1)
    return args


if __name__ == '__main__':
    main()
