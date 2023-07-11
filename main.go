// this starts the NATS server
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
	"github.com/pkg/errors"
)

// DatafileURLTemplate is used to construct the endpoint for retrieving regular datafile from the CDN
const DatafileURLTemplate = "https://cdn.optimizely.com/datafiles/%s.json"

var Datafile []byte

func syncDatafile(ctx context.Context, dataCh chan<- []byte) error {
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

			if !reflect.DeepEqual(Datafile, datafile) {
				Datafile = datafile
				dataCh <- datafile
			}
			time.Sleep(1 * time.Minute)
		}
	}
}

func publishDatafile(js nats.JetStreamContext, dataCh <-chan []byte) {
	for {
		select {
		case datafile, ok := <-dataCh:
			if !ok {
				return
			}
			natsMsg := &nats.Msg{
				Subject: "DATAFILE.PUBLISH",
				Data:    datafile,
			}
			_, err := js.PublishMsg(natsMsg)
			if err != nil {
				fmt.Errorf("failed to publish datafile, err: %s", err.Error())
			}
			fmt.Println("successfully published datafile at: ", time.Now().String())
		}
	}
}

func getDatafile(js nats.JetStreamContext) error {
	strInfo, err := js.AddStream(&nats.StreamConfig{
		Name:        "DATAFILE",
		Description: "Take order for pizza from customer",
		Subjects:    []string{"DATAFILE.*"},
	})
	if err != nil {
		return errors.Wrap(err, "failed to add stream")
	}

	conInfo, err := js.AddConsumer(strInfo.Config.Name, &nats.ConsumerConfig{
		Durable:       "CONSUMER",
		AckPolicy:     nats.AckExplicitPolicy,
		FilterSubject: "DATAFILE.PUBLISH",
	})
	if err != nil {
		return err
	}

	sub, err := js.PullSubscribe(conInfo.Config.FilterSubject, conInfo.Name, nats.BindStream(conInfo.Stream))
	if err != nil {
		return err
	}
	defer func() {
		err := sub.Unsubscribe()
		if err != nil {
			panic(err)
		}
	}()

	for {
		msgs, err := sub.Fetch(1)
		if err == nats.ErrTimeout {
			continue
		}
		if err != nil {
			fmt.Errorf("failed to fetch message: %s", err.Error())
			continue
		}
		if len(msgs) == 0 {
			continue
		}
		fmt.Println("fetched updated datafile: ")
		err = msgs[0].Ack()
		if err != nil {
			fmt.Errorf("failed to actknowledge message: %s", err.Error())
			continue
		}
		fmt.Println("Received datafile: ", string(msgs[0].Data))
		// update datafile
		Datafile = msgs[0].Data
		fmt.Println("updated datafile at: ", time.Now().String())
	}
}

func main() {
	// please make sure nats server is already running
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dataCh := make(chan []byte)
	defer close(dataCh)

	go func() {
		if err := syncDatafile(ctx, dataCh); err != nil {
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

	go func() {
		if err := getDatafile(js); err != nil {
			panic(err)
		}
	}()
	go publishDatafile(js, dataCh)

	// blocking main go routine
	select {}
}
