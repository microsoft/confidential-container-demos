import subprocess
import string
import os
import stat
# Importing flask module in the project is mandatory
# An object of Flask class is our WSGI application.
from flask import Flask
# Flask constructor takes the name of
# current module (__name__) as argument.
app = Flask(__name__)

# The route() function of the Flask class is a decorator,
# which tells the application which URL should call
# the associated function.


@app.route('/')
def index():
    filename = "./verbose-report"
    # make sure the file is executable
    if not os.access(filename, os.X_OK):
        # make it executable if it's not
        st = os.stat(filename)
        os.chmod(filename, st.st_mode |
                 stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH)
    out = (subprocess.run(filename,
                          capture_output=True, encoding="UTF-8")).stdout

    formatted_text = out.replace("\n", " ").split(" ")
    formatted_text = [x for x in formatted_text if x != ""]

    def is_hex(x): return all(c in string.hexdigits for c in x)

    out = []
    temp_out = ["<br>"]
    counter = 0
    start_flag = False
    for item in formatted_text:
        # there are some headers before the actual data we want to display
        if "AMD" in item:
            start_flag = True

        if start_flag:
            # add a line break before and after each header
            if item.endswith(":"):
                temp_out.append(item)
                temp_out.append("<br>")
                # bold the header
                out.append("<strong>")
                out.append(" ".join(temp_out))
                out.append("</strong>")
                temp_out = ["<br>"]
                counter = 0

            # these are the header words before the colon at the end of the line
            elif not is_hex(item):
                temp_out.append(item)
                counter = 0
            # fall-through case of data
            else:
                if counter == 2:
                    out.append("<br>")
                    counter = 0
                out.append(item)
                counter += 1

    # ACI image source
    image = "<img src=\"https://azure.microsoft.com/svghandler/container-instances?width=600&height=315\" alt=\"Microsoft ACI Logo\" width=\"600\" height=\"315\"><br>"
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
        "<div>" + "<h1>Welcome to Confidential containers on Azure Container Instances!</h1>" +
        image + " ".join(out) +
        "</div>"
    )


# main driver function
if __name__ == '__main__':

    # run() method of Flask class runs the application
    # on the local development server.
    app.run(host='0.0.0.0', port=80)
