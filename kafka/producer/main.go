package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/Shopify/sarama"

	crand "crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
)

var totalMsg int64
var totalBurst int
var totalSendDur int
var totalRestDur int
var topic = getenv("TOPIC", "my-topic")
var brokers = getenv("BROKERS", "my-cluster-kafka-bootstrap:9092")
var userconfiguredmsg = getenv("MSG", "Azure confidential computing. Increase data privacy by protecting data in use")
var ratedeviation, _ = strconv.ParseInt(getenv("RATEDEVIATION", "2"), 10, 64)
var rate, _ = strconv.ParseInt(getenv("RATE", "10"), 10, 64)
var senddurationdeviation, _ = strconv.ParseInt(getenv("SENDDURATIONDEVIATION", "2"), 10, 64)
var sendduration, _ = strconv.ParseInt(getenv("SENDDURATION", "5"), 10, 64)
var restdurationdeviation, _ = strconv.ParseInt(getenv("RESTDURATIONDEVIATION", "2"), 10, 64)
var restduration, _ = strconv.ParseInt(getenv("RESTDURATION", "10"), 10, 64)
var stats struct {
	i  int
	j  int
	Mu sync.Mutex
}

func main() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		time.Sleep(3 * time.Second)
		log.Println("Keyboard interrupted. Exit Program")
		log.Printf("total produced messages: %d", totalMsg)
		log.Printf("average burst rate is: %d", int(totalMsg)/totalBurst)
		log.Printf("average rest duration is: %d", totalRestDur/totalBurst)
		log.Printf("average send duration is: %d", totalSendDur/totalBurst)
		os.Exit(1)
	}()

	go loopPrint()

	config, err := inClusterKafkaConfig()
	if err != nil {
		log.Printf("unable to construct kafka config: %s", err.Error())
	}

	producer, err := sarama.NewSyncProducer([]string{brokers}, config)
	if err != nil {
		log.Printf("unable to start kafka sync config: %s", err.Error())
	}
	produce(int(rate), int(ratedeviation), int(restduration), int(restdurationdeviation), int(sendduration), int(senddurationdeviation), &producer)
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func produce(targetRate int, rateSpan int, targetRestDuration int, targetRestSpan int, targetSendDuration int, targetSendSpan int, p *sarama.SyncProducer) {
	rest := false
	for {
		if rest {
			ima := time.Now()
			if targetRestDuration != 0 {
				restDuration := generateRandomNum(targetRestSpan, targetRestDuration)
				time.Sleep(time.Duration(restDuration) * time.Second)
				log.Println("slept ", time.Since(ima), " seconds.")
				totalRestDur += restDuration
			}
			rest = false
		} else {
			produceMessage(targetRate, rateSpan, targetSendDuration, targetSendSpan, &rest, p)
		}
	}
}

func produceMessage(targetRate int, rateSpan int, targetSendDuration int, targetSendSpan int, rest *bool, p *sarama.SyncProducer) {
	totalBurst += 1
	rate, sendDuration := generateRandomNum(rateSpan, targetRate), generateRandomNum(targetSendSpan, targetSendDuration)
	totalSendDur += sendDuration
	start := time.Now()
	ticker := time.NewTicker(time.Duration(sendDuration) * time.Second)
	log.Println("will send for", sendDuration, "seconds.")
	stopped := false
	count := 0

	go func() {
		for {
			select {
			case <-ticker.C:
				stopped = true
				ticker.Stop()
				break
			}
			break
		}
	}()
	for {
		if stopped {
			log.Println("sent", count, "within", time.Since(start))
			*rest = true
			break
		}
		if count >= rate {
			continue
		}

		count += 1
		value := fmt.Sprintf("Msg %d: %s", count, userconfiguredmsg)

		// Encrypt the message here using the public key. Replace 'path_to_public_key.pem' with the actual path to your public key.
		encryptedValue, err := encryptMessage(value, "path_to_public_key.pem")
		if err != nil {
			log.Printf("Error encrypting message: %s", err.Error())
			continue
		}

		message := &sarama.ProducerMessage{
			Topic: topic,
			Value: sarama.StringEncoder(encryptedValue),
		}
		_, _, err = (*p).SendMessage(message)
		atomic.AddInt64(&totalMsg, 1)
		if err != nil {
			log.Println(err.Error())
			break
		}
		log.Println("Produced message to partition")
		count += 1

	}
}

func encryptMessage(plaintext string, publicKeyPath string) (string, error) {
	var pubpem []byte
	var err error
	if pkey := os.Getenv("PUBKEY"); len(pkey) > 0 {
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

func generateRandomNum(span int, targetRate int) int {
	if span == 0 {
		return targetRate
	}
	rand.Seed(time.Now().UnixNano())
	min := targetRate - span
	return rand.Intn((targetRate+span)-min) + min + 1
}

func loopPrint() {
	for {
		time.Sleep(5 * time.Second)
		log.Printf("total Msg is: %d", totalMsg)
		log.Printf("average burst rate is: %d", int(totalMsg)/totalBurst)
		log.Printf("average rest duration is: %d", totalRestDur/totalBurst)
		log.Printf("average send duration is: %d", totalSendDur/totalBurst)
	}
}

func inClusterKafkaConfig() (kafkaConfig *sarama.Config, err error) {
	kafkaConfig = sarama.NewConfig()
	kafkaConfig.ClientID = "kafka-on-kata-cc-mariner"
	kafkaConfig.Producer.RequiredAcks = sarama.WaitForLocal
	kafkaConfig.Producer.Return.Successes = true

	return kafkaConfig, nil
}
