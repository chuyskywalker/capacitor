# Go Simple Event Push

The concept is simple; you push an event over HTTP to this API, it repeats the event to everyone in the config file.

## Demo

Setup, you'll need 4 terminals:

```
# Terminal 1
go run demo/demo.go 8001

# Terminal 2
go run demo/demo.go 8002

# Terminal 3
cd path/to/go-sep
go install
$GOPATH/bin/go-sep -config dev

# Terminal 4
# Nothing, we'll curl from here
```

Now, let's ping each of the receivers to ensure they're happy:

```
# Terminal 4
curl -s -X POST -d "12345678" http://127.0.0.1:8001/
curl -s -X POST -d "12345678" http://127.0.0.1:8002/
```

Should get back `Thanks!` and see logs show up in terminal 1 and 2.

Now, let's push out an event that the two services are listening for:

```
curl -s -X POST -d "A demo event!" http://127.0.0.1:8000/demo
```

Curl should return something like `{ "id":"e94b1336-3427-432d-59df-6dc0ee3caf22" }` and in both terminal 1 **and** 2 you should see something like `time="<timestamp here>" level=info msg="got event" content="A demo event!" id=e94b1336-3427-432d-59df-6dc0ee3caf22 len=13`.

Congratulations, you're now publishing events!

You can also look at the output of `go-sep` itself to see some metrics around the delivery rates pushing to stdout every 5 seconds. For example:

```
time="2016-12-26T14:50:20.6602892-08:00" level=info msg=metrics-queue chanlen=0 current=0 failure=0 lost=0 queue=demo success=1 total=1 url="http://127.0.0.1:8001/event-inbound"
time="2016-12-26T14:50:20.6602892-08:00" level=info msg=metrics-queue chanlen=0 current=0 failure=0 lost=0 queue=demo success=1 total=1 url="http://127.0.0.1:8002/event-inbound"
```

## Some Questions

*Why is the config a `go` file? Why not YAML, etc?*
I wanted the config to be baked into the application and very explicitly type checked. The best way I know of right now, is to make it a `go` file. I thought about allowing for `yml` files and then going a `go:generate` command to create them, but, uh, didn't take the time.

*What if a target endpoint is down?*
`go-sep` allows for a small buffer to account for endpoints which are temporarily down. Keep in mind that `go-sep` is _not_ a permanent queue solution -- it only holds items in memory! As memory is a farily finite resource, some level of failure must be set and action taken. `go-sep` utilizes buffered channels for each queue+endpoint combination and when a channel fills up, the oldest messages will be dropped. Since each queue/endpoint gets its own buffer, one service beind down does not affect others.

The amount of buffer you choose is highly variable depending on your specific situation. You must take into account how many queue+endpoint combos you have (for overall, max memory usage), how fast those events are being push into `go-sep`, the size of the average payload, and even how fast the downstream endpoints can process incoming requests. All of this will factor into how large a channel buffer you can setup; _caveat emptor_ on this front, my friends -- test and measure.
