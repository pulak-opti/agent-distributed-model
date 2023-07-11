package main

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

func main() {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		panic(err)
	}
	js, err := nc.JetStream()
	if err != nil {
		panic(err)
	}

	for {
		if err := publishDatafile(js); err != nil {
			panic(err)
		}
		time.Sleep(5 * time.Second)
	}
}

func publishDatafile(js nats.JetStreamContext) error {
	natsMsg := &nats.Msg{
		Subject: "DATAFILE.PUBLISH",
		Data:    []byte(fmt.Sprintf("hello nats: time: %s", time.Now().String())),
	}
	_, err := js.PublishMsg(natsMsg)
	if err != nil {
		return err
	}
	fmt.Println("successfully published msg")
	return nil
}
