#!/bin/bash

# Copyright (c) Microsoft Corporation.
# Licensed under the MIT License.

set -e

export MAA_ENDPOINT="sharedeus2.eus2.attest.azure.net"
export MANAGED_IDENTITY="/subscriptions/85c61f94-8912-4e82-900e-6ab44de9bdf8/resourceGroups/accct-mariner-kata-aks-testing/providers/Microsoft.ManagedIdentity/userAssignedIdentities/accct-mariner-kata-aks-testing-identity" 
export WORKLOAD_MEASUREMENT="9a31d1313c17b5ff7e1d239d60dbe80fa6a8042b3a1fbb3bf5cba69c03afda90"