This document discusses setting up local development for this project on your own personal Linux box.

## Install Dependencies 

For developers who are actively developing on this solution, we recommend using a Python virtual environment to manage dependencies and installing the local package in `editable` mode.

Install the dependencies that we need into a python virtual environment, `venv`.

    python3 -m venv venv
    source venv/bin/activate
    pip install --upgrade pip
    pip install -r requirements.txt

## Generate gRPC source files

Now that we have all the needed dependencies, we can generate the gRPC python code that is needed.

> Note: We keep the latest generated protobuf files in our git repo.
> So, you can skip this step in general (unless you are updating the protobuf/gRPC definitions).
    
    cd proto
    make proto_py

## Running from Source

Finally, all the pieces are in place so that we can install this tool.

    pip install -e .

Now, the `routeviews-google-upload` CLI tools will be available! 

> As this is installed in editable mode, any updates made to the source code will be reflected immediately in your shell session.

## Dependency Management

We use `requirements.txt` to define all dependencies for development, while `setup.py` holds a *looser* list of package that are installed when the package is installed via pip.
This follows [general Python practices, (discussed on python.org)](https://packaging.python.org/discussions/install-requires-vs-requirements/#install-requires)

### Development Dependencies: `requirements.txt`

`requirements.txt` holds all of the development dependencies for this project.

If you make changes to dependencies, be sure to update requirements.txt.
The following command will update requirements.txt (and will correctly omit 'editable' packages).

    pip freeze | grep -v ^-e > requirements.txt

### Production Dependencies: `setup.py`

In `setup.py`, there is the **minimum** List of packages required for installation: `install_requires`.
This list should follow best practices, I.e.,

1. do **NOT** pin specific versions, and 
2. do **NOT** specify sub-dependencies.

#### Testing Production Dependencies

> The normal development workflow will install dependencies specified in `requirements.txt` -- here we omit that step so that we can test the setup.py 

Production dependencies outlined in `setup.py` **should be tested when they are created/updated**, to ensure the dependencies are sufficiant.
Create a virtual environment specifically for this purpose:

    python3 -m venv venv-setup.py
    source venv-setup.py/bin/activate
    pip install --editable .

Ensure your package tools/tests work as expected using this `venv-setup.py`!

## Automated Testing

pytest is used to automatically testing this project.

Simply run the `pytest` CLI tool from the local directory to run all of the tests!

    $ pytest
    =========================== test session starts ============================
    platform linux -- Python 3.x.y, pytest-6.x.y, py-1.x.y, pluggy-1.x.y
    cachedir: $PYTHON_PREFIX/.pytest_cache
    ... omitted for brevity...
