from setuptools import setup, find_packages
import codecs
import os.path


def read(rel_path):
    here = os.path.abspath(os.path.dirname(__file__))
    with codecs.open(os.path.join(here, rel_path), 'r') as fp:
        return fp.read()

# From https://packaging.python.org/guides/single-sourcing-package-version/#single-sourcing-the-package-version
def get_version(rel_path):
    for line in read(rel_path).splitlines():
        if line.startswith('__version__'):
            delim = '"' if '"' in line else "'"
            return line.split(delim)[1]
    else:
        raise RuntimeError("Unable to find version string.")


install_requires = [
    "grpcio>=1.0.0",
]

setup(name="routeviews-google-upload",
      packages=find_packages(),
      version=get_version('routeviews_google_upload/__init__.py'),
      license='apache-2.0',  # Chose a license from here: https://help.github.com/articles/licensing-a-repository
      description="CLI tool for uploading RouteViews files to Google Cloud Storage (and other Google Cloud services).",
      author='University of Oregon',
      author_email='rleonar7@uoregon.edu',
      url='https://github.com/morrowc/rv',
      keywords=['RouteViews', 'Google', 'Cloud', 'Storage', 'Backup', 'Archive'],  # Keywords that define your package best
      scripts=['routeviews_google_upload/routeviews-google-upload'],
      install_requires=install_requires,
      setup_requires=install_requires,
      classifiers=[
          'Development Status :: 3 - Alpha',
          # Chose either "3 - Alpha", "4 - Beta" or "5 - Production/Stable" as the current state of your package
          'Intended Audience :: Developers',  # Define that your audience are developers
          'Topic :: Software Development',
          'License :: OSI Approved :: Apache Software License',  # Again, pick a license
          'Programming Language :: Python :: 3.4',
          'Programming Language :: Python :: 3.5',
          'Programming Language :: Python :: 3.6',
          'Programming Language :: Python :: 3.7',
          'Programming Language :: Python :: 3.8',
      ]
      )
