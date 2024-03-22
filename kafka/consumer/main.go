// --------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"
)

var keyEnabled bool

const eventHubNamespace = "EVENTHUB_NAMESPACE"
const eventHub = "EVENTHUB"

func main() {
	relayMessage := make(chan string)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	t, err := template.ParseFiles("/webtemplates/index.html")
	if err != nil {
		log.Printf("Error parsing templates: %s", err.Error())
		os.Exit(1)
	}

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	eventHubNamespace := fmt.Sprintf(
		"%s.servicebus.windows.net",
		getEnv(eventHubNamespace))

	if err != nil {
		fmt.Println(err.Error())
	}

	// Event Hubs processor
	consumerClient, err := azeventhubs.NewConsumerClient(
		eventHubNamespace,
		getEnv(eventHub),
		azeventhubs.DefaultConsumerGroup,
		credential,
		nil)

	if err != nil {
		fmt.Println(err.Error())
	}

	defer consumerClient.Close(context.TODO())

	partitionClient, err := consumerClient.NewPartitionClient("0", &azeventhubs.PartitionClientOptions{
		StartPosition: azeventhubs.StartPosition{
			Earliest: to.Ptr(false),
		},
	})

	if err != nil {
		panic(err)
	}

	defer partitionClient.Close(context.TODO())

	getRoot := func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			Encrypted bool
			Message   string
		}{
			Encrypted: keyEnabled,
		}
		timer := time.NewTimer(10 * time.Second)
		select {
		case data.Message = <-relayMessage:
			log.Printf("got / request")
		case <-timer.C:
			data.Message = "Timeout waiting to read data from Kafka.  Please refresh the page to try again."
		}
		err := t.Execute(w, data)
		if err != nil {
			log.Printf("Unable to serve webpage: %s", err.Error())
		}
	}

	http.HandleFunc("/", getRoot)
	http.Handle("/web/", http.StripPrefix("/web", http.FileServer(http.Dir("/web"))))
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "/web/favicon.ico")
	})

	go func() {
		err = http.ListenAndServe(":3333", nil)
		if errors.Is(err, http.ErrServerClosed) {
			log.Printf("server closed\n")
		} else if err != nil {
			log.Printf("error starting server: %s\n", err)
			os.Exit(1)
		}
	}()

	key, err := retrieveKey()
	if err != nil {
		log.Printf("Unable to retrieve key: %s", err.Error())
	}

	for {
		// Will wait up to 1 minute for 100 events. If the context is cancelled (or expires)
		// you'll get any events that have been collected up to that point.
		receiveCtx, cancel := context.WithTimeout(context.TODO(), time.Second*10)
		log.Println("Start to receive events ")
		events, err := partitionClient.ReceiveEvents(receiveCtx, 100, nil)
		cancel()

		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			panic(err)
		}

		for _, event := range events {

			// We're assuming the Body is a byte-encoded string. EventData.Body supports any payload
			// that can be encoded to []byte.
			message := string(event.Body)
			if key != nil {
				annotationBytes, err := base64.StdEncoding.DecodeString(message)
				if err != nil {
					log.Printf("error decoding message value %q", err.Error())
				}
				plaintext, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, key, annotationBytes, nil)
				if err != nil {
					log.Printf("error decrypting message %q", err.Error())
				}
				message = string(plaintext)
			}
			select {
			case relayMessage <- message:
			default:
			}
			log.Printf("Message received: %s\n", message)
		}

		select {
		case sig := <-signals:
			log.Printf("Got signal: %v", sig)
			return
		default:
			log.Println("Have not reiceved signal yet. Continue processing..")
		}

	}

}

var datakey struct {
	Key string `json:"key"`
}

func getEnv(envName string) string {
	value := os.Getenv(envName)
	if value == "" {
		fmt.Println("Environment variable '" + envName + "' is missing")
		os.Exit(1)
	}
	return value
}

func retrieveKey() (*rsa.PrivateKey, error) {
	client := &http.Client{}
	var data = strings.NewReader("{\"maa_endpoint\": \"" + os.Getenv("SkrClientMAAEndpoint") + "\", \"akv_endpoint\": \"" + os.Getenv("SkrClientAKVEndpoint") + "\", \"kid\": \"" + os.Getenv("SkrClientKID") + "\"}")
	req, err := http.NewRequest("POST", "http://localhost:8080/key/release", data)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	var bodyText []byte
	if resp != nil && resp.Body != nil {
		limitSize := resp.ContentLength
		const mb134 = 1 << 27 //134MB
		if limitSize < 1 || limitSize > mb134 {
			limitSize = mb134
		}
		bodyText, _ = io.ReadAll(io.LimitReader(resp.Body, int64(limitSize)))
		resp.Body.Close()
	}
	if err != nil {
		log.Printf("Error response body from SKR: %s", bodyText)
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 207 {
		return nil, fmt.Errorf("unable to retrieve key from SKR.  Response Code %d.  Message %s", resp.StatusCode, string(bodyText))
	}

	if err := json.Unmarshal(bodyText, &datakey); err != nil {
		log.Printf("retrieve key unmarshal error: %s", err.Error())
	}

	key, err := RSAPrivateKeyFromJWK([]byte(datakey.Key))
	if err != nil {
		log.Printf("construct private rsa key from jwk key error: %s", err.Error())
	}
	keyEnabled = true

	return key, nil

}

func RSAPrivateKeyFromJWK(key1 []byte) (*rsa.PrivateKey, error) {

	var jwkData struct {
		N string `json:"n"`
		E string `json:"e"`
		D string `json:"d"`
		P string `json:"p"`
		Q string `json:"q"`
	}

	if err := json.Unmarshal(key1, &jwkData); err != nil {
		log.Println(err.Error())
	}
	n, err := base64.RawURLEncoding.DecodeString(jwkData.N)
	if err != nil {
		log.Println(err.Error())
	}
	e, err := base64.RawURLEncoding.DecodeString(jwkData.E)
	if err != nil {
		log.Println(err.Error())
	}
	d, err := base64.RawURLEncoding.DecodeString(jwkData.D)
	if err != nil {
		log.Println(err.Error())
	}
	p, err := base64.RawURLEncoding.DecodeString(jwkData.P)
	if err != nil {
		log.Println(err.Error())
	}
	q, err := base64.RawURLEncoding.DecodeString(jwkData.Q)
	if err != nil {
		log.Println(err.Error())
	}

	key := &rsa.PrivateKey{
		PublicKey: rsa.PublicKey{
			N: new(big.Int).SetBytes(n),
			E: int(new(big.Int).SetBytes(e).Int64()),
		},
		D: new(big.Int).SetBytes(d),
		Primes: []*big.Int{
			new(big.Int).SetBytes(p),
			new(big.Int).SetBytes(q),
		},
	}

	return key, nil
}
