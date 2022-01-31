import sys
import os
# The next 3 lines enable gRPC to operate even when we are calling this from the context of a 
# python package (e.g. when installed into your user site-package directory).
import routeviews_google_upload
this_package_path = os.path.dirname(routeviews_google_upload.__file__)
sys.path.append(this_package_path)
