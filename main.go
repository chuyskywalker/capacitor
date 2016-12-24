package main

import (
	"net/http"
	"fmt"
	"log"
	"io/ioutil"
	"github.com/nu7hatch/gouuid"
	"time"
	"runtime"
	"bytes"
)

type RequestMessage struct {
	URL string
	Method string
	Source string
	Headers http.Header
	Body []byte
}

// why not just []string of urls? In case we need meta data for these later on.
type EventTarget struct {
	Url string
}

var targets = map[string][]EventTarget{
    "registration": []EventTarget{
		EventTarget{"http://127.0.0.1:8001/brokenville"},
		//EventTarget{"http://127.0.0.1:8000/registration2", 2, time.Duration(1)},
	},
	//"registration2": []EventTarget{
	//	EventTarget{"http://127.0.0.1:8000/registration3", 2, time.Duration(1)},
	//},
	//"registration3": []EventTarget{
	//	EventTarget{"http://127.0.0.1:8000/registration4", 2, time.Duration(1)},
	//},
	//"score-pulled": []string{
	//	"http://place.com",
	//	"http://place2.com",
	//},
	//"clicked-cc": []string{
	//	"http://place.com",
	//	"http://place2.com",
	//},
}

func handler(w http.ResponseWriter, r *http.Request) {

	body, _ := ioutil.ReadAll(r.Body)

	requestObj := RequestMessage{
		URL: r.RequestURI,
		Method: r.Method,
		Source: r.RemoteAddr,
		Headers: r.Header,
		Body: body,
	}

	// get a UUID for this transaction
	u5, err := uuid.NewV4()
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	queue := requestObj.URL[1:]

	fmt.Fprintf(w, "{ \"id\":\"%s\" }\n", u5) // to lazy to do a real json.Marshal, etc

	if eventTargets, ok := targets[queue]; ok {
		for _, eventTarget := range eventTargets {
			//log.Printf("id=%s queue=%s msg=%s url=%s\n", u5, queue, "sendingto", eventTarget.Url)
			addchan <- CounterKey{queue, eventTarget.Url}
			go sendEvent(u5, queue, eventTarget, requestObj)
		}
	} else {
		log.Printf("id=%s queue=%s msg=%s\n", u5, queue, "no such queue")
	}
}

func sendEvent(u5 *uuid.UUID, queue string, eventTarget EventTarget, req RequestMessage) {
	start := time.Now()
	var sent bool
	sent = false
	attempts := 0
	sleepDuration := time.Millisecond * 100
	for {
		attempts++

		client := &http.Client{
			Timeout: 10 * time.Second,
		}
		httpReq, _ := http.NewRequest(req.Method, eventTarget.Url, bytes.NewBuffer(req.Body))
		httpReq.Header = req.Headers
		httpReq.Header.Set("X-Wsq-Id", (*u5).String())
		resp, err := client.Do(httpReq)

		if err == nil && resp.StatusCode == 200 {
			sent = true
			break
		}

		// max duration, ever
		if time.Since(start) > time.Second * 60 {
			break
		}

		// oops, didn't work; have a pause and try again in a bit
		time.Sleep(sleepDuration)
		// slowly ramp up our sleep interval, shall we?
		sleepDuration = time.Duration(float64(sleepDuration) * 1.5);
	}
    elapsed := time.Since(start)

	delchan <- CounterKey{queue, eventTarget.Url}
	log.Printf("id=%s queue=%s msg=%s url=%s, attempts=%v, sent=%v, duration=%.3f\n",
		u5, queue, "endsend", eventTarget.Url, attempts, sent, elapsed.Seconds() * 1e3)
}

type CounterKey struct {
	Queue string
	Url string
}

var counters = make(map[CounterKey]uint64)
var addchan = make(chan CounterKey, 100)
var delchan = make(chan CounterKey, 100)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	// setup counters
	for queue, eventTargets := range targets {
		for _, eventTarget := range eventTargets {
			counters[CounterKey{queue, eventTarget.Url}] = 0
		}
	}

	// goroutine to keep the counters up-to-date
	go func(){
		for {
			// watch each channel as items rolls in and modify the counters as needed
			select {
			case control := <-addchan:
				counters[control]++
			case control := <-delchan:
				counters[control]--
			}
		}
	}()

	// A dumb goroutine to watch memory usage and counter metrics
	go func(){
		var mem runtime.MemStats
		for {
			runtime.ReadMemStats(&mem)
			log.Printf("alloc=%v totalalloc=%v, heapalloc=%v, heapsys=%v, routines=%v\n",
				mem.Alloc, mem.TotalAlloc, mem.HeapAlloc, mem.HeapSys, runtime.NumGoroutine())
			log.Printf("counters: %+v\n", counters)
			time.Sleep(time.Second * 5)
		}
	}()

	// Oh, hey, there's the webserver!
	fmt.Println("Starting server")
	http.HandleFunc("/", handler)
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

}
