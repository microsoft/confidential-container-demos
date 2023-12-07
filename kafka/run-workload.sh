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
export CLUSTER_NAME="skr-kafka-demo-rg-3195" 

# Check workload idenitty enabled
echo "Checking workload identity enablement on the cluster"
workload_identity_enabled=$(az aks show --name "${CLUSTER_NAME}" --resource-group "${RESOURCE_GROUP}" --query 'securityProfile|workloadIdentity|enabled' -otsv)
oidc_issuer_enabled=$(az aks show --name "${CLUSTER_NAME}" --resource-group "${RESOURCE_GROUP}" --query 'oidcIssuerProfile|enabled' -otsv)
if [ "$workload_identity_enabled" != true ] || [ "$oidc_issuer_enabled" != true ]; then
    echo "Workload identity is not properly enabled on the cluster, enabling workload identity..."
    az aks update -g "${RESOURCE_GROUP}" -n "${CLUSTER_NAME}" --enable-oidc-issuer --enable-workload-identity 
else
    echo "Workload identity is properly enabled on the cluster"
fi


az aks get-credentials --name "${CLUSTER_NAME}" --resource-group "${RESOURCE_GROUP}" --overwrite-existing
# This is the region of the resource group your AKS cluster resides 
export LOCATION="eastus"  
# This is the kubernetes namespace you intend to run kafka consumer workload
export SERVICE_ACCOUNT_NAMESPACE="kafka"  
export SERVICE_ACCOUNT_NAME="workload-identity-sa"  
export SUBSCRIPTION="$(az account show --query id --output tsv)" 
export USER_ASSIGNED_IDENTITY_NAME="accct-mariner-kata-aks-testing-identity"  
export FEDERATED_IDENTITY_CREDENTIAL_NAME="myFedIdentity"  

export AKS_OIDC_ISSUER="$(az aks show -n "${CLUSTER_NAME}" -g "${RESOURCE_GROUP}" --query "oidcIssuerProfile.issuerUrl" -otsv)" 
echo "Setting AKS_OIDC_ISSUER to $AKS_OIDC_ISSUER"

result=$(az identity show --name "${USER_ASSIGNED_IDENTITY_NAME}" --resource-group "${RESOURCE_GROUP}" --subscription "${SUBSCRIPTION}" 2>&1 || true)
if [[ $result == *"not found"* ]]; then
    echo "Identity ${USER_ASSIGNED_IDENTITY_NAME} not found. Creating... "
    az identity create --name "${USER_ASSIGNED_IDENTITY_NAME}" --resource-group "${RESOURCE_GROUP}" --location "${LOCATION}" --subscription "${SUBSCRIPTION}" 
else
    echo Identity ${USER_ASSIGNED_IDENTITY_NAME} already exists. 
fi

export USER_ASSIGNED_CLIENT_ID="$(az identity show --resource-group "${RESOURCE_GROUP}" --name "${USER_ASSIGNED_IDENTITY_NAME}" --query 'clientId' -otsv)"
echo "Setting USER_ASSIGNED_CLIENT_ID to $USER_ASSIGNED_CLIENT_ID"

# RESOURCE_GROUP is the name of the resource group your newly created managed identity resides in. 
# USER_ASSIGNED_IDENTITY_NAME is the name of the newly created managed identity. 
export MANAGED_IDENTITY="$(az identity show --resource-group "${RESOURCE_GROUP}" --name "${USER_ASSIGNED_IDENTITY_NAME}" --query 'id' -otsv)"
echo "Setting MANAGED_IDENTITY to $MANAGED_IDENTITY"

#check kafka namespace exists and create
result=$(kubectl get namespace kafka 2>&1 || true)
if [[ $result == *"not found"* ]]; then
    echo "kafka namespace not found. Create kafka namespace..."
    kubectl create namespace kafka
else
    echo "kafka namespace already exists."
fi


kubectl delete sa -n kafka workload-identity-sa 2>&1 || true
cat <<EOF | kubectl apply -f - 
apiVersion: v1 
kind: ServiceAccount 
metadata: 
  annotations: 
    azure.workload.identity/client-id: ${USER_ASSIGNED_CLIENT_ID} 
  name: ${SERVICE_ACCOUNT_NAME} 
  namespace: ${SERVICE_ACCOUNT_NAMESPACE} 
EOF

# check federated credential existence
result=$(az identity federated-credential show --name ${FEDERATED_IDENTITY_CREDENTIAL_NAME} --identity-name ${USER_ASSIGNED_IDENTITY_NAME} --resource-group ${RESOURCE_GROUP} 2>&1 || true)
if [[ $result == *$AKS_OIDC_ISSUER* ]]; then
    echo "Federated identity already exists"
else
    echo "Federated identity not found. Creating... "
    az identity federated-credential create --name ${FEDERATED_IDENTITY_CREDENTIAL_NAME} --identity-name ${USER_ASSIGNED_IDENTITY_NAME} --resource-group ${RESOURCE_GROUP} --issuer ${AKS_OIDC_ISSUER} --subject system:serviceaccount:${SERVICE_ACCOUNT_NAMESPACE}:${SERVICE_ACCOUNT_NAME} 
