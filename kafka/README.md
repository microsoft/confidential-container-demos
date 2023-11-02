# Encrypted Kafka Message Example 

## Table of Contents
- [Description](#description)
- [Step by Step Example](#step-by-step-example)

### Description

Apache Kafka is a powerful distributed data store designed for efficiently ingesting and processing streaming data in real-time. It offers numerous advantages such as scalability, data durability, and low latency. However, it's essential to note that an out-of-the-box Apache Kafka installation does not provide data encryption at rest. By default, all data traffic is transmitted in plain text, potentially allowing unauthorized access to sensitive information. While Apache Kafka does support data encryption in transit using SSL or SASL_SSL, as of today, data at rest encryption is currently not natively supported. To ensure end-to-end data security, including data at rest, heap dumps, and log files, users need to implement end-to-end encryption. 

In this example, we demonstrate the implementation of end-to-end encryption for Kafka messages using encryption keys managed by Azure Managed Hardware Security Modules (mHSM). The key is only released when the Kafka consumer runs within a confidential container with azure attestation secret provisioning container injected into the pod.

This example comprises four components: 

Kafka Cluster: A simple Kafka cluster deployed in the Kafka namespace on the cluster. 

Kafka Producer: A Kafka producer running as a vanilla Kubernetes pod that sends encrypted user-configured messages using a public key to a Kafka topic. 

Kafka Consumer: A Kafka consumer pod running with the kata-cc runtime, equipped with a secure key release container to retrieve the private key for decrypting Kafka messages. 

Web Service: Consumed messages are sent to a web service for display on a web UI. Messages, whether successfully decrypted or not, will be displayed. If not decrypted, they will appear as base64-encoded ciphertext.  

### Step by Step Example 

#### Enable Confidential Container on AKS cluster during creation.  
az aks create -g myResourceGroup -n myManagedCluster –kubernetes-version <1.24.0 and above> --os-sku AzureLinux –vm-size <VM sizes capable of nested SNP VM> --workload-runtime <kataCcIsolation> 

#### Enable workload identities on the cluster.  

Update the AKS cluster using the az aks update command with the `--enable-oidc-issuer` parameter to use the OIDC Issuer.

```
export RESOURCE_GROUP="myResourceGroup" # This is the name of the resource group your AKS cluster resides 
az aks update -g "${RESOURCE_GROUP}" -n myAKSCluster --enable-oidc-issuer --enable-workload-identity
```

Or append `--enable-oidc-issuer` `--enable-workload-identity` parameters to the end of your az aks create command so that the cluster is created to use the OIDC issuer. 

#### Setup Federated Identity using Managed Identity as the Parent Resource 

```
export LOCATION="westcentralus" # This is the region of the resource group your AKS cluster resides 
export SERVICE_ACCOUNT_NAMESPACE="default" # This is the kubernetes namespace you intend to run encfs workload
export SERVICE_ACCOUNT_NAME="workload-identity-sa" 
export SUBSCRIPTION="$(az account show --query id --output tsv)"
export USER_ASSIGNED_IDENTITY_NAME="myIdentity" 
export FEDERATED_IDENTITY_CREDENTIAL_NAME="myFedIdentity" 
```

Get the OIDC Issuer URL and save it to an environmental variable using the following command. 
Replace the default value for the arguments -n, which is the name of the cluster.

```
export AKS_OIDC_ISSUER="$(az aks show -n aks-cluster-name -g "${RESOURCE_GROUP}" --query "oidcIssuerProfile.issuerUrl" -otsv)"
```

Create a managed identity in the same resource group your AKS cluster resides in and export the client_id of the managed identity

```
az identity create --name "${USER_ASSIGNED_IDENTITY_NAME}" --resource-group "${RESOURCE_GROUP}" --location "${LOCATION}" --subscription "${SUBSCRIPTION}"

export USER_ASSIGNED_CLIENT_ID="$(az identity show --resource-group "${RESOURCE_GROUP}" --name "${USER_ASSIGNED_IDENTITY_NAME}" --query 'clientId' -otsv)"
```

Create a service account

```
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    azure.workload.identity/client-id: ${USER_ASSIGNED_CLIENT_ID}
  name: ${SERVICE_ACCOUNT_NAME}
  namespace: ${SERVICE_ACCOUNT_NAMESPACE}
EOF
```

The following output resembles successful creation of the identity:

```
Serviceaccount/workload-identity-sa created
```

Create the federated identity credential between the managed identity, service account issuer, and subject using the az identity federated-credential create command.

```
az identity federated-credential create --name ${FEDERATED_IDENTITY_CREDENTIAL_NAME} --identity-name ${USER_ASSIGNED_IDENTITY_NAME} --resource-group ${RESOURCE_GROUP} --issuer ${AKS_OIDC_ISSUER} --subject system:serviceaccount:${SERVICE_ACCOUNT_NAMESPACE}:${SERVICE_ACCOUNT_NAME}
```

#### Setup dependency resources (AKV/mHSM)

The user needs to instantiate an Azure Key Vault resource that supports storing keys in an HSM: a [Premium vault](https://learn.microsoft.com/en-us/azure/key-vault/general/overview) or an [MHSM resource](https://docs.microsoft.com/en-us/azure/key-vault/managed-hsm/overview). Set the value of [SkrClientAKVEndpoint](consumer.yamlL#33) with the full url of the AKV/mHSM resource. 

#### Obtain Attestation Endpoint 

If you don't already have a valid attestation endpoint, create a [Microsoft Azure Attestation](https://learn.microsoft.com/en-us/azure/attestation/overview) endpoint to author the attestation token and run the following command to get the endpoint value:

```
az attestation show --name "<ATTESTATION PROVIDER NAME>" --resource-group "<RESOURCE GROUP>"
```

Copy the AttestURI endpoint value to [SkrClientMAAEndpoint](consumer.yamlL#31) 

#### Setup role access for the managed identity 

Assign the managed identity you created <USER_ASSIGNED_IDENTITY_NAME> in step 3 with the correct access permissions. The managed identity needs Key Vault Crypto Officer and Key Vault Crypto User roles if using AKV key vault or Managed HSM Crypto Officer and Managed HSM Crypto User roles for /keys if using AKV managed HSM.

#### Install Kafka Cluster 

Install Kafka Cluster: Install the Kafka cluster in the Kafka namespace following the instructions here:  Quickstarts (strimzi.io) 

#### Configure Kafka Consumer

Select an appropriate name for the RSA asymmetric key pair and replace [SkrClientKID](consumer.yamlL#29). You can leave the remaining configuration environment variables as is, with the option to change the Kafka topic. 

#### Generate Security Policy 

To generate security policies, install the Azure confcom CLI extension by following the instructions [here](https://github.com/Azure/azure-cli-extensions/tree/main/src/confcom/azext_confcom#microsoft-azure-cli-confcom-extension-examples)

Generate the security policy for the Kafka consumer YAML file and obtain the hash of the security policy. 

```
az confirm katapolicygen -y consumer.yaml
```

#### Prepare RSA Encryption/Decryption Key

Use the provided script [setup-key-mhsm.sh](setup-key-mhsm.sh) to prepare encryption key for the workload. 
The script depends on several environment variables that we need to set before running the script. 
Replace the value of [WORKLOAD_MEASUREMENT](setup-key-mhsm.shL#18) with the hash of the security policy. 
Replace the value of the [MANAGED_IDENTITY](setup-key-mhsm.shL#17) with the identity Resource ID created in the previous step. 
Replace the [MAA_ENDPOINT](setup-key-mhsm.shL#16) with the MAA endpoint with... 

Run the script: ```bash setup setup-key-mhsm.sh <SkrClientKID> <mHSM-name>``` 

The script generates an RSA asymmetric key pair (public and private keys) in mHSM under the `SkrCLientKID`, creates a key release policy with user-configured data, uploads the key release policy to the Azure mHSM under the `SkrCLientKID` and downloads the public key.  

Once the public key is downloaded, replace the PUBKEY env var on the producer YAML file with the public key.  

#### Deployment

Deploy the consumer, producer, and web service respectively, and obtain the IP address of the web service using the following commands: 

```
kubectl apply –f web-service.yaml  
kubectl apply –f consumer.yaml  
kubectl apply –f producer.yaml  
kubectl get svc –n kafka 
```

Copy and paste the IP address of the web service into your web browser and observe the decrypted messages. You should also attempt to run the consumer as a regular Kubernetes pod by removing the aasp container and kata-cc runtime class spec. Since we are not running the consumer with kata-cc runtime class, we no longer need the policy. Remove the entire policy. Observe the messages again on the web UI after redeploying the workload. Messages will appear as base64-encoded ciphertext because the private encryption key cannot be retrieved. The key cannot be retrieved because the consumer is no longer running in a confidential environment, and the aasp container is missing, preventing decryption of messages. 

 

This example demonstrates how to enhance the security of your Apache Kafka cluster/application by implementing end-to-end encryption for both data in transit and at rest using confidential kata-cc AKS container, allowing key retrieval from Azure mHSM, thus safeguarding your data from potential security threats. 