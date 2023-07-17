package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"

)

// DatafileURLTemplate is used to construct the endpoint for retrieving regular datafile from the CDN
const DatafileURLTemplate = "https://cdn.optimizely.com/datafiles/%s.json"

var Datafile []byte

func generateRandomString(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	randomBytes := make([]byte, length)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", nil
	}

	for i := 0; i < length; i++ {
		randomBytes[i] = charset[randomBytes[i]%byte(len(charset))]
	}

	return string(randomBytes), nil
}

func webhook(w http.ResponseWriter, r *http.Request) {
	df, err := downloadDatafile()
	if err != nil {
		w.Write([]byte(err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	nc, err := nats.Connect(os.Getenv("NATS_URL"))
	if err != nil {
		w.Write([]byte(err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	js, err := nc.JetStream()
	if err != nil {
		w.Write([]byte(err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := publishDatafile(js, df); err != nil {
		w.Write([]byte(err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write([]byte("received updated datafile notification"))
}

func downloadDatafile() ([]byte, error) {
	url := fmt.Sprintf(DatafileURLTemplate, os.Getenv("SDK_KEY"))

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	datafile, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch datafile, statue code: %d", resp.StatusCode)
	}
	return datafile, nil
}

func publishDatafile(js nats.JetStreamContext, datafile []byte) error {
	natsMsg := &nats.Msg{
		Subject: "DATAFILE.PUBLISH",
		Data:    datafile,
	}
	_, err := js.PublishMsg(natsMsg)
	if err != nil {
		return fmt.Errorf("failed to publish datafile, err: %s", err.Error())
	}
	fmt.Println("successfully published datafile at: ", time.Now().String())
	return nil
}

func listenForDatafile(js nats.JetStreamContext) error {
	strInfo, err := js.AddStream(&nats.StreamConfig{
		Name:        "DATAFILE",
		Description: "Take order for pizza from customer",
		Subjects:    []string{"DATAFILE.*"},
	})
	if err != nil {
		return errors.Wrap(err, "failed to add stream")
	}

	randSuffix, err := generateRandomString(5)
	if err != nil {
		return errors.Wrap(err, "failed to create consumer durable name")
	}
	conInfo, err := js.AddConsumer(strInfo.Config.Name, &nats.ConsumerConfig{
		Durable:       fmt.Sprintf("CONSUMER-%s", randSuffix),
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

	fmt.Println("datafile listener is running")

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
	nc, err := nats.Connect(os.Getenv("NATS_URL"))
	if err != nil {
		panic(err)
	}
	js, err := nc.JetStream()
	if err != nil {
		panic(err)
	}

	go func() {
		if err := listenForDatafile(js); err != nil {
			panic(err)
		}
	}()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/webhook/optimizely", webhook)
	fmt.Println("Optimizely Agent server is running")
	http.ListenAndServe(":8080", r)
}
