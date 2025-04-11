#!/bin/bash

# --------------------------------------------------------------------------------------------
# Copyright (c) Microsoft Corporation. All rights reserved.
# Licensed under the MIT License. See License.txt in the project root for license information.
# --------------------------------------------------------------------------------------------


set -e

# This script creates a RSA key in MHSM with a release policy, then downloads
# the public key and saves the key info

if [ $# -ne 2 ] ; then
	echo "Usage: $0 <KEY_NAME> <AZURE_AKV_RESOURCE_ENDPOINT>"
	exit 1
fi

https="https://"
http="http://"
KEY_NAME=$1

# if https://, http:// and trailing / exists, remove them from url 
AZURE_AKV_RESOURCE_ENDPOINT=${2#$https}
AZURE_AKV_RESOURCE_ENDPOINT=${AZURE_AKV_RESOURCE_ENDPOINT#$http}
AZURE_AKV_RESOURCE_ENDPOINT=${AZURE_AKV_RESOURCE_ENDPOINT%%/*}


MAA_ENDPOINT=${MAA_ENDPOINT#$https}
MAA_ENDPOINT=${MAA_ENDPOINT#$http}
MAA_ENDPOINT=${MAA_ENDPOINT%%/*}

key_vault_name=$(echo "$AZURE_AKV_RESOURCE_ENDPOINT" | cut -d. -f1)

if [[ -z "${MAA_ENDPOINT}" ]]; then
	echo "Error: Env MAA_ENDPOINT is not set. Please set up your own MAA instance or select from a region where MAA is offered (e.g. sharedeus.eus.attest.azure.net):"
	echo ""
	echo "https://azure.microsoft.com/en-us/explore/global-infrastructure/products-by-region/?products=azure-attestation"
	exit 1
fi

policy_file_name="${KEY_NAME}-release-policy.json"

echo { \"anyOf\":[ { \"authority\":\"https://${MAA_ENDPOINT}\", \"allOf\":[ > ${policy_file_name}
echo '{"claim":"x-ms-attestation-type", "equals":"sevsnpvm"},' >> ${policy_file_name}

export EVENTHUB_NAMESPACE=kafka-demo-ehubns
export EVENTHUB=kafka-demo-topic
CONSUMER_IMAGE=$(echo $CONSUMER_IMAGE | sed 's/\//\\\//g')
SIDECAR_IMAGE=$(echo $SIDECAR_IMAGE | sed 's/\//\\\//g')
sed -i 's/$EVENTHUB_NAMESPACE/'"$EVENTHUB_NAMESPACE"'/g; s/$EVENTHUB/'"$EVENTHUB"'/g; s/$SkrClientKID/'"$SkrClientKID"'/g; s/$LOG_FILE/'\"\"'/g; s/$MAA_ENDPOINT/'"$MAA_ENDPOINT"'/g; s/$AZURE_AKV_RESOURCE_ENDPOINT/'"$AZURE_AKV_RESOURCE_ENDPOINT"'/g; s/$CONSUMER_IMAGE/'"$CONSUMER_IMAGE"'/g; s/$SIDECAR_IMAGE/'"$SIDECAR_IMAGE"'/g' consumer/consumer.yaml
echo "Generating Security Policy for consumer"

export WORKLOAD_MEASUREMENT=$(az confcom katapolicygen -y consumer/consumer.yaml --print-policy | base64 --decode | sha256sum | cut -d' ' -f1)
cat consumer/consumer.yaml
if [[ -z "${WORKLOAD_MEASUREMENT}" ]]; then
	echo "Warning: Env WORKLOAD_MEASUREMENT is not set. Set this to condition releasing your key on your security policy matching the expected value.  Recommended for production workloads."
else
	echo {\"claim\":\"x-ms-sevsnpvm-hostdata\", \"equals\":\"${WORKLOAD_MEASUREMENT}\"}, >> ${policy_file_name}
fi

az confcom katapolicygen -y consumer/consumer.yaml

echo {\"claim\":\"x-ms-compliance-status\", \"equals\":\"azure-signed-katacc-uvm\"}, >> ${policy_file_name}
echo {\"claim\":\"x-ms-sevsnpvm-is-debuggable\", \"equals\":\"false\"}, >> ${policy_file_name}

echo '] } ], "version":"1.0.0" }' >> ${policy_file_name}
echo "......Generated key release policy ${policy_file_name}"

# Create RSA key
az keyvault key create --id https://$AZURE_AKV_RESOURCE_ENDPOINT/keys/${KEY_NAME} --ops wrapKey unwrapkey encrypt decrypt --kty RSA-HSM --size 3072 --exportable --policy ${policy_file_name}
echo "......Created RSA key in ${AZURE_AKV_RESOURCE_ENDPOINT}"


# # Download the public key
public_key_file=${KEY_NAME}-pub.pem
rm -f ${public_key_file}

if [[ "$AZURE_AKV_RESOURCE_ENDPOINT" == *".vault.azure.net" ]]; then
    az keyvault key download --vault-name ${key_vault_name} -n ${KEY_NAME} -f ${public_key_file}
	echo "......Downloaded the public key to ${public_key_file}"
elif [[ "$AZURE_AKV_RESOURCE_ENDPOINT" == *".managedhsm.azure.net" ]]; then

    az keyvault key download --hsm-name ${key_vault_name} -n ${KEY_NAME} -f ${public_key_file}
	echo "......Downloaded the public key to ${public_key_file}"
fi

# generate key info file
key_info_file=${KEY_NAME}-info.json
echo {  > ${key_info_file}
echo \"public_key_path\": \"${public_key_file}\", >> ${key_info_file}
echo \"kms_endpoint\": \"$AZURE_AKV_RESOURCE_ENDPOINT\", >> ${key_info_file}
echo \"attester_endpoint\": \"${MAA_ENDPOINT}\" >> ${key_info_file}
echo }  >> ${key_info_file}
echo "......Generated key info file ${key_info_file}"
echo "......Key setup successful!"

sleep 2
PRODUCER_IMAGE=$(echo $PRODUCER_IMAGE | sed 's/\//\\\//g')
sed -i 's/$EVENTHUB_NAMESPACE/'"$EVENTHUB_NAMESPACE"'/g; s/$EVENTHUB/'"$EVENTHUB"'/g; s/$LOG_FILE/'\"\"'/g; s/$PRODUCER_IMAGE/'"$PRODUCER_IMAGE"'/g ' producer/producer.yaml
awk '{printf "%s", $0; if (NR > 1) printf "auniqueidentifier"} END {print ""}' $SkrClientKID-pub.pem > $SkrClientKID-pub-temp.pem
cat $SkrClientKID-pub-temp.pem
export PUBKEY=$(cat $SkrClientKID-pub-temp.pem)
PUBKEY=$(echo $PUBKEY | sed 's/\//\\\//g')
sed -i "s/\$PUBKEY/${PUBKEY}/g" producer/producer.yaml
sed -i 's/auniqueidentifier/\n/g ' producer/producer.yaml
sed -i 's/-----BEGIN PUBLIC KEY-----/-----BEGIN PUBLIC KEY-----\n/g ' producer/producer.yaml
sed -i '25s/^/            /' producer/producer.yaml
sed -i '26s/^/            /' producer/producer.yaml
sed -i '27s/^/            /' producer/producer.yaml
sed -i '28s/^/            /' producer/producer.yaml
sed -i '29s/^/            /' producer/producer.yaml
sed -i '30s/^/            /' producer/producer.yaml
sed -i '31s/^/            /' producer/producer.yaml
sed -i '32s/^/            /' producer/producer.yaml
sed -i '33s/^/            /' producer/producer.yaml
sed -i '34s/^/            /' producer/producer.yaml
cat producer/producer.yaml


kubectl apply -f consumer/consumer.yaml 2>&1
sleep 10
kubectl apply -f producer/producer.yaml 2>&1
sleep 10