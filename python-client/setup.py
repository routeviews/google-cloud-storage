from setuptools import setup, find_packages
import codecs
import os.path


def read(rel_path):
    here = os.path.abspath(os.path.dirname(__file__))
    with codecs.open(os.path.join(here, rel_path), 'r') as fp:
        return fp.read()


install_and_setup_requires = [
    'grpcio>=1,<2',
    'google-auth>=2,<3',
    'uologging==0.6.1',
    'requests>=2,<3',
]


setup(
    name="routeviews-google-upload",
    version='0.2.0',       # Try to follow 'semantic versioning' scheme, e.g. https://semver.org/
    description="CLI tool for uploading RouteViews files to Google.",
    long_description_content_type='text/markdown',
    long_description=read('docs/user-guide.md'),
    include_package_data=True,
    package_dir={'': 'src'},
    packages=find_packages('src'),
    license='apache-2.0',
    author='University of Oregon',
    author_email='rleonar7@uoregon.edu',
    url='https://github.com/routeviews/google-cloud-storage',
    keywords=['RouteViews', 'Google', 'Cloud', 'Storage', 'Backup', 'Archive'],  # Keywords that define your package best
    entry_points={
        'console_scripts': [
            'routeviews-google-upload=routeviews_google_upload.__main__:main',
            'routeviews-google-upload-test-server=routeviews_google_upload.echo_server:serve',
        ]
    },
    install_requires=install_and_setup_requires,
    setup_requires=install_and_setup_requires,
    classifiers=[
        'Development Status :: 3 - Alpha',
        'Intended Audience :: Developers',
        'Topic :: Software Development',
        'License :: OSI Approved :: Apache Software License',
        'Programming Language :: Python :: 3.6',
        'Programming Language :: Python :: 3.7',
        'Programming Language :: Python :: 3.8',
    ]
)
