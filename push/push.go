package main

import (
	"net/http"
	"time"
	"os"
	"strconv"
	"io"
	"bytes"
	"fmt"
	"io/ioutil"
)

func runClient(i, count int, d chan int) {
	fmt.Println("Starting client", i)
	client := &http.Client{
		Timeout: 1 * time.Second,
		Jar: nil,
	}
	postdata := []byte("hi there")
	for i :=0; i < count; i++ {
		//fmt.Println("Sending Request", i)
		httpReq, _ := http.NewRequest(http.MethodPost, "http://127.0.0.1:8000/demo", bytes.NewBuffer(postdata))
		resp, err := client.Do(httpReq)
		if err == nil {
			//io.Copy(os.Stdout, resp.Body)
			io.Copy(ioutil.Discard, resp.Body)
			resp.Body.Close()
		}
	}
	d <- i
}

func main() {
	d := make(chan int)
	p, _ := strconv.Atoi(os.Args[1])
	count, _ := strconv.Atoi(os.Args[2])
	fmt.Println("Starting threads...")
	start := time.Now()
	for i := 0; i < p; i++ {
		go runClient(i+1, count, d)
	}
	for i := 0; i < p; i++ {
		fmt.Println(<-d)
	}
	elapsed := time.Since(start)
	fmt.Println("Elapsed: " + elapsed.String())
}
