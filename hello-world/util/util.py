"""
Utility functions for the hello-world applications
"""
import os
import stat
import string
import subprocess


def get_verbose_report():
    """
    Function to run the verbose report script and return its output.
    This is used in both AKS and ACI hello-world applications.
    """
    verbose_report = "./verbose-report"
    # make sure the file is executable
    if not os.access(verbose_report, os.X_OK):
        # make it executable if it's not
        st = os.stat(verbose_report)
        os.chmod(verbose_report, st.st_mode |
                 stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH)
    out = (subprocess.run(verbose_report,
                          capture_output=True, encoding="UTF-8", check=False)).stdout

    formatted_text = out.replace("\n", " ").split(" ")
    formatted_text = [x for x in formatted_text if x != ""]

    return formatted_text

def is_hex(x):
    """
    Check if a string is a valid hexadecimal number.
    
    Parameters:
        x (str): The string to check.
    Returns:
        bool: True if the string is a valid hexadecimal number, False otherwise.
    """
    return all(c in string.hexdigits for c in x)

def get_html_str(url, service_name):
    """
    Generate HTML string for the attestation report.
    
    Parameters:
        url (str): The URL of the image to include in the HTML.
        service_name (str): The name of the service to display in the HTML header.
    Returns:
        str: A complete HTML string containing the attestation report, 
        including headers, formatted text, and an embedded image.
    """
    formatted_text = get_verbose_report()

    html_str = []
    temp_out = ["<br>"]
    counter = 0
    for item in formatted_text:
        # add a line break before and after each header
        if item.endswith(":"):
            temp_out.append(item)
            temp_out.append("<br>")
            # bold the header
            html_str.append("<strong>")
            html_str.append(" ".join(temp_out))
            html_str.append("</strong>")
            temp_out = ["<br>"]
            counter = 0

        # these are the header words before the colon at the end of the line
        elif not is_hex(item):
            temp_out.append(item)
            counter = 0
        # fall-through case of data
        else:
            if counter == 2:
                html_str.append("<br>")
                counter = 0
            html_str.append(item)
            counter += 1

    # ACI image source
    image = f"<img src=\"{url}\" alt=\"Microsoft ACI Logo\" width=\"600\" height=\"315\"><br>"
    style = """
    <style>
        body {
            text-align: center;
            font-family: 'Courier New', monospace;
        }
    </style>
    """
    # put everything together
    return (
        style +
        "<div>" + f"<h1>Welcome to Confidential {service_name}!</h1>" +
        image + " ".join(html_str) +
        "</div>"
    )
