#!/bin/bash

# Copyright (c) Microsoft Corporation.
# Licensed under the MIT License.

set -e

echo building get-snp-report
pushd ../../tools/get-snp-report
make 

cp ./bin/verbose-report ../../hello-world/AKS/verbose-report

make clean
popd