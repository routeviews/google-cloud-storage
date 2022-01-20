from setuptools import setup, find_packages
import codecs
import os.path


def read(rel_path):
    here = os.path.abspath(os.path.dirname(__file__))
    with codecs.open(os.path.join(here, rel_path), 'r') as fp:
        return fp.read()


install_and_setup_requires = [
    "grpcio>=1.0.0",
]


setup(name="routeviews-google-upload",
    packages=find_packages(),
    version='0.1.7',       # Try to follow 'semantic versioning' scheme, e.g. https://semver.org/
    license='apache-2.0',  # Chose a license from here: https://help.github.com/articles/licensing-a-repository
    description="CLI tool for uploading RouteViews files to Google Cloud Storage (and other Google Cloud services).",
    long_description_content_type='text/markdown',
    long_description=read('routeviews_google_upload/README.md'),
    author='University of Oregon',
    author_email='rleonar7@uoregon.edu',
    url='https://github.com/routeviews/google-cloud-storage',
    keywords=['RouteViews', 'Google', 'Cloud', 'Storage', 'Backup', 'Archive'],  # Keywords that define your package best
    entry_points={
        'console_scripts': [
            'routeviews-google-upload=routeviews_google_upload.__main__:main'
        ]
    },
    install_requires=install_and_setup_requires,
    setup_requires=install_and_setup_requires,
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
