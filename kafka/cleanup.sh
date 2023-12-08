#!/bin/bash

# --------------------------------------------------------------------------------------------
# Copyright (c) Microsoft Corporation. All rights reserved.
# Licensed under the MIT License. See License.txt in the project root for license information.
# --------------------------------------------------------------------------------------------


set -e


kubectl delete -f consumer-example.yaml 2>&1 || true 
kubectl delete -f producer-example.yaml 2>&1 || true 
kubectl -n kafka delete $(kubectl get strimzi -o name -n kafka) 2>&1 || true 
kubectl -n kafka delete -f 'https://strimzi.io/install/latest?namespace=kafka' 2>&1 || true 
kubectl delete namespace kafka  2>&1 || true 

az aks stop --resource-group accct-mariner-kata-aks-testing --name skr-kafka-demo-rg-3417 2>&1 || true 
az aks delete --resource-group accct-mariner-kata-aks-testing --name skr-kafka-demo-rg-3417 --no-wait --yes 2>&1 || true 