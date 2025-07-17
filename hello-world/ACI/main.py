"""
This file is the main entry point for the Azure Container Instances (ACI)
hello-world attestation report example.
"""
# Importing flask module in the project is mandatory
# An object of Flask class is our WSGI application.
import sys
from flask import Flask

sys.path.append("..")
# pylint: disable=no-name-in-module
# pylint: disable=wrong-import-position
# pylint: disable=import-error
from util.util import get_html_str

# Flask constructor takes the name of
# current module (__name__) as argument.
app = Flask(__name__)

# The route() function of the Flask class is a decorator,
# which tells the application which URL should call
# the associated function.
@app.route('/')
def index():
    """Function that gets the HTML content for the ACI hello-world attestation report."""
    return get_html_str(
        "https://azure.microsoft.com/svghandler/container-instances?width=600&height=315",
        "Containers on Azure Container Instances"
    )


# main driver function
if __name__ == '__main__':

    # run() method of Flask class runs the application
    # on the local development server.
    app.run(host='0.0.0.0', port=80)
