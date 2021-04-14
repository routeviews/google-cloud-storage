# RouteViews Google Cloud Storage Client

This project provides a (Python) client that takes a file and sends it to the Google Cloud 
Storage solution for the RouteViews project. 

> This solution is based on protobufs, which is discussed more in [this project's top-level 
> README](../README.md).

## Installation and Usage

    # NOTE: We recommend using a python virtual environment.
    pip install -r requirments.txt

    # Generate the needed gRPC protobuf python source-files
    cd ../proto
    make
    cd -