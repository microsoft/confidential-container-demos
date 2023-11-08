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
	"html/template"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Shopify/sarama"
)

var topic = getenv("TOPIC", "my-topic")
var brokers = getenv("BROKERS", "my-cluster-kafka-bootstrap:9092")
var consumergroup = getenv("CONSUMERGROUP", "strimzikafkaconsumergroupid")

var keyEnabled bool

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func main() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	cg, _ := inClusterKafkaConfig()

	t, err := template.ParseFiles("/webtemplates/index.html")
	if err != nil {
		log.Printf("Error parsing templates: %s", err.Error())
		os.Exit(1)
	}

	consumerGroup, err := sarama.NewConsumerGroup([]string{brokers}, consumergroup, cg)
	if err != nil {
		log.Printf("Error creating the Sarama consumer: %v", err)
		os.Exit(1)
	}

	cgh := &consumerGroupHandler{
		ready:    make(chan struct{}),
		end:      make(chan struct{}),
		done:     make(chan struct{}),
		messages: make(chan string),
	}

	ctx := context.Background()
	go func() {
		for {
			// this method calls the methods handler on each stage: setup, consume and cleanup
			consumerGroup.Consume(ctx, []string{topic}, cgh)
		}

	}()

	getRoot := func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			Encrypted bool
			Message   string
		}{
			Encrypted: keyEnabled,
		}
		data.Message = <-cgh.messages
		fmt.Printf("got / request")
		err := t.Execute(w, data)
		if err != nil {
			log.Printf("Unable to serve webpage: %s", err.Error())
		}
	}

	<-cgh.ready // Await till the consumer has been set up
	log.Println("Sarama consumer up and running!...")

	http.HandleFunc("/", getRoot)
	http.Handle("/web/", http.StripPrefix("/web", http.FileServer(http.Dir("/web"))))
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "/web/favicon.ico")
	})

	go func() {
		err = http.ListenAndServe(":3333", nil)
		if errors.Is(err, http.ErrServerClosed) {
			fmt.Printf("server closed\n")
		} else if err != nil {
			fmt.Printf("error starting server: %s\n", err)
			os.Exit(1)
		}
	}()

	// waiting for the end of all messages received or an OS signal
	select {
	case <-cgh.end:
		log.Printf("Finished to receive %d messages", 100)
	case sig := <-signals:
		log.Printf("Got signal: %v", sig)
		close(cgh.done)
	}

	err = consumerGroup.Close()
	if err != nil {
		log.Printf("Error closing the Sarama consumer: %v", err)
		os.Exit(1)
	}
	log.Printf("Consumer closed")
}

// struct defining the handler for the consuming Sarama method
type consumerGroupHandler struct {
	ready    chan struct{}
	end      chan struct{}
	done     chan struct{}
	messages chan string
}

func (cgh *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	close(cgh.ready)
	log.Printf("Consumer group handler setup")
	return nil
}

func (cgh *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	log.Printf("Consumer group handler cleanup")
	return nil
}

var datakey struct {
	Key string `json:"key"`
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
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
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

func (cgh *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	key, err := retrieveKey()
	if err != nil {
		log.Printf("not able to retrieve key: %s", err.Error())
	}

	messageChan := claim.Messages()
	for {
		select {
		case message, ok := <-messageChan:
			if !ok {
				log.Printf("message channel was closed")
				return nil
			}
			messageContents := ""
			if key != nil {
				annotationBytes, err := base64.StdEncoding.DecodeString(string(message.Value))
				if err != nil {
					log.Printf("error decoding message value %q", err.Error())
				}

				plaintext, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, key, annotationBytes, nil)
				if err != nil {
					log.Printf("error decrypting message %q", err.Error())
				}

				messageContents = string(plaintext)
			} else {
				messageContents = string(message.Value)
			}

			select {
			case cgh.messages <- messageContents:
			default:
			}
			log.Printf("Message received: value=%s, partition=%d, offset=%d", messageContents, message.Partition, message.Offset)

			session.MarkMessage(message, "")
		// Should return when `session.Context()` is done.
		// If not, will raise `ErrRebalanceInProgress` or `read tcp <ip>:<port>: i/o timeout` when kafka rebalance. see:
		// https://github.com/IBM/sarama/issues/1192
		case <-cgh.done:
			return nil
		}
	}
}

func inClusterKafkaConfig() (kafkaConfig *sarama.Config, err error) {
	kafkaConfig = sarama.NewConfig()
	kafkaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	kafkaConfig.Version = sarama.V0_10_2_1
	return kafkaConfig, nil
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
