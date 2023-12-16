package main

import (
	"io"
	"log"
	"os"

	"github.com/nats-io/stan.go"
	"github.com/renegatemaster/wb-l0/initializers"
)

func init() {
	initializers.LoadEnvVariables()
}

func main() {

	// Config
	var (
		clusterID string = os.Getenv("CLUSTER_ID")
		clientID  string = os.Getenv("CLIENT_ID_PUB")
		path      string = os.Getenv("PATH_TO_FILES")
		channel   string = os.Getenv("CHANNEL")
		URL       string = stan.DefaultNatsURL
	)

	// Uncomment line below to test trash publishing
	// path = os.Getenv("PATH_TO_TRASH")

	// Connection to NATS
	sc, err := stan.Connect(clusterID, clientID, stan.NatsURL(URL))
	if err != nil {
		log.Fatalf("Can't connect: %v.\nMake sure a NATS Streaming Server is running at: %s", err.Error(), URL)
	}
	log.Printf("Connected to NATS Streaming with clusterID '%s', clientID '%s' and URL '%s'", clusterID, clientID, URL)

	defer func(sc stan.Conn) {
		err := sc.Close()
		if err != nil {
			log.Println(err.Error())
			return
		}
	}(sc)

	// Reading files to publish
	files, err := os.ReadDir(path)
	if err != nil {
		log.Fatal(err.Error())
	}

	// Publishing files to channel
	for _, file := range files {
		msg, err := readData(path + "/" + file.Name())
		if err != nil {
			log.Println(err.Error())
			return
		}
		err = sc.Publish(channel, msg)
		if err != nil {
			log.Println(err.Error())
		} else {
			log.Printf("Published from: %s", file.Name())
		}
	}
}

func readData(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	buf, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return buf, nil
}