fi


# Create Kafka cluster regardless whether resources exist or not. 
kubectl create -f 'https://strimzi.io/install/latest?namespace=kafka' -n kafka 2>&1 || true 
# Apply the `Kafka` Cluster CR file
kubectl apply -f https://strimzi.io/examples/latest/kafka/kafka-persistent-single.yaml -n kafka 2>&1 || true 

echo "Sleep for 1 minute and wait for Kafka cluster to be creating and fully working..."
sleep 60
# Check if SkrClientKID is an empty string
if [ -z "$SkrClientKID" ]; then
    # If it's empty, set its value to demo default DEMO_DEFAULT_SkrClientKID value 
    export SkrClientKID=$DEMO_DEFAULT_SkrClientKID
    echo "SkrClientKID is now set to: $SkrClientKID"
else
    # If it's not empty, do nothing (you can add more logic here if needed)
    echo "SkrClientKID is not empty. Current value is: $SkrClientKID"
    export SkrClientKID=$SkrClientKID
fi

# Check if SkrClientMAAEndpoint is an empty string
if [ -z "$SkrClientMAAEndpoint" ]; then
    # If it's empty, set its value to demo default DEMO_DEFAULT_SkrClientMAAEndpoint value 
    export SkrClientMAAEndpoint=$DEMO_DEFAULT_SkrClientMAAEndpoint
    echo "SkrClientMAAEndpoint is now set to: $SkrClientMAAEndpoint"
else
    # If it's noet empty, do nothing (you can add more logic here if needed)
    echo "SkrClientMAAEndpoint is not empty. Current value is: $SkrClientMAAEndpoint"
    export SkrClientMAAEndpoint=$SkrClientMAAEndpoint
fi

# Check if TOPIC is an empty string
if [ -z "$TOPIC" ]; then
    # If it's empty, set its value to demo default DEMO_DEFAULT_TOPIC value 
    export TOPIC=$DEMO_DEFAULT_TOPIC
    echo "TOPIC is now set to: $DEMO_DEFAULT_TOPIC"
else
    # If it's not empty, do nothing (you can add more logic here if needed)
    echo "TOPIC is not empty. Current value is: $TOPIC"
    export TOPIC=$TOPIC
fi

if [ -z "$AKV_NAME" ]; then
    export SkrClientAKVEndpoint=$(az keyvault show --hsm-name $MHSM_NAME --resource-group $AKV_MHSM_RESOURCE_GROUP --query 'properties|hsmUri' -otsv)
else
    export SkrClientAKVEndpoint=$(az keyvault show --name $AKV_NAME --resource-group $AKV_MHSM_RESOURCE_GROUP --query 'properties|vaultUri' -otsv)
fi

SkrClientAKVEndpoint=${SkrClientAKVEndpoint#"https://"}
SkrClientAKVEndpoint=${SkrClientAKVEndpoint#"http://"}
export SkrClientAKVEndpoint=${SkrClientAKVEndpoint%%/*}


echo "SkrClientKID value is: $SkrClientKID"
echo "SkrClientMAAEndpoint value is: $SkrClientMAAEndpoint"
echo "SkrClientAKVEndpoint value is: $SkrClientAKVEndpoint"
echo "TOPIC value is: $TOPIC"

rm consumer-example.yaml 2>&1 || true 
envsubst <consumer/consumer.yaml> consumer-example.yaml 

# Install confcom extension 
az extension add --name confcom

export WORKLOAD_MEASUREMENT=$(az confcom katapolicygen -y consumer-example.yaml -j consumer/genpolicy-debug-settings.json --print-policy | base64 --decode | sha256sum | cut -d' ' -f1)

chmod +x setup-key.sh

rm $SkrClientKID-info.json 2>&1 || true 
rm $SkrClientKID-pub.pem 2>&1 || true 
rm $SkrClientKID-release-policy.json 2>&1 || true 
bash setup-key.sh $SkrClientKID $SkrClientAKVEndpoint
sleep 2

export PUBKEY=$(cat $SkrClientKID-pub.pem)
rm producer-example.yaml 2>&1 || true 
envsubst <producer/producer.yaml> producer-example.yaml

sed -i '25s/^/            /' producer-example.yaml
sed -i '26s/^/            /' producer-example.yaml
sed -i '27s/^/            /' producer-example.yaml
sed -i '28s/^/            /' producer-example.yaml
sed -i '29s/^/            /' producer-example.yaml
sed -i '30s/^/            /' producer-example.yaml
sed -i '31s/^/            /' producer-example.yaml
sed -i '32s/^/            /' producer-example.yaml
sed -i '33s/^/            /' producer-example.yaml
sed -i '34s/^/            /' producer-example.yaml


kubectl delete -f consumer-example.yaml 2>&1 || true 
sleep 5 
kubectl apply -f consumer-example.yaml 2>&1 || true 
sleep 10

kubectl delete -f producer-example.yaml 2>&1 || true 
sleep 5 
kubectl apply -f producer-example.yaml 2>&1 || true 
