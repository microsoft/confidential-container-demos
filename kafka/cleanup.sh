#!/bin/bash

# --------------------------------------------------------------------------------------------
# Copyright (c) Microsoft Corporation. All rights reserved.
# Licensed under the MIT License. See License.txt in the project root for license information.
# --------------------------------------------------------------------------------------------


set -e


kubectl delete -f consumer-example.yaml
kubectl delete -f producer-example.yaml 
kubectl -n kafka delete $(kubectl get strimzi -o name -n kafka)
kubectl -n kafka delete -f 'https://strimzi.io/install/latest?namespace=kafka'
kubectl delete namespace kafka 

az aks stop --resource-group accct-mariner-kata-aks-testing --name skr-kafka-demo-rg-3195 

az aks delete --resource-group accct-mariner-kata-aks-testing --name skr-kafka-demo-rg-3195 --no-wait --yes