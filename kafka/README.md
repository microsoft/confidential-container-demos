# Encrypted Kafka Message Example 

## Table of Contents
- [Description](#description)
- [Step by Step Example](#step-by-step-example)

### Description
This demonstration shows Kafka consumer running in an confidential computing environment on Mariner Kata AKS and retrieves a RSA private key from managed HSM. The kafka consumer decrypts the encrypted message using the RSA private key and displays the message to a web UI. 

### Step by Step Example 

#### Install a Strimzi Kafka Cluster 

See [strimzi installation instruction](https://strimzi.io/quickstarts/)

Verify the cluster is correctly install by issuing the following command: 
```
kubectl get pod -n kafka 
```

