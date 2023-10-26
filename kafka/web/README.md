# Kafka Web App for Confidential Containers on Azure

## About Web App

This is based on the Next.js app starter that contains a frontend in React and backend.
Every five seconds the frontend calls the backend endpoint /api/data and is returned:

```json
{
  "message":"<your-kafka-message-here>"
}
```

Where the message is either encrypted (when running without a security policy in a Confidential Container environment or on a non-confidential environment)
