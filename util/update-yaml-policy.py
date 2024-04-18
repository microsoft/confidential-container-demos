#!/usr/bin/env python3

import argparse
import base64
import json
import re
import sys
import yaml


def update_policy(encoded_policy: str, write: bool, exec: bool) -> str:
    """
    Decodes the given base64 policy, finds the JSON in 'policy_data := { ... }',
    updates `request_defaults.WriteStreamRequest` and `request_defaults.ExecProcessRequest.commands`,
    then re-encodes the policy and returns the updated base64.
    """
    # Decode base64 => text
    decoded_policy = base64.b64decode(encoded_policy).decode("utf-8")

    # Regex to find: policy_data := { ... }
    pattern = r'(policy_data\s*:=\s*)(\{[\s\S]*?\n\})'
    match = re.search(pattern, decoded_policy)
    if not match:
        raise ValueError("Could not find 'policy_data := { ... }' block in the decoded policy.")

    prefix = match.group(1)            # "policy_data := "
    policy_data_block = match.group(2) # The raw JSON object { ... }

    # Convert block to Python dict
    policy_data_dict = json.loads(policy_data_block)

    # Update fields
    if write:
        policy_data_dict["request_defaults"]["WriteStreamRequest"] = True
    if exec:
        policy_data_dict["request_defaults"]["ExecProcessRequest"]["commands"] = ["/bin/bash", "/bin/sh"]

    # Convert dict back to JSON
    updated_block = prefix + json.dumps(policy_data_dict, indent=2) + "\n"

    # Splice it back into the full Rego policy text
    modified_policy = (
        decoded_policy[: match.start()] +
        updated_block +
        decoded_policy[match.end():]
    )

    # Re-encode as base64
    return base64.b64encode(modified_policy.encode("utf-8")).decode("utf-8")


def parse_and_update(file_path: str, write: bool, exec: bool) -> str:
    """
    Loads all docs from the YAML file, finds the Pod annotation:
      metadata.annotations["io.katacontainers.config.agent.policy"]
    Updates that policy, overwrites it, writes back to the file,
    and returns the new base64 policy string.
    """
    with open(file_path, "r") as f:
        docs = list(yaml.safe_load_all(f))

    new_base64 = None
    updated_docs = []

    for doc in docs:
        if doc is None:
            updated_docs.append(doc)
            continue

        # Update only if this doc is a Pod with the relevant annotation
        if doc.get("kind") == "Pod":
            annotations = doc.setdefault("metadata", {}).setdefault("annotations", {})
            old_policy = annotations.get("io.katacontainers.config.agent.policy")
            if old_policy:
                new_base64 = update_policy(old_policy, write, exec)
                # Overwrite in the annotation
                annotations["io.katacontainers.config.agent.policy"] = new_base64

        updated_docs.append(doc)

    # Write everything back to the same file
    with open(file_path, "w") as f:
        yaml.safe_dump_all(updated_docs, f)

    # Return the new policy. If no policy found, we'll return None.
    return new_base64


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Update the base64 policy in a Pod YAML.")
    parser.add_argument("--file", required=True, help="Path to the YAML file with the Pod definition.")
    parser.add_argument("--write", action='store_true', help="Update the policy WriteStream to True")
    parser.add_argument("--exec", action='store_true', help="Update the policy to allow /bin/bash and /bin/sh ExecProcessRequests.")
    args = parser.parse_args()

    new_policy = parse_and_update(args.file, args.write, args.exec)

    # Print the updated base64 policy to stdout
    if new_policy is None:
        print("No Pod with an existing policy annotation was found.", file=sys.stderr)
        sys.exit(1)
    else:
        print(new_policy)