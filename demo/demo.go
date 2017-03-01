package main

/**
 * A simple demo app that will listen on argv[1] and report on recieved events
 *
 *     go run demo.go 8001
 */

import (
	"io/ioutil"
	"net/http"
	"time"
	"os"

	log "github.com/Sirupsen/logrus"
)

func main() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339Nano,
	})
	log.Info("Server starting @ port "+os.Args[1])
	http.HandleFunc("/", func (w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
		log.WithFields(log.Fields{
			"len":     r.ContentLength,
			"content": string(body),
			"id":      r.Header.Get("X-Capacitor-Id"),
		}).Info("event")
		w.Write([]byte("Thanks!"))
	})
	http.ListenAndServe(":"+os.Args[1], nil)
}
