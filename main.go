package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/nu7hatch/gouuid"
	"gopkg.in/yaml.v2"
)

// requestMessage represents an http event to be repeated to eventTargets
// This is, in essence, an http.Request, but those aren't very clean to pass around
type requestMessage struct {
	UUID    string
	URL     string
	Method  string
	Source  string
	Headers http.Header
	Body    []byte
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, "{ \"error\": \"No such queue\" }\n")
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	mapB, _ := json.Marshal(counters) // meh @ error handling here :)
	w.Write(mapB)
}

func makeIncomingHandler(queueItems QueueItems) http.HandlerFunc {
	// I'm doing it like this so my IDE highlighting doesn't get stupid
	h := func(w http.ResponseWriter, r *http.Request) {
		// get a UUID for this transaction
		u5, err := uuid.NewV4()
		if err != nil {
			log.WithFields(log.Fields{
				"error": err.Error(),
			}).Error("failed-to-generate-uuid")
			return
		}

		body, _ := ioutil.ReadAll(r.Body)

		requestObj := requestMessage{
			UUID:    u5.String(),
			URL:     r.URL.Path[1:], // trim the leading /
			Method:  r.Method,
			Source:  r.RemoteAddr,
			Headers: r.Header,
			Body:    body,
		}

		queue := requestObj.URL
		var worked int
		worked = 1

		for name, eventTarget := range queueItems {
			qu := Queue{queue, name, eventTarget.URL}
			addchan <- qu
			// this select/case/default is a non-blocking chan push
			select {
			case requestBuffers[qu] <- requestObj:
			default:
				// metricize that we're dropping messages
				dellchan <- qu
				// kill off the oldest, not-in-flight message
				worked = 2
				// todo: it could possibly make sense to kill the inflight message, but...have to think on that more
				<-requestBuffers[qu]
				// we attempt to send the current message one last time, but this is still not guaranteed to work
				select {
				case requestBuffers[qu] <- requestObj:
				default:
					worked = 3
					// well, we tried our damndest, log it and move on
					log.WithFields(log.Fields{
						"id":    u5,
						"queue": queue,
						"url":   eventTarget.URL,
					}).Info("queue-full-message-lost")
				}
			}
		}

		// to lazy to do a real json.Marshal, etc
		switch worked {
		case 1:
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "{ \"id\":\"%s\" }\n", u5)
		case 2:
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "{ \"id\":\"%s\", \"error\": \"Queue full, old message dropped\" }\n", u5)
		case 3:
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "{ \"id\":\"%s\", \"error\": \"Queue full, old message dropped, new message un-queued\" }\n", u5)
		}
	}
	return h
}

func sendEvent(client *http.Client, qu Queue, req requestMessage, workerId uint) {

	log.WithFields(log.Fields{
		"queue": qu,
		"req":   req,
		"wid":   workerId,
	}).Debug("sending")

	start := time.Now()
	var sent bool
	sent = false
	attempts := 0
	sleepDuration := time.Millisecond * 100
	for {
		attempts++
		httpReq, _ := http.NewRequest(req.Method, qu.OutboundURL, bytes.NewBuffer(req.Body))
		for headerName, values := range req.Headers {
			for _, value := range values {
				httpReq.Header.Add(headerName, value)
			}
		}
		httpReq.Header.Set("X-Capacitor-Id", req.UUID)
		resp, err := client.Do(httpReq)

		if err == nil {
			// get rid of the response, we don't care
			// but we do need to clean it out, so the client can reuse the same connection
			io.Copy(ioutil.Discard, resp.Body)
			resp.Body.Close()
		}

		if err == nil && resp.StatusCode == 200 {
			sent = true
			break
		}

		// max duration, ever
		// todo: make this configurable
		if time.Since(start) > time.Second*60 {
			break
		}

		// oops, didn't work; have a pause and try again in a bit
		time.Sleep(sleepDuration)

		// slowly ramp up our sleep interval, shall we? But cap it too
		if sleepDuration < time.Duration(time.Second*15) {
			sleepDuration = time.Duration(float64(sleepDuration) * 1.5)
		} else {
			sleepDuration = time.Duration(time.Second * 15)
		}
	}
	elapsed := time.Since(start)

	if sent {
		deltchan <- qu
	} else {
		delfchan <- qu
	}

	log.WithFields(log.Fields{
		"id":       req.UUID,
		"queue":    qu.InboundName,
		"outbound": qu.OutboundName,
		"wid":      workerId,
		"attempts": attempts,
		"sent":     sent,
		"duration": elapsed.Seconds() * 1e3, /* ms hack */
	}).Info("relay-end")
}

