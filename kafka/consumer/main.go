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
	"math"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"
	"github.com/microsoft/confidential-container-demos/kafka/util"
)

var keyEnabled bool

const eventHubNamespace = "EVENTHUB_NAMESPACE"
const eventHub = "EVENTHUB"
const source = "SOURCE"

const (
	maxRetries     = 5
	initialBackoff = 1 * time.Second
	maxBackoff     = 30 * time.Second
)

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

	relayMessage := make(chan string)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	t, err := template.ParseFiles("/webtemplates/index.html")
	if err != nil {
		log.Fatalf("Error parsing templates: %s", err.Error())
	}

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatalf("Retrieving Azure Credential failed: %s", err.Error())
	}

	eventHubNamespace := fmt.Sprintf("%s.servicebus.windows.net", util.GetEnv(eventHubNamespace))

	// Event Hubs processor
	consumerClient, err := azeventhubs.NewConsumerClient(
		eventHubNamespace,
		util.GetEnv(eventHub),
		azeventhubs.DefaultConsumerGroup,
		credential,
		nil)

	if err != nil {
		log.Fatalf("Creating Consumer Client failed: %s", err.Error())
	}

	defer consumerClient.Close(context.Background())
	start := time.Now().UTC()

	partitionClient, err := consumerClient.NewPartitionClient(
		"0",
		&azeventhubs.PartitionClientOptions{
			StartPosition: azeventhubs.StartPosition{
				EnqueuedTime: &start,
				Inclusive:    false,
			},
		},
	)

	if err != nil {
		log.Fatalf("Creating Consumer Partition Client failed: %s", err.Error())
	}

	defer partitionClient.Close(context.Background())

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
			log.Fatalf("Unable to serve webpage: %s", err.Error())
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
			log.Fatalf("Error: server closed: %s\n", err.Error())
		} else if err != nil {
			log.Fatalf("error starting server: %s\n", err.Error())
		}
	}()

	err = getStatus()
	if err != nil {
		log.Fatalf("Unable to get SKR status: %s", err.Error())
	}

	key, err := retrieveKey()
	if err != nil {
		log.Fatalf("Unable to retrieve key: %s", err.Error())
	}

	for {
		// Will wait up to 10 seconds for 100 events. If the context is cancelled (or expires)
		// you'll get any events that have been collected up to that point.
		receiveCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		log.Println("Start to receive events.")
		events, err := partitionClient.ReceiveEvents(receiveCtx, 100, nil)
		cancel()

		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			log.Fatalf("Receiving events failed due to the following reason: %s", err.Error())
		}

		for _, event := range events {
			sourceVal := ""
			if val, ok := event.Properties["source"]; ok {
				sourceVal, _ = val.(string)
			}

			if sourceVal != util.GetEnv(source) {
				log.Printf("Skipping event from a different source (source=%s)", sourceVal)
				continue
			}

			fmtTime := event.EnqueuedTime.Format(time.RFC3339)
			log.Printf("Enqueued @ %s  Seq %d", fmtTime, event.SequenceNumber)

			// We're assuming the Body is a byte-encoded string. EventData.Body supports any payload
			// that can be encoded to []byte.
			message := string(event.Body)
			log.Printf("Encrypted message received: %s\n", message)
			if key != nil {
				annotationBytes, err := base64.StdEncoding.DecodeString(message)
				if err != nil {
					log.Fatalf("error decoding message value %s", err.Error())
				}
				plaintext, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, key, annotationBytes, nil)
				if err != nil {
					log.Fatalf("error decrypting message %s", err.Error())
				}
				message = string(plaintext)
			}
			select {
			case sig := <-signals:
				log.Printf("Got signal: %v", sig)
				return
			case relayMessage <- message:
			default:
			}
			log.Printf("Decrypted message: %s\n", message)
		}

		select {
		case sig := <-signals:
			log.Printf("Got signal: %v", sig)
			return
		default:
		}
	}
}

var datakey struct {
	Key string `json:"key"`
}

