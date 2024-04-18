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

	"github.com/microsoft/confidential-container-demos/kafka/util"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"
)

const eventHubNamespace = "EVENTHUB_NAMESPACE"
const eventHub = "EVENTHUB"
const msg = "MSG"
const source = "SOURCE"

var eventId = 0
var logLocation = util.GetEnv("LOG_FILE")

func main() {
	if len(logLocation) > 0 {
		f, err := os.OpenFile(logLocation, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0740)
		if err != nil {
			log.Fatalf("Unable to open file log location: %s", err.Error())
		}
		log.SetOutput(f)
		defer f.Close()
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatalf("Retrieving Azure Credential failed: %s", err.Error())
	}

	eventHubNamespace := fmt.Sprintf("%s.servicebus.windows.net", util.GetEnv(eventHubNamespace))

	// Event Hubs producer
	producerClient, err := azeventhubs.NewProducerClient(
		eventHubNamespace,
		util.GetEnv(eventHub),
		credential,
		nil)

	if err != nil {
		log.Fatalf("Creating Producer Client failed: %s", err.Error())
	}

	defer producerClient.Close(context.Background())
	for {
		event := createEventsForDemo()
		newBatchOptions := &azeventhubs.EventDataBatchOptions{}
		// Creates an EventDataBatch, which you can use to pack multiple events together, allowing for efficient transfer.
		batch, err := producerClient.NewEventDataBatch(context.Background(), newBatchOptions)
		if err != nil {
			log.Fatalf("Creating event batch failed: %s", err.Error())
		}

		err = batch.AddEventData(event, nil)
		if err != nil {
			log.Fatalf("Adding event data to batch failed: %s", err.Error())
		}

		if err := producerClient.SendEventDataBatch(context.Background(), batch, nil); err != nil {
			log.Fatalf("Event sending failed %s", err.Error())
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

func createEventsForDemo() *azeventhubs.EventData {
	eventId += 1
	rawMessage := util.GetEnv(msg)
	value := fmt.Sprintf("Message Id %d: %s", eventId, rawMessage)
	log.Printf("Sending message: %s", value)

	encryptedValue, err := encryptMessage(value)
	if err != nil {
		log.Fatalf("Encrypting message failed: %s", err.Error())
	}
	log.Printf("Encrypted message: %s", encryptedValue)
	return &azeventhubs.EventData{
		Body: []byte(encryptedValue),
		Properties: map[string]interface{}{
			"source": util.GetEnv(source),
		},
	}
}

func encryptMessage(plaintext string) (string, error) {
	var pubpem []byte
	var err error
	if pkey := util.GetEnv("PUBKEY"); len(pkey) > 0 {
		pubpem = []byte(pkey)
	}
	block, _ := pem.Decode([]byte(pubpem))
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("invalid public key: %w", err)
	}

	var ciphertext []byte
	if pubkey, ok := key.(*rsa.PublicKey); ok {
		log.Printf("producer modulus (hex head): %x\n", pubkey.N.Bytes()[:32])
		ciphertext, err = rsa.EncryptOAEP(sha256.New(), crand.Reader, pubkey, []byte(plaintext), nil)
		if err != nil {
			return "", fmt.Errorf("failed to encrypt with the public key: %w", err)
		}
	} else {
		return "", fmt.Errorf("invalid public RSA key: %v", pubkey)
	}

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}
