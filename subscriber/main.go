// this will act like an agent instance
package main

import (
	"fmt"

	"github.com/nats-io/nats.go"
)

func getDatafile(js nats.JetStreamContext) error {
	strInfo, err := js.AddStream(&nats.StreamConfig{
		Name:        "DATAFILE",
		Description: "Take order for pizza from customer",
		Subjects:    []string{"DATAFILE.*"},
	})
	if err != nil {
		return err
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
			return err
		}
		if len(msgs) == 0 {
			continue
		}
		err = msgs[0].Ack()
		if err != nil {
			return err
		}
		fmt.Println("Received datafile: ", string(msgs[0].Data))
	}
}

func main() {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		panic(err)
	}
	js, err := nc.JetStream()
	if err != nil {
		panic(err)
	}
	getDatafile(js)
}
