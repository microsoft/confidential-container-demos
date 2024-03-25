// --------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
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

func main() {

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Println(err.Error())
	}

	eventHubNamespace := fmt.Sprintf(
		"%s.servicebus.windows.net",
		getEnv(eventHubNamespace))

	// Event Hubs producer
	producerClient, err := azeventhubs.NewProducerClient(
		eventHubNamespace,
		getEnv(eventHub),
		credential,
		nil)

	if err != nil {
		log.Println(err.Error())
	}

	defer producerClient.Close(context.Background())
	for {
		events := createEventsForDemo()

		newBatchOptions := &azeventhubs.EventDataBatchOptions{}
		// Creates an EventDataBatch, which you can use to pack multiple events together, allowing for efficient transfer.
		batch, err := producerClient.NewEventDataBatch(context.Background(), newBatchOptions)
		if err != nil {
			panic(err)
		}

		for i := 0; i < len(events); i++ {
			err = batch.AddEventData(events[i], nil)

			if errors.Is(err, azeventhubs.ErrEventDataTooLarge) {
				if batch.NumEvents() == 0 {
					// This one event is too large for this batch, even on its own. No matter what we do it
					// will not be sendable at its current size.
					log.Println("The single event is too large to be sent.")
				}

				// This batch is full - we can send it and create a new one and continue
				// packaging and sending events.
				if err := producerClient.SendEventDataBatch(context.Background(), batch, nil); err != nil {
					log.Printf("Event sending failed %s", err.Error())
				}

				// create the next batch we'll use for events, ensuring that we use the same options
				// each time so all the messages go the same target.
				tmpBatch, err := producerClient.NewEventDataBatch(context.Background(), newBatchOptions)

				if err != nil {
					log.Printf("Creating new batch failed: %s", err.Error())
				}

				batch = tmpBatch

				// rewind so we can retry adding this event to a batch
				i--
			} else if err != nil {
				log.Printf("Errored while adding events to batch: %s", err.Error())
			}
		}

		// if we have any events in the last batch, send it
		if batch.NumEvents() > 0 {
			log.Println("sending events")
			if err := producerClient.SendEventDataBatch(context.Background(), batch, nil); err != nil {
				log.Printf("Error sending events: %s", err.Error())
			}
		}
		log.Println("Rest for 10 seconds before sending additional events.")
		time.Sleep(time.Second * 1)
	}
}

func createEventsForDemo() []*azeventhubs.EventData {
	rand.Seed(time.Now().UnixNano())
	randomId := rand.Intn(90000) + 10000
	rawMessage := getEnv(msg)
	value := fmt.Sprintf("Message Id %d: %s", randomId, rawMessage)
	encryptedValue, err := encryptMessage(value, "path_to_public_key.pem")
	if err != nil {
		log.Println("Encrypting message failed.")
	}
	return []*azeventhubs.EventData{
		{
			Body: []byte(encryptedValue),
		},
		{
			Body: []byte(encryptedValue),
		},
	}
}

func encryptMessage(plaintext string, publicKeyPath string) (string, error) {
	var pubpem []byte
	var err error
	if pkey := getEnv("PUBKEY"); len(pkey) > 0 {
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

func getEnv(envName string) string {
	value := os.Getenv(envName)
	if value == "" {
		fmt.Println("Environment variable '" + envName + "' is missing")
		os.Exit(1)
	}
	return value
}
