# RouteViews Google Cloud Storage Client

This project provides a (Python) client that takes a file and sends it to the Google Cloud 
Storage solution for the RouteViews project. 

> This solution is based on protobufs, which is discussed more in [this project's top-level 
> README](../README.md).

## Installation and Usage

This solution is deployed to PyPI, so you can install it simply using the command `pip install 
routeviews-google-upload`.

    pip install routeviews-google-upload

The simplest invocation of this tool is to upload a file to a target gRPC server.

    routeviews-google-upload --dest https://grpc.routeviews.org --file /bgpdata/2021.03/UPDATES/update.20210331.2345.bz2

> Run the command with the `--help` argument to see all the expected and available arguments.

### Local Debug::Echo server

If you are interested in running the solution end-to-end but don't have a gRPC target server in-mind, then you might 
be interested in running a local "debug echo gRPC server". 
Fortunately, we have baked a simple gRPC server into `routeviews-google-upload`!
To use this debug server, you'll need two terminals open -- one for the `server` and one for the `client`.

First, in the `server` terminal window:

    $ routeviews-google-upload --server
    RouteViews gRPC debug server is running...

Then, in the `client` terminal window, you can run the tool with `--dest localhost:50051` and will recieve the 
`DEBUG::ECHO` response from the local server:

    $ routeviews-google-upload --dest localhost:50051 --file requirements.txt 
    Status: 2
    Error Message: DEBUG::ECHO::
        filename: "requirements.txt"
        md5: "1af62f45fdf90b6a1addfb2b86043acb"
        content: "grpcio==1.37.0\ngrpcio-tools==1.37.0\nprotobuf==3.15.8\nsix==1.15.0\n"
        project: ROUTEVIEWS

In fact, back in the `server` terminal, you should now see a "Recieved a request..." message printed along with the 
"error message" details that match the output that was seen in the client:

    $ routeviews-google-upload --server
    RouteViews gRPC debug server is running...
    Received a request, responding with `failure status` and the following error_message: DEBUG::ECHO::
        filename: "requirements.txt"
        md5: "1af62f45fdf90b6a1addfb2b86043acb"
        content: "grpcio==1.37.0\ngrpcio-tools==1.37.0\nprotobuf==3.15.8\nsix==1.15.0\n"
        project: ROUTEVIEWS

# For Developers

For developers who are actively developing on this solution, we recommend using a Python virtual environment to manage 
dependencies and installing the local package in `editable` mode. 

Install the dependencies that we need into a python virtual environment, `venv`.

    python3 -m venv venv
    pip install --upgrade pip
    pip install -r requirements.txt
    source venv/bin/activate

Now that we have all the needed dependencies, we can generate the gRPC python code that provides the Client 'Stub' 
interface (which is implemented in our `client.py` source code). 
    
    cd proto
    make proto_py

Finally, all the pieces are in place so that we can install the Python client.

    pip install -e .

Now, the `routeviews-google-upload` CLI will be available! 
Any updates made to the source code will be reflected immediately in your shell session.  

## Continuous Integration and Deployment

This solution is deployed to PyPI via the [Jenkinsfiles](../Jenkinsfile) in this repository. 
Whenever the `main` branch has new changes pushed to it, the Jenkins Pipeline will attempt to deploy those changes to 
PyPI.

> VERSION MANAGEMENT: The **version** of the solution is manually managed by updating the `__version__` string in the 
[\_\_init\_\_.py](__init__.py) source file as appropriate (following "Semantic Versioning" scheme).

### Recommended Git workflow

We follow the [GitHub Git Flow](https://guides.github.com/introduction/flow/) for this project.
This couples nicely with the CICD scheme described above.

> ### Recommended GitHub Repository Settings
> It would be good to have a "GitHub Branch protection rule" in place to help enforce our GitHub Flow.
The following are some 'protection rules' that we might want to consider
> * Any changes to `main` branch must go through a pull request.
> * Any pull request requires 1+ approval.

## Jenkins Jobs

We have a single Jenkins Pipeline that supports CICD for this project.
The CICD solution does depend on some Jenkins Credentials, specified below.

|           Credential ID           | Type                          | Where to find the secret value?|
|-----------------------------------|-------------------------------|--------------------------------|
| nts_github_jenkins_rsa           | SSH Username with private key | NTS KeePass -> General -> PKI Keys -> nts_github_app_jenkins_rsa | 


### Jenkins Pipeline: routeviews-google-upload-CICD 
**Basic setup details**

| Jenkinsfile | Folder | Job Name | Type | 
|-------------|--------|----------|------|
| Jenkinsfile | NTS | routeviews-google-upload-CICD | Multibranch Pipeline |

**Full configuration details**

* Branch Sources
    * Git
        * *Project Repository*: This project 
        * *Credentials*: `nts_github_jenkins_rsa` (see above)
        * *Behaviors*: 
            - Discover Branches
            - Clean before checkout ('YES' to 'Delete untracked nested repositories')
* Build Configuration
    * by Jenkinsfile 
* Scan Multibranch Pipeline Triggers
  * Periodically if not otherwise run
    * Interval: `2 minutes`
* Orphaned Item Strategy
    * Discard old items
        * *Days to keep old items*: `90`

