# Copyright (c) Microsoft Corporation. All rights reserved.
# Licensed under the MIT License.

import argparse
import json
import os
import yaml
import shutil
import tempfile


def parse_json(file: str):
    with open(file, "r+") as f:
        data = json.load(f)
        data["parameters"]["name"]["defaultValue"] = f'helloworld-aci-{os.environ["WORKFLOW_ID"]}'
        data["parameters"]["image"]["defaultValue"] = os.environ["HELLO_WORLD_IMAGE"]
        # write json back to file
        f.seek(0)
        json.dump(data, f, indent=4)
        f.truncate()

def parse_yaml(file: str):
    with open(file, "r+") as f:
        data = yaml.safe_load(f)
        data["spec"]["containers"][0]["image"] = os.environ["HELLO_WORLD_IMAGE"]
        # write yaml back to file
        f.seek(0)
        yaml.dump(data, f)
        f.truncate()


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Substitute Environment Variables")
    parser.add_argument(
        "--file",
        help="The name of the file to substitute environment variables in",
        type=str,
        required=True,
    )
    parser.add_argument(
        "--file-type",
        help="The type of file being parsed: json or yaml",
        type=str,
        required=True,
    )

    args = parser.parse_args()

    # copy the file to a temporary location
    with tempfile.NamedTemporaryFile(delete=False) as tmp_file:
        tmp_file_path = tmp_file.name
        shutil.copyfile(args.file, tmp_file_path)

        if args.file_type == "json":
            parse_json(tmp_file_path)
        elif args.file_type == "yaml":
            parse_yaml(tmp_file_path)

        # copy updated file back to original location
        shutil.copyfile(tmp_file_path, args.file)