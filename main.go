package main

import (
	"bytes"
	"fmt"
	"github.com/nu7hatch/gouuid"
	"io/ioutil"
	//"log"
    log "github.com/Sirupsen/logrus"
	"net/http"
	"runtime"
	"time"
	"flag"
	"io"
)

type TargetList map[string][]EventTarget

// why not just []string of urls? In case we need meta data for these later on.
type EventTarget struct {
	Url string
}

// Our object used to repeat events.
// todo: should I just pass the http.Request? Is that thread safe?
// 		 I feel like the body being an ioreader is kind of a problem -- so most likely not
type RequestMessage struct {
	UUID    string
	URL     string
	Method  string
	Source  string
	Headers http.Header
	Body    []byte
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	//w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("No such event endpoint"))
}

func handleIncomingEvent(w http.ResponseWriter, r *http.Request) {

	// get a UUID for this transaction
	u5, err := uuid.NewV4()
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	body, _ := ioutil.ReadAll(r.Body)

	requestObj := RequestMessage{
		UUID:    u5.String(),
		URL:     r.RequestURI,
		Method:  r.Method,
		Source:  r.RemoteAddr,
		Headers: r.Header,
		Body:    body,
	}

	queue := requestObj.URL[1:]

	fmt.Fprintf(w, "{ \"id\":\"%s\" }\n", u5) // to lazy to do a real json.Marshal, etc

	for _, eventTarget := range targets[queue] {
		qu := QueueUrl{queue, eventTarget.Url}
		addchan <- qu
		// this select/case/default is a non-blocking chan push
		// todo: maybe circular chans here? IE: throw away oldest items when the chan is full.
		select {
		case sendPool[qu].RequestChan <- requestObj:
		default:
			log.WithFields(log.Fields{
				"id": u5,
				"queue": queue,
			}).Info("queue-full")
		}
	}
}

func sendEvent(client *http.Client, qu QueueUrl, req RequestMessage) {
	start := time.Now()
	var sent bool
	sent = false
	attempts := 0
	sleepDuration := time.Millisecond * 100
	for {
		attempts++
		httpReq, _ := http.NewRequest(req.Method, qu.Url, bytes.NewBuffer(req.Body))
		for headerName, values := range req.Headers {
			for _, value := range values {
				httpReq.Header.Add(headerName, value)
			}
		}
		httpReq.Header.Set("X-Wsq-Id", req.UUID)
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
		if time.Since(start) > time.Second*10 {
			break
		}

		// oops, didn't work; have a pause and try again in a bit
		time.Sleep(sleepDuration)

		// slowly ramp up our sleep interval, shall we? But cap at a minute, thanks.
		if sleepDuration < time.Duration(time.Minute) {
			sleepDuration = time.Duration(float64(sleepDuration) * 1.5)
		} else {
			sleepDuration = time.Duration(time.Minute)
		}
	}
	elapsed := time.Since(start)

	if sent {
		deltchan <- qu
	} else {
		delfchan <- qu
	}

	log.WithFields(log.Fields{
		"id": req.UUID,
		"queue": qu.Queue,
		"url": qu.Url,
		"attempts": attempts,
		"sent": sent,
		"duration": elapsed.Seconds() * 1e3, /* ms hack */
	}).Info("relay-end")
}

type QueueUrl struct {
	Queue string
	Url   string
}

type CounterVals struct {
	Current uint64
	Total   uint64
	Success uint64
	Failure uint64
}

var counters = make(map[QueueUrl]CounterVals)
var addchan = make(chan QueueUrl, 100)
var deltchan = make(chan QueueUrl, 100)
var delfchan = make(chan QueueUrl, 100)

type Worker struct {
  QueueUrl    QueueUrl
  RequestChan chan RequestMessage
  QuitChan    chan bool
}

func (w Worker) Start() {
	go func() {
		client := &http.Client{
			// todo: reasonable default?
			Timeout: 10 * time.Second,
			// no cookies, please
			Jar: nil,
		}
		for {
			work := <-w.RequestChan
			sendEvent(client, w.QueueUrl, work)
		}
	}()
}

var sendPool = make(map[QueueUrl]Worker)

var targets TargetList

func main() {
	//log.SetFlags(log.LstdFlags | log.Lmicroseconds)
    log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
		TimestampFormat: time.RFC3339Nano,
	})

	configId := flag.String("config", "default", "Which stanza of the config to use")
	flag.Parse()

	var ok bool
	targets, ok = allTargets[*configId]
	if !ok {
		panic("Could not load expected configuration")
	}

	// initialize counters to zero
	// You don't _have_ to do this, but I like having all the counters
	// reporting 0 immediately for stat collection purposes.
	for queue, eventTargets := range targets {
		for _, eventTarget := range eventTargets {
			qu := QueueUrl{queue, eventTarget.Url}
			counters[qu] = CounterVals{0,0,0,0}
			sendPool[qu] = Worker{
				QueueUrl: qu,
				RequestChan: make(chan RequestMessage, 10000),
				QuitChan: make(chan bool),
			}
			sendPool[qu].Start()
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
			}
		}
	}()

	// A dumb goroutine to watch memory usage and counter metrics
	go func() {
		var mem runtime.MemStats
		for {
			runtime.ReadMemStats(&mem)
			log.WithFields(log.Fields{
				"mem.Alloc": mem.Alloc,
				"mem.TotalAlloc": mem.TotalAlloc,
				"mem.HeapAlloc": mem.HeapAlloc,
				"mem.HeapSys": mem.HeapSys,
				"runtime.NumGoroutine": runtime.NumGoroutine(),
			}).Info("metrics-mem")
			for cKeys, cVals:= range counters {
				log.WithFields(log.Fields{
					"queue": cKeys.Queue,
					"url": cKeys.Url,
					"current": cVals.Current,
					"total": cVals.Total,
					"success": cVals.Success,
					"failure": cVals.Failure,
					"chanlen": len(sendPool[cKeys].RequestChan),
				}).Info("metrics-queue")
			}
			time.Sleep(time.Second * 5)
		}
	}()

	// Oh, hey, there's the webserver!
	log.Info("starting server")
	for queue, _ := range targets {
		log.Info("registering queue @ /" + queue)
		http.HandleFunc("/" + queue, handleIncomingEvent)
	}
	http.HandleFunc("/", defaultHandler)
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

}
