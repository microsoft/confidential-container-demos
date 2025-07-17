"""
This file is the main entry point for the Azure Kubernetes Service (AKS) 
hello-world attestation report example.
"""
import subprocess
import os
import sys
sys.path.append("..")
# pylint: disable=no-name-in-module
# pylint: disable=wrong-import-position
# pylint: disable=import-error
from util.util import get_html_str

def index():
    """Function that gets the HTML content for the AKS hello-world attestation report."""
    return get_html_str(
        "https://azure.microsoft.com/svghandler/kubernetes-service?width=600&height=315",
        "Pods on Azure Kubernetes Service"
    )

# main driver function
if __name__ == '__main__':
    HTML = index()

    FILENAME = "/etc/nginx/html/index.html"
    os.makedirs(os.path.dirname(FILENAME), exist_ok=True)
    with open(FILENAME, "w", encoding="UTF-8") as f:
        f.write(HTML)

    output = (subprocess.run(["/usr/sbin/nginx", "-g", "daemon off;"],
                          capture_output=True, encoding="UTF-8", check=False)).stdout

    print(output)
