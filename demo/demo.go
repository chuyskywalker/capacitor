package main

/**
 * A simple demo app that will listen on argv[1] and report on recieved events
 *
 *     go run demo.go 8001
 *
 * You may also pass a "duration" value to force the application to take a
 * sleep pause to emulate "real" load
 *
 *     go run demo.go 8001 150ms
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
	var d time.Duration
	var err error
	if len(os.Args) >= 3 {
		d, err = time.ParseDuration(os.Args[2])
		if err != nil {
			log.Error(err.Error())
		}
	}
	log.Info("Server starting @ port "+os.Args[1])
	http.HandleFunc("/", func (w http.ResponseWriter, r *http.Request) {
		time.Sleep(d)
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
