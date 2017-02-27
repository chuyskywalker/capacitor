package main

/**
 * A simple demo app that will listen on 8001/8002 and report on recieved events
 *
 *     go run demo.go
 */

import (
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"time"
)

func getEvent(w http.ResponseWriter, r *http.Request, port int) {
	body, _ := ioutil.ReadAll(r.Body)
	log.WithFields(log.Fields{
		"len":     r.ContentLength,
		"content": string(body),
		"id":      r.Header.Get("X-Capacitor-Id"),
		"port":    port,
	}).Info("got-event")
	w.Write([]byte("Thanks!"))
}

type srv8001 struct {}
func (m *srv8001) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	getEvent(w, r, 8001)
}

type srv8002 struct {}
func (m *srv8002) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	getEvent(w, r, 8002)
}

func main() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339Nano,
	})
	log.Info("starting server")
	go func() {
		http.ListenAndServe(":8001", &srv8001{})
	}()
	http.ListenAndServe(":8002", &srv8002{})
}
