# Kafka Web App for Confidential Containers on Azure

## About Web App

This is based on the Next.js app starter that contains a frontend in React and backend.
Every five seconds the frontend calls the backend endpoint /api/data and returned is:

```json
{
  "message":"<your-kafka-message-here>"
}
```

Where the message is either encrypted (when running without a security policy in a Confidential Container environment or on a non-confidential environment) or decrypted (when running in a Confidential Container environment with a security policy)

## Instructions

Build OCI image of the web app by running: `docker build -t <image-name>:<tag> .`
Push image to Azure Container Registry:
`az acr login -n <registry-name>`
`docker push <image-name>:<tag>`
Deploy application with `kubectl`
