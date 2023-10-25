#!/bin/bash

# Copyright (c) Microsoft Corporation.
# Licensed under the MIT License.

set -e

echo building get-snp-report
pushd ../../tools/get-snp-report
make 

cp ./bin/verbose-report ../../hello-world/ACI/app/verbose-report

make clean
popd