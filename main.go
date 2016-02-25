package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/bitly/go-nsq"
	"github.com/gorilla/mux"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func closeConn() {
	if reader != nil {
		reader.Close()
	}
}

type Data struct {
	CubeaconID   string `json:"cubeaconId"`
	Flag         string `json:"flag"`
	Value        int    `json:"value"`
	Sensor       string `json:"sensor"`
	Interactions int    `json:"interactions"`
}

type BeaconDataPostResponse struct {
	Result  int    `json:"result"`
	Message string `json:"message"`
}

var reader io.ReadCloser

func main() {

	// NSQ Publisher
	publisherStopChan := make(chan struct{}, 1)
	stop := false
	signalChan := make(chan os.Signal, 1)

	go func() {
		<-signalChan
		stop = true
		log.Println("Stopping...")
		closeConn()
	}()

	// set signal
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// chan for beacons
	beacons := make(chan string)

	// anonymous method for producer
	go func() {
		pub, _ := nsq.NewProducer("172.17.8.101:4150", nsq.NewConfig())
		for beacon := range beacons {
			// publish beacon
			pub.Publish("beacons", []byte(beacon))
		}

		log.Println("Publisher: Stopping")

		pub.Stop()

		log.Println("Publisher: Stopped")

		// fill channel with empty struct
		publisherStopChan <- struct{}{}
	}()

	// HTTP Method
	mx := mux.NewRouter()

	result := BeaconDataPostResponse{}
	mx.HandleFunc("/beacons/interactions", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Access-Control-Allow-Origin", "*")
		reader = r.Body
		buf := new(bytes.Buffer)
		buf.ReadFrom(reader) // Does a complete copy of the bytes in the buffer.

		data := Data{}
		if jsonString := buf.String(); jsonString != "<nil>" {

			if jsonString == "" {
				result.Result = 0
				result.Message = "Failed to send. No JSON body found"
				fmt.Println("No JSON body found")
			}

			json.Unmarshal([]byte(jsonString), &data)

			if data.CubeaconID == "" {
				result.Result = 0
				result.Message = "Failed to send. CubeaconID is empty"
				fmt.Println("CubeaconID is empty")
			} else {

				log.Println("beacons:", jsonString)
				beacons <- jsonString

				result.Result = 1
				result.Message = "Successfully send."
			}

		} else {
			result.Result = 0
			result.Message = "failed to send"
			log.Println("Empty JSON String")
		}

		b, err := json.Marshal(result)
		if err != nil {
			log.Println("error:", err)
		}

		w.Write([]byte(b))

	}).Methods("POST")

	mx.HandleFunc("/beacons/interactions/{cubeaconid}", GetAnalytics).Methods("GET")
	mx.HandleFunc("/beacon/analytics", GetBeaconAnalytics).Methods("GET")
	mx.HandleFunc("/beacon/analytics/flag", GetFlagAnalytics).Methods("GET")

	fmt.Println("Apps is running in port :9090")
	http.ListenAndServe(":9090", mx)

	go func() {
		for {
			time.Sleep(1 * time.Minute)
			closeConn()
			if stop {
				break
				os.Exit(1)
			}
		}
	}()

	close(beacons)
	<-beacons
	<-publisherStopChan
}
