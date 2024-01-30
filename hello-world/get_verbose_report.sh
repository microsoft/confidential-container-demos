#!/bin/bash

# Copyright (c) Microsoft Corporation.
# Licensed under the MIT License.

set -e

echo getting verbose-report
curl -L https://github.com/microsoft/confidential-sidecar-containers/releases/latest/download/verbose-report > verbose-report

cp verbose-report ./hello-world/AKS/verbose-report
mv verbose-report ./hello-world/ACI/app/verbose-report