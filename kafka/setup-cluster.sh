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

random_number=$((RANDOM % 10000 + 1))
az aks create --resource-group accct-mariner-kata-aks-testing --name skr-kafka-demo-rg-${random_number} --kubernetes-version 1.28.3 --os-sku AzureLinux --node-vm-size Standard_DC4as_cc_v5 --node-count 1 --enable-oidc-issuer --enable-workload-identity --generate-ssh-keys
az aks get-credentials --resource-group accct-mariner-kata-aks-testing --name skr-kafka-demo-rg-${random_number} --overwrite-existing


cat << EOF > runtimeClass-cc.yaml
kind: RuntimeClass
apiVersion: node.k8s.io/v1
metadata:
    name: kata-cc-isolation
handler: kata-cc
overhead:
    podFixed:
        memory: "160Mi"
        cpu: "250m"
scheduling:
  nodeSelector:
    katacontainers.io/kata-runtime: "true"
EOF

kubectl apply -f runtimeClass-cc.yaml


NODE=$(kubectl get node -o=name)
kubectl label ${NODE} katacontainers.io/kata-runtime=true