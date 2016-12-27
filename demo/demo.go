package main

/**
 * A simple demo app that will listen on a given port for events
 *
 *     go run demo.go 8001
 */

import (
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

func getEvent(w http.ResponseWriter, r *http.Request) {
	body, _ := ioutil.ReadAll(r.Body)
	log.WithFields(log.Fields{
		"len":     r.ContentLength,
		"content": string(body),
		"id":      r.Header.Get("X-Wsq-Id"),
	}).Info("got event")
	w.Write([]byte("Thanks!"))
}

func main() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339Nano,
	})
	log.Info("starting server")
	http.HandleFunc("/", getEvent)
	err := http.ListenAndServe(":"+os.Args[1], nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
