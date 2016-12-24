package main

import (
	"bytes"
	"fmt"
	"github.com/nu7hatch/gouuid"
	"io/ioutil"
	"log"
	"net/http"
	"runtime"
	"time"
	"flag"
)


type TargetList map[string][]EventTarget

// why not just []string of urls? In case we need meta data for these later on.
type EventTarget struct {
	Url string
}

// Our object use to repeat events.
// todo: should I just pass the http.Request? Is that thread safe?
type RequestMessage struct {
	URL     string
	Method  string
	Source  string
	Headers http.Header
	Body    []byte
}

func handler(w http.ResponseWriter, r *http.Request) {

	body, _ := ioutil.ReadAll(r.Body)

	requestObj := RequestMessage{
		URL:     r.RequestURI,
		Method:  r.Method,
		Source:  r.RemoteAddr,
		Headers: r.Header,
		Body:    body,
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
		//httpReq.Header = req.Headers
		for headerName, values := range req.Headers {
			for _, value := range values {
				httpReq.Header.Add(headerName, value)
			}
		}
		httpReq.Header.Set("X-Wsq-Id", (*u5).String())
		resp, err := client.Do(httpReq)

		if err == nil && resp.StatusCode == 200 {
			sent = true
			break
		}

		// max duration, ever
		if time.Since(start) > time.Second*60 {
			break
		}

		// oops, didn't work; have a pause and try again in a bit
		time.Sleep(sleepDuration)
		// slowly ramp up our sleep interval, shall we?
		// todo: if sleepDuration > N value, don't increase it again -- apply a cap
		sleepDuration = time.Duration(float64(sleepDuration) * 1.5)
	}
	elapsed := time.Since(start)

	if sent {
		deltchan <- CounterKey{queue, eventTarget.Url}
	} else {
		delfchan <- CounterKey{queue, eventTarget.Url}
	}

	log.Printf("id=%s queue=%s msg=%s url=%s attempts=%v sent=%v duration=%.3f\n",
		u5, queue, "endsend", eventTarget.Url, attempts, sent, elapsed.Seconds()*1e3)
}

type CounterKey struct {
	Queue string
	Url   string
}

type CounterVals struct {
	Current uint64
	Total   uint64
	Success uint64
	Failure uint64
}

var counters = make(map[CounterKey]CounterVals)
var addchan = make(chan CounterKey, 100)
var deltchan = make(chan CounterKey, 100)
var delfchan = make(chan CounterKey, 100)



var targets TargetList

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	configId := flag.String("config", "default", "Which stanza of the config to use")
	flag.Parse()

	targets, ok := allTargets[*configId]
	if !ok {
		panic("Could not load expected configuration")
	}

	// initialize counters to zero
	// You don't _have_ to do this, but I like having all the counters
	// reporting 0 immediately for stat collection purposes.
	for queue, eventTargets := range targets {
		for _, eventTarget := range eventTargets {
			counters[CounterKey{queue, eventTarget.Url}] = CounterVals{0,0,0,0}
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
			log.Printf("metrics=ram alloc=%v totalalloc=%v heapalloc=%v heapsys=%v routines=%v\n",
				mem.Alloc, mem.TotalAlloc, mem.HeapAlloc, mem.HeapSys, runtime.NumGoroutine())
			for cKeys, cVals:= range counters {
				log.Printf("metrics=queues queue=%s endpoint=%s current=%d total=%d success=%d failure=%d\n",
					cKeys.Queue, cKeys.Url, cVals.Current, cVals.Total, cVals.Success, cVals.Failure)
			}
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