// queueUrl is a structure for uniqu'ifying tracking maps
type Queue struct {
	InboundName  string
	OutboundName string
	OutboundURL  string
}

type counterVals struct {
	Current uint64 `json:"current"`
	Total   uint64 `json:"total"`
	Success uint64 `json:"success"`
	Failure uint64 `json:"failure"`
	Lost    uint64 `json:"lost"`
}

func (u StatsMap) MarshalJSON() ([]byte, error) {
	fmt.Println("marshalling StatsMap")
	var v = make(map[string]counterVals)
	for i, e := range u {
		v[i.InboundName+":"+i.OutboundName] = e
	}
	return json.Marshal(v)
}

type StatsMap map[Queue]counterVals

var counters = make(StatsMap)
var addchan = make(chan Queue, 100)
var deltchan = make(chan Queue, 100)
var delfchan = make(chan Queue, 100)
var dellchan = make(chan Queue, 100)

func StartWorker(incoming chan requestMessage, queue Queue, id uint) {
	client := &http.Client{
		// todo: reasonable default?
		Timeout: 10 * time.Second,
		// no cookies, please
		Jar: nil,
	}
	for {
		work := <-incoming
		sendEvent(client, queue, work, id)
	}
}

var requestBuffers = make(map[Queue]chan requestMessage)

func main() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339Nano,
	})

	configFile := flag.String("config", "config.yml", "Path to the config file")
	debug := flag.Bool("debug", false, "Enable debug logging")
	port := flag.Uint("port", uint(8000), "Listen port")
	flag.Parse()

	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	if _, err := os.Stat(*configFile); os.IsNotExist(err) {
		log.Errorf("Config must be specified and must exist: %s", *configFile)
		os.Exit(1)
	}

	yamlFile, err := ioutil.ReadFile(*configFile)
	if err != nil {
		panic(err)
	}

	var config Config
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		panic(err)
	}

	// TODO: add config file validation, such as:
	//if target.QueueLength <= 0 {
	//	panic("Buffer length must be > 0")
	//}
	// TODO: `/api` is reserved as a root for the app

	// Setup workers and initialize counters to zero
	for inbound, targets := range config {
		for targetid, target := range targets {
			qu := Queue{inbound, targetid, target.URL}
			counters[qu] = counterVals{0, 0, 0, 0, 0}
			requestBuffers[qu] = make(chan requestMessage, target.QueueLength)
			for i := uint(1); i <= target.MaxParallel; i++ {
				go StartWorker(requestBuffers[qu], qu, i)
			}
		}
	}

	// goroutine to keep the counters up-to-date
	go func() {
		for {
			// watch each channel as items rolls in and modify the counters as needed
			select {
			// you can't do counters[control].Current++ in go, so this mess is what results
			case control := <-addchan:
				tmp := counters[control]
				tmp.Current++
				tmp.Total++
				counters[control] = tmp
			case control := <-deltchan:
				tmp := counters[control]
				tmp.Current--
				tmp.Success++
				counters[control] = tmp
			case control := <-delfchan:
				tmp := counters[control]
				tmp.Current--
				tmp.Failure++
				counters[control] = tmp
			case control := <-dellchan:
				tmp := counters[control]
				tmp.Current--
				tmp.Lost++
				counters[control] = tmp
			}
		}
	}()

	// A dumb goroutine to watch memory usage and counter metrics
	go stats()

	// Oh, hey, there's the webserver!
	portStr := strconv.Itoa(int(*port))
	log.Info("Starting server on port "+portStr)
	for inbound, queueItems := range config {
		log.Info("Registering queue @ /" + inbound)
		http.HandleFunc("/"+inbound, makeIncomingHandler(queueItems))
	}
	http.HandleFunc("/", defaultHandler)
	http.HandleFunc("/api/metrics", metricsHandler)
	log.Fatal("ListenAndServe: ", http.ListenAndServe(":"+portStr, nil))
}
