package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"time"

	"github.com/nats-io/nats.go"
)

// DatafileURLTemplate is used to construct the endpoint for retrieving regular datafile from the CDN
const DatafileURLTemplate = "https://cdn.optimizely.com/datafiles/%s.json"

var Datafile []byte

func syncDatafile(ctx context.Context, ch chan bool) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			url := fmt.Sprintf(DatafileURLTemplate, os.Getenv("SDK_KEY"))

			resp, err := http.Get(url)
			if err != nil {
				return err
			}

			datafile, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			if resp.StatusCode != http.StatusOK {
				fmt.Println("failed to fetch datafile, statue code: ", resp.StatusCode)
				time.Sleep(5 * time.Minute)
				continue
			}
			if !reflect.DeepEqual(datafile, Datafile) {
				ch <- true
				Datafile = datafile
			}
			time.Sleep(5 * time.Minute)
		}
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan bool)
	defer close(ch)

	go func() {
		err := syncDatafile(ctx, ch)
		if err != nil {
			panic(err)
		}
	}()

	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		panic(err)
	}
	js, err := nc.JetStream()
	if err != nil {
		panic(err)
	}

	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
			if err := publishDatafile(js); err != nil {
				fmt.Println("failed to publish datafile: ", err)
			}
		}
	}
}

func publishDatafile(js nats.JetStreamContext) error {
	natsMsg := &nats.Msg{
		Subject: "DATAFILE.PUBLISH",
		Data:    Datafile,
	}
	_, err := js.PublishMsg(natsMsg)
	if err != nil {
		return err
	}
	fmt.Println("successfully published msg")
	return nil
}
