#!/bin/bash

# --------------------------------------------------------------------------------------------
# Copyright (c) Microsoft Corporation. All rights reserved.
# Licensed under the MIT License. See License.txt in the project root for license information.
# --------------------------------------------------------------------------------------------


set -e


#These are the demo default values. Please do not change the values of the following variables. 
DEMO_DEFAULT_SkrClientKID=kafka-encryption-demo
DEMO_DEFAULT_SkrClientMAAEndpoint=sharedeus2.eus2.test.attest.azure.net
DEMO_DEFAULT_TOPIC=kafka-demo-topic


SkrClientKID="kafka-encryption-demo"
SkrClientMAAEndpoint=""
TOPIC=""
AKV_NAME=""
MHSM_NAME="accmhsm"
AKV_MHSM_RESOURCE_GROUP="acc-mhsm-rg"

# This is the name of the resource group the cluster resides in
export RESOURCE_GROUP="accct-mariner-kata-aks-testing" 
# name of the cluster 
export CLUSTER_NAME="nov7eastus" 

az extension list -o table 
result=$(az extension list -o table  2>&1 || true)
if [[ $result == *"aks-preview"* ]]; then
    echo "aks-preview already installed, upgrading aks-preview version."
    az extension update --name aks-preview
else
    echo "aks-preview extension not found. Installing aks-preview..."
    az extension add --name aks-preview
fi

az feature register --namespace "Microsoft.ContainerService" --name "KataCcIsolationPreview"
sleep 5
az feature show --namespace "Microsoft.ContainerService" --name "KataCcIsolationPreview"
sleep 5
az provider register --namespace "Microsoft.ContainerService"
sleep 5

random_number=$((RANDOM % 10000 + 1))
az aks create --resource-group accct-mariner-kata-aks-testing --name skr-kafka-demo-rg-${random_number} --kubernetes-version 1.28.3 --os-sku AzureLinux --node-vm-size Standard_DC4as_cc_v5 --workload-runtime KataCcIsolation --node-count 1 --generate-ssh-keys
az aks get-credentials --resource-group accct-mariner-kata-aks-testing --name skr-kafka-demo-rg-${random_number} --overwrite-existing