func WithRetry(operation func() error) error {
	backoff := initialBackoff
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := operation()
		if err == nil {
			return nil // success on this attempt
		}
		lastErr = err // capture for final return if out of tries

		log.Printf("[WithRetry] attempt %d/%d failed (%v). Retrying in %s...",
			attempt, maxRetries, err, backoff)

		time.Sleep(backoff)
		// exponential growth with clamp
		backoff = time.Duration(math.Min(float64(backoff*2), float64(maxBackoff)))
	}

	return fmt.Errorf("operation failed after %d retries: %w", maxRetries, lastErr)
}

func getStatus() error {
	operation := func() error {
		client := &http.Client{}
		req, err := http.NewRequest("GET", "http://localhost:8080/status", nil)
		if err != nil {
			return fmt.Errorf("Error creating HTTP GET request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("HTTP GET error from SKR: %w", err)
		}

		if resp.StatusCode < 200 || resp.StatusCode > 207 {
			return fmt.Errorf("HTTP GET Status code not 2xx: %d.", resp.StatusCode)
		}

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

		log.Printf("SKR status: %s", string(bodyText))

		return nil
	}

	return WithRetry(operation)
}

func retrieveKey() (*rsa.PrivateKey, error) {
	maaEndpoint := os.Getenv("SkrClientMAAEndpoint")
	akvEndpoint := os.Getenv("SkrClientAKVEndpoint")
	skrClientKID := os.Getenv("SkrClientKID")

	var key *rsa.PrivateKey
	log.Printf("[retrieveKey] Using environment variables:\n  SkrClientMAAEndpoint=%s\n  SkrClientAKVEndpoint=%s\n  SkrClientKID=%s", maaEndpoint, akvEndpoint, skrClientKID)

	operation := func() error {
		client := &http.Client{}

		payload := fmt.Sprintf(`{"maa_endpoint": "%s", "akv_endpoint": "%s", "kid": "%s"}`, maaEndpoint, akvEndpoint, skrClientKID)

		log.Printf("[retrieveKey] Sending JSON payload to SKR:\n%s", payload)
		var data = strings.NewReader(payload)

		req, err := http.NewRequest("POST", "http://localhost:8080/key/release", data)
		if err != nil {
			return fmt.Errorf("Error creating HTTP POST request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("HTTP POST error from SKR: %w", err)
		}

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

		if resp.StatusCode < 200 || resp.StatusCode > 207 {
			log.Printf("[retrieveKey] SKR returned %d. Body:\n%s",
				resp.StatusCode, string(bodyText))
			return fmt.Errorf("Unable to retrieve key from SKR. HTTP POST Response Code %d.", resp.StatusCode)
		}

		log.Printf("[retrieveKey] SKR returned status: %d, body: %s", resp.StatusCode, string(bodyText))

		if err := json.Unmarshal(bodyText, &datakey); err != nil {
			return fmt.Errorf("Unmarshalling key error: %w bodyText: %s", err, string(bodyText))
		}

		k, err := RSAPrivateKeyFromJWK([]byte(datakey.Key))
		if err != nil {
			return fmt.Errorf("Constructing private rsa key from jwk key error: %w", err)
		}

		key = k
		keyEnabled = true
		return nil
	}

	if err := WithRetry(operation); err != nil {
		return nil, fmt.Errorf("Error retrieving key: %w", err)
	}

	log.Printf("consumer modulus (hex head) = %x", key.N.Bytes()[:32])

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
		return nil, fmt.Errorf("error unmarshalling JWK: %w", err)
	}
	n, err := base64.RawURLEncoding.DecodeString(jwkData.N)
	if err != nil {
		return nil, fmt.Errorf("error decoding n: %w", err)
	}
	e, err := base64.RawURLEncoding.DecodeString(jwkData.E)
	if err != nil {
		return nil, fmt.Errorf("error decoding e: %w", err)
	}
	d, err := base64.RawURLEncoding.DecodeString(jwkData.D)
	if err != nil {
		return nil, fmt.Errorf("error decoding d: %w", err)
	}
	p, err := base64.RawURLEncoding.DecodeString(jwkData.P)
	if err != nil {
		return nil, fmt.Errorf("error decoding p: %w", err)
	}
	q, err := base64.RawURLEncoding.DecodeString(jwkData.Q)
	if err != nil {
		return nil, fmt.Errorf("error decoding q: %w", err)
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
