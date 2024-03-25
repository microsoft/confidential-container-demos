# Encrypted Kafka Message Example 

## Table of Contents
- [Description](#description)
- [Step by Step Example](#step-by-step-example)

### Description

Apache Kafka is a powerful distributed data store designed for efficiently ingesting and processing streaming data in real-time. It offers numerous advantages such as scalability, data durability, and low latency. However, it's essential to note that an out-of-the-box Apache Kafka installation does not provide data encryption at rest. By default, all data traffic is transmitted in plain text, potentially allowing unauthorized access to sensitive information. While Apache Kafka does support data encryption in transit using SSL or SASL_SSL, as of today, data at rest encryption is currently not natively supported. To ensure end-to-end data security, including data in transit, at rest, heap dumps, and log files, users need to implement end-to-end encryption. 

In this example, we demonstrate the implementation of end-to-end encryption for Kafka messages using encryption keys managed by AKV/mHSM. The key is only released when the Kafka consumer runs within a confidential container environment with Secure Key Release(skr) container injected into the pod.

This example comprises three components: 

Kafka Cluster: A simple kafka cluster deployed in the kafka namespace on an AKS cluster. 

Kafka Producer: A kafka producer running as a vanilla k8s pod that sends encrypted user-configured messages using a public key to a kafka topic. 

Kafka Consumer: A Kafka consumer pod running with the kata-cc runtime, equipped with a secure key release container to retrieve the private key for decrypting Kafka messages and render the messages to web UI. 

### Step by Step Example 

#### Enable Confidential Container on AKS cluster during creation.  

```
az aks create -g myResourceGroup -n myManagedCluster –kubernetes-version <1.24.0 and above> --os-sku AzureLinux –vm-size <VM sizes capable of nested SNP VM> --workload-runtime <kataCcIsolation> 
```

#### Enable workload identities on the cluster.  

If the cluster you have was not created with both `--enable-oidc-issuer` and `--enable-workload-identity`. Please issue the following command: 

```bash
# This is the name of the resource group the cluster resides in
export RESOURCE_GROUP="" 
# name of the cluster 
export CLUSTER_NAME="" 
az aks update -g "${RESOURCE_GROUP}" -n "${CLUSTER_NAME}" --enable-oidc-issuer --enable-workload-identity 
az aks get-credentials --name "${CLUSTER_NAME}" --resource-group "${RESOURCE_GROUP}" --overwrite-existing
``` 

#### Setup Federated Identity using Managed Identity as the Parent Resource 

```bash
# This is the region of the resource group your AKS cluster resides 
export LOCATION="westcentralus"  
# This is the kubernetes namespace you intend to run kafka consumer workload
export SERVICE_ACCOUNT_NAME="workload-identity-sa"  
export SUBSCRIPTION="$(az account show --query id --output tsv)" 
export USER_ASSIGNED_IDENTITY_NAME="myIdentity"  
export FEDERATED_IDENTITY_CREDENTIAL_NAME="myFedIdentity"  
```

Get the OIDC Issuer URL and save it to an environmental variable using the following command. Replace the default value for the arguments -n, which is the name of the cluster.  

```bash
export AKS_OIDC_ISSUER="$(az aks show -n "${CLUSTER_NAME}" -g "${RESOURCE_GROUP}" --query "oidcIssuerProfile.issuerUrl" -otsv)" 
```

Create a managed identity in the same resource group your AKS cluster resides in and export the `USER_ASSIGNED_IDENTITY_NAME` of the managed identity:  

```bash
az identity create --name "${USER_ASSIGNED_IDENTITY_NAME}" --resource-group "${RESOURCE_GROUP}" --location "${LOCATION}" --subscription "${SUBSCRIPTION}" 
export USER_ASSIGNED_CLIENT_ID="$(az identity show --resource-group "${RESOURCE_GROUP}" --name "${USER_ASSIGNED_IDENTITY_NAME}" --query 'clientId' -otsv)" 
```

Once you completed above, obtain the resource id of the newly created managed identity because [setup-key.sh](setup-key.sh) relies on the it. Issue the following command: 

```bash
# RESOURCE_GROUP is the name of the resource group your newly created managed identity resides in. 
# USER_ASSIGNED_IDENTITY_NAME is the name of the newly created managed identity. 
export MANAGED_IDENTITY="$(az identity show --resource-group "${RESOURCE_GROUP}" --name "${USER_ASSIGNED_IDENTITY_NAME}" --query 'id' -otsv)"
```

Create a service account: 

```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    azure.workload.identity/client-id: ${USER_ASSIGNED_CLIENT_ID}
  name: ${SERVICE_ACCOUNT_NAME}
EOF
```
 
The following output resembles successful creation of the identity: 

```bash
Serviceaccount/workload-identity-sa created 
```

Create the federated identity credential between the managed identity, service account issuer, and subject using the az identity federated-credential create command. 

```bash
az identity federated-credential create --name ${FEDERATED_IDENTITY_CREDENTIAL_NAME} --identity-name ${USER_ASSIGNED_IDENTITY_NAME} --resource-group ${RESOURCE_GROUP} --issuer ${AKS_OIDC_ISSUER} --subject system:serviceaccount:default:${SERVICE_ACCOUNT_NAME} 
```

#### Obtain Attestation Endpoint

Below are the MAA endpoints for the four regions Confidential Container AKS is currently available. 

- East US: sharedeus.eus.attest.azure.net	
- West US: sharedwus.wus.attest.azure.net
- North Europe: sharedneu.neu.attest.azure.net
- West Europe: sharedweu.weu.attest.azure.net

Set the `MAA_ENDPOINT` environment variable. We are using `East US` as an example: 

```bash
export MAA_ENDPOINT="sharedeus.eus.attest.azure.net"
```

#### Setup dependency resources (AKV/mHSM)

Setup dependency resources (AKV/mHSM):  The user needs to instantiate an [premium Azure Key Vault(AKV)](https://learn.microsoft.com/en-us/azure/key-vault/general/overview) or a [Managed Hardware Security Module(mHSM)]((https://docs.microsoft.com/en-us/azure/key-vault/managed-hsm/overview)) resource that supports storing keys in an HSM. 
Important NOTE: In this demo, we include both AKV and mHSM related instructions and the script for setting up RSA asymmetric keys supports both AKV and mHSM. 
Although using an mHSM is recommended for production, due to its high cost, we recommend using AKV for running this demo. 

Set the `AZURE_AKV_RESOURCE_ENDPOINT` environment variable:

```bash
export AZURE_AKV_RESOURCE_ENDPOINT="<akv-name>.vault.azure.net or <mHSM-name>.managedhsm.azure.net"
```

#### Create Azure Event Hub Resource 

```bash 
export EVENTHUB_NAMESPACE=kafka-demo-ehubns
export EVENTHUB=kafka-demo-topic
az eventhubs namespace create --name $EVENTHUB_NAMESPACE --resource-group $RESOURCE_GROUP --location $LOCATION --enable-kafka true --enable-auto-inflate false
az eventhubs eventhub create --name $EVENTHUB --resource-group $RESOURCE_GROUP --namespace-name $EVENTHUB_NAMESPACE --partition-count 10
```

#### Setup role access for the managed identity 

Assign the managed identity you created `USER_ASSIGNED_IDENTITY_NAME` in "Deploy and configure workload identity" step with the correct access permissions. 

The managed identity needs Key Vault Crypto Officer and Key Vault Crypto User roles if using AKV key vault or Managed HSM Crypto Officer and Managed HSM Crypto User roles for /keys if using AKV managed HSM. 
The managed identity you created will be used for accessing the key vault during workload runtime. 
Thus, this step is for granting key vault access to the managed identity you created. 

If using mHSM, you can do so by going into the mHSM you created (you may need to select `Show Hidden Types` in your resource group), Local RBAC, Add, and in the Search box adding the Client ID of the managedIdentity you created earlier.

Note: You need to be the owner of the event hub to do the following. Follow the instruction [here](https://learn.microsoft.com/en-us/azure/event-hubs/passwordless-migration-event-hubs?tabs=roles-azure-portal%2Csign-in-azure-cli%2Cdotnet%2Cazure-portal-create%2Cazure-portal-associate%2Capp-service-identity%2Capp-service-connector%2Cassign-role-azure-portal), assign the managed identity `USER_ASSIGNED_IDENTITY_NAME` with the Azure Event Hubs Data Sender and Azure Event Hubs Data Reader role for the Event Hub created in the prior step. 

#### Setup role access for your own alias. 

This demo depends on users running [setup-key.sh](setup-key.sh) script to setup RSA asymmetric keys in AKV/mHSM. The script is run on local environment. Thus, users need to setup role access for their alias as well in order to create keys in AKV/mHSM: 

```bash
# using mHSM
az keyvault role assignment create --hsm-name mhsm-name --assignee alias@microsoft.com --role "Managed HSM Crypto User" --scope /keys --subscription 85c****bdf8
az keyvault role assignment create --hsm-name mhsm-name --assignee alias@microsoft.com --role "Managed HSM Crypto Officer" --scope /keys --subscription 85c****bdf8

# using AKV. Replace <alias> with your own alias.  
AKV_SCOPE=`az keyvault show --name <AZURE_AKV_RESOURCE_NAME> --query id --output tsv` 
az role assignment create --role "Key Vault Crypto Officer" --assignee <alias>@microsoft.com --scope $AKV_SCOPE
az role assignment create --role "Key Vault Crypto User" --assignee <alias>@microsoft.com --scope $AKV_SCOPE

```

NOTE: Only the subscription owner can setup role access for AKV/mHSM, so if you are seeing authorization related error messages during role access setup steps, please seek out the proper personel to setup role access. 


#### Generate Consumer Pod YAML Files (Only If Using Azure Event Hub Resource)

```bash
export SIDECAR_IMAGE="mcr.microsoft.com/aci/skr:2.7"
export CONSUMER_IMAGE="mcr.microsoft.com/acc/samples/kafka/consumer:2.0"
SIDECAR_IMAGE=$(echo $SIDECAR_IMAGE | sed 's/\//\\\//g')
CONSUMER_IMAGE=$(echo $CONSUMER_IMAGE | sed 's/\//\\\//g')
sed -i 's/$EVENTHUB_NAMESPACE/'"$EVENTHUB_NAMESPACE"'/g; s/$EVENTHUB/'"$EVENTHUB"'/g; s/$MAA_ENDPOINT/'"$MAA_ENDPOINT"'/g; s/$AZURE_AKV_RESOURCE_ENDPOINT/'"$AZURE_AKV_RESOURCE_ENDPOINT"'/g; s/$CONSUMER_IMAGE/'"$CONSUMER_IMAGE"'/g; s/$SIDECAR_IMAGE/'"$SIDECAR_IMAGE"'/g' consumer/consumer.yaml
```

#### Generate Security Policy 

Install the Azure confcom CLI extension by running the following command: 

```bash
az extension add --name confcom
```

Generate the security policy for the Kafka consumer YAML file and obtain the hash of the security policy. Set `WORKLOAD_MEASUREMENT` to the hash of the security policy because `setup-key.sh` script depends on this env var. Run the following commands: 

```bash
$ export WORKLOAD_MEASUREMENT=$(az confcom katapolicygen -y consumer/consumer.yaml --print-policy | base64 --decode | sha256sum | cut -d' ' -f1)
```

- If az confcom katapolicygen returns an error, run the following commands and try again:

```bash
$ az extension remove --name confcom
$ az extension add --source https://acccliazext.blob.core.windows.net/confcom/confcom-0.3.13-py3-none-any.whl -y
```

#### Prepare RSA Encryption/Decryption Key

Use the provided script [setup-key.sh](setup-key.sh) to prepare encryption key for the workload. 


Run the script: 
```bash 
$ bash setup-key.sh "kafka-demo-pipeline" $AZURE_AKV_RESOURCE_ENDPOINT

```

The script generates an RSA asymmetric key pair (public and private keys) in mHSM under the secure key release client key id([SkrCLientKID](consumer/consumer.yaml#L29)), creates a key release policy with user-configured data, uploads the key release policy to the Azure mHSM under the `SkrCLientKID` and downloads the public key. 

Verify the keys have been successfully uploaded to the AKV. <Name of AKV> is the name of the AKV. Eg. If you have a AKV and its full url is `my-akv.vault.azure.net`, then my-akv is <Name of AKV>

```bash 
$ az account set --subscription "Subscription ID"
# using mHSM
$ az keyvault key list --hsm-name <Name of mHSM> -o table | grep kafka-demo-pipeline
# using AKV
az keyvault key list --vault-name <Name of AKV> -o table | grep kafka-demo-pipeline
```


#### Generate Producer Pod YAML Files (Only If Using Azure Event Hub Resource)

```bash
export PRODUCER_IMAGE="mcr.microsoft.com/acc/samples/kafka/producer:2.0"
PRODUCER_IMAGE=$(echo $PRODUCER_IMAGE | sed 's/\//\\\//g')
sed -i 's/$EVENTHUB_NAMESPACE/'"$EVENTHUB_NAMESPACE"'/g; s/$EVENTHUB/'"$EVENTHUB"'/g; s/$PRODUCER_IMAGE/'"$PRODUCER_IMAGE"'/g ' producer/producer.yaml

awk '{printf "%s", $0; if (NR > 1) printf "auniqueidentifier"} END {print ""}' kafka-demo-pipeline-pub.pem > kafka-demo-pipeline-pub-temp.pem
export PUBKEY=$(cat kafka-demo-pipeline-pub-temp.pem)
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
```

#### Deployment

Deploy the consumer and producer respectively using the producer and consumer YAML files above, and obtain the IP address of the web service using the following commands:

```bash
$ kubectl apply –f consumer/consumer.yaml  
$ kubectl apply –f producer/producer.yaml  
$ kubectl get svc consumer
```
Copy and paste the IP address of the consumer service into your web browser and observe the decrypted messages. You should also attempt to run the consumer as a regular Kubernetes pod by removing the skr container and kata-cc runtime class spec. Since we are not running the consumer with kata-cc runtime class, we no longer need the policy. Remove the entire policy. Observe the messages again on the web UI after redeploying the workload. Messages will appear as base64-encoded ciphertext because the private encryption key cannot be retrieved. The key cannot be retrieved because the consumer is no longer running in a confidential environment, and the skr container is missing, preventing decryption of messages.

This example demonstrates how to enhance the security of your Apache Kafka cluster/application by implementing end-to-end encryption for both data in transit and at rest using confidential kata-cc AKS container, allowing key retrieval from Azure mHSM, thus safeguarding your data from potential security threats.

#### Cleanup: 

```bash
az eventhubs eventhub delete -n $EVENT_HUB_NAME -g $RESOURCE_GROUP --namespace-name $EVENT_HUB_NAMESPACE
az eventhubs namespace delete -n $EVENT_HUB_NAMESPACE -g $RESOURCE_GROUP
```

