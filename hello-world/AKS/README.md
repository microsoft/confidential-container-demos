# Hello World AKS Example

## Azure Kubernetes Service Confidential Hello World

This sample is a basic Python application used to demonstrate Confidential Pods on Azure Kubernetes Service. In this sample an AMD SEV SNP report containing the container's firmware measurements will be displayed on the web using nginx.

![Hello World Hardware Report](./media/hello-world-cc.png)

The container is hosted publicly on [Azure Container Registry](mcr.microsoft.com/acc/samples/aks/helloworld:1.7).

## Getting Started

Run `docker build -t <image-name:tag> .` from the same directory as the Dockerfile.

## Deploying to Azure Kubernetes Service

### Run the az confcom katapolicygen tool on the helloworld.yaml file

```az confcom katapolicygen -y helloworld.yaml```

### Run kubectl apply on the updated yaml file and the service yaml file

```kubectl apply -f helloworld.yaml```
```kubectl apply -f service.yaml```

### Get the external IP of the LoadBalancer service

```kubectl get svc helloworld-service```

The pod uses a LoadBalancer service to expose the web server to the internet.
The IP address is specified as LoadBalancer Ingress and may take a few seconds to populate after the service is created.

### Check the webpage

```curl http://<LoadBalancer Ingress>:80```

Alternatively, you can navigate to the IP address in a web browser.
