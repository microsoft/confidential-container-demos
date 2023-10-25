#!/bin/bash

# Copyright (c) Microsoft Corporation.
# Licensed under the MIT License.

set -e

echo getting verbose-report
curl -L https://github.com/microsoft/confidential-sidecar-containers/releases/latest/download/verbose-report > verbose-report

cp verbose-report ./AKS/verbose-report
mv verbose-report ./ACI/app/verbose-report