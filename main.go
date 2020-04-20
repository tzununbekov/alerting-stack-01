package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	cloudevents "github.com/cloudevents/sdk-go"
)

var (
	nameOfMonitoredParameter = "apiserver write requests"
	allowedIncreaseFactor    = 2.0
)

type promResp struct {
	Data struct {
		Result []struct {
			Metric struct{}        `json:"metric"`
			Values [][]interface{} `json:"values"`
		} `json:"result"`
		ResultType string `json:"resultType"`
	} `json:"data"`
	Status string `json:"status"`
}

func process(ctx context.Context, event cloudevents.Event) {
	fmt.Printf("☁️  cloudevents.Event\n%s", event.String())
	var data promResp
	err := event.DataAs(&data)
	if err != nil {
		log.Printf("data parse: %v\n", err)
		return
	}
	if data.Status != "success" {
		log.Printf("status: %s\n", data.Status)
		return
	}

	for _, v := range data.Data.Result {
		l := len(v.Values)
		if l < 2 {
			return
		}
		if len(v.Values[0]) != 2 || len(v.Values[l-1]) != 2 {
			return
		}
		firstVal, err := stringToFloat(v.Values[0][1])
		if err != nil {
			log.Printf("value parse error: %v\n", err)
			return
		}
		lastVal, err := stringToFloat(v.Values[l-1][1])
		if err != nil {
			log.Printf("value parse error: %v\n", err)
			return
		}
		if lastVal/firstVal > allowedIncreaseFactor {
			message := fmt.Sprintf("%q values exceeded allowed factor of %f: %f -> %f\n",
				nameOfMonitoredParameter, allowedIncreaseFactor, firstVal, lastVal)
			err := send(message)
			if err != nil {
				log.Printf("error sending event to slack: %v\n", err)
			}
			return
		}
	}
}

func stringToFloat(val interface{}) (float64, error) {
	firstValStr, ok := val.(string)
	if !ok {
		return 0, fmt.Errorf("value is not string")
	}
	return strconv.ParseFloat(firstValStr, 64)
}

func send(data string) error {
	slackURI := os.Getenv("TARGET")
	if slackURI == "" {
		return fmt.Errorf("\"TARGET\" env not set")
	}
	var jsonStr = []byte(fmt.Sprintf("{\"text\":%q}", data))
	req, err := http.NewRequest("POST", slackURI, bytes.NewBuffer(jsonStr))
	if err != nil {
		return fmt.Errorf("request creation error: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send request error: %v", err)
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
	return nil
}

func main() {
	c, err := cloudevents.NewDefaultClient()
	if err != nil {
		log.Fatal("Failed to create client, ", err)
	}
	log.Fatal(c.StartReceiver(context.Background(), process))
}
