// --------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	crand "crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"
)

const eventHubNamespace = "EVENTHUB_NAMESPACE"
const eventHub = "EVENTHUB"
const msg = "MSG"

var eventId = 0
var logLocation = GetEnv("LOG_FILE")

func main() {
	if len(logLocation) > 0 {
		f, err := os.OpenFile(logLocation, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0740)
		if err != nil {
			log.Printf("Unable to open file log location: %s", err.Error())
		}
		log.SetOutput(f)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Printf("Retrieving Azure Credential failed: %s", err.Error())
		os.Exit(1)
	}

	eventHubNamespace := fmt.Sprintf(
		"%s.servicebus.windows.net",
		GetEnv(eventHubNamespace))

	// Event Hubs producer
	producerClient, err := azeventhubs.NewProducerClient(
		eventHubNamespace,
		GetEnv(eventHub),
		credential,
		nil)

	if err != nil {
		log.Printf("Creating Producer Client failed: %s", err.Error())
		os.Exit(1)
	}

	defer producerClient.Close(context.Background())
	for {
		events := createEventsForDemo()
		newBatchOptions := &azeventhubs.EventDataBatchOptions{}
		// Creates an EventDataBatch, which you can use to pack multiple events together, allowing for efficient transfer.
		batch, err := producerClient.NewEventDataBatch(context.Background(), newBatchOptions)
		if err != nil {
			log.Printf("Creating event batch failed: %s", err.Error())
			os.Exit(1)
		}

		err = batch.AddEventData(events[0], nil)
		if err != nil {
			log.Printf("Adding event data to batch failed: %s", err.Error())
		}

		if err := producerClient.SendEventDataBatch(context.Background(), batch, nil); err != nil {
			log.Printf("Event sending failed %s", err.Error())
		}

		select {
		case sig := <-signals:
			log.Printf("Got signal: %v", sig)
			return
		default:
		}

		time.Sleep(time.Second * 1)
	}
}

func GetEnv(envName string) string {
	value, exists := os.LookupEnv(envName)
	if !exists {
		log.Println("Environment variable '" + envName + "' is not set.")
		os.Exit(1)
	}
	return value
}

func createEventsForDemo() []*azeventhubs.EventData {
	eventId += 1
	rawMessage := GetEnv(msg)
	value := fmt.Sprintf("Message Id %d: %s", eventId, rawMessage)
	encryptedValue, err := encryptMessage(value, "path_to_public_key.pem")
	if err != nil {
		log.Println("Encrypting message failed.")
	}
	return []*azeventhubs.EventData{
		{
			Body: []byte(encryptedValue),
		},
	}
}

func encryptMessage(plaintext string, publicKeyPath string) (string, error) {
	var pubpem []byte
	var err error
	if pkey := GetEnv("PUBKEY"); len(pkey) > 0 {
		pubpem = []byte(pkey)
	}
	if len(pubpem) == 0 {
		pubpem, err = os.ReadFile(publicKeyPath)
		if err != nil {
			return "", fmt.Errorf("failed to read public key file %v", publicKeyPath)
		}
	}
	block, _ := pem.Decode([]byte(pubpem))
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("invalid public key: %v", err)
	}

	var ciphertext []byte
	if pubkey, ok := key.(*rsa.PublicKey); ok {
		ciphertext, err = rsa.EncryptOAEP(sha256.New(), crand.Reader, pubkey, []byte(plaintext), nil)
		if err != nil {
			return "", fmt.Errorf("failed to encrypt with the public key: %v", err)
		}
	} else {
		return "", fmt.Errorf("invalid public RSA key")
	}

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}
