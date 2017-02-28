# ⊣⊢ Capacitor
> HTTP request repeater

Capacitor is a queue-based, HTTP request forwarding agent. It can be used like an Event pub/sub system or as a background job processing queue.

## Basic Install

Install and run the application:
```bash
go install github.com/chuyskywalker/capacitor
$GOPATH/bin/capacitor -config example_config.yml
```

### Demo

You'll need 2 more terminals:

```
# Terminal 1, run a demo agent which will emulate two services
go run demo/demo.go

# Terminal 2
# Nothing, we'll curl from here
```

Now, let's ping the receivers to test:

```
# Terminal 2
curl -s -X POST -d "12345678" http://127.0.0.1:8001/
curl -s -X POST -d "12345678" http://127.0.0.1:8002/
```

Should get back `Thanks!` and see logs show up in terminal 1.

Now, let's push out an event that the two services are listening for:

```
curl -s -X POST -d "A demo msg!" http://127.0.0.1:8000/demo
```

Curl should return something like `{ "id":"e94b1336-3427-432d-59df-6dc0ee3caf22" }` and you should see that both 8001 and 8002 ports on the demo app got the message; it will look something like `time="<timestamp here>" level=info msg="got event" content="A demo msg!" id=e94b1336-3427-432d-59df-6dc0ee3caf22 len=13`.

Congratulations, you're now publishing!

You can also look at the output of `capacitor` itself to see some metrics around the delivery rates. For example:

```
time="2017-02-26T22:19:31.3625261-08:00" level=info msg=metrics-queue chanlen=0 chanmax=10 current=0 failure=0 lost=0 outname=demo-8001 queue=demo success=1 total=1
time="2017-02-26T22:19:31.3625261-08:00" level=info msg=metrics-queue chanlen=0 chanmax=10 current=0 failure=0 lost=0 outname=demo-8002 queue=demo success=1 total=1
```

## Configuration Format

```yaml
inbound:               # string of the incoming endpoint [a-zA-Z]+[a-zA-Z0-9-]*
  outbound1:           # A stats friendly name [a-zA-Z]+[a-zA-Z0-9-]*
    url: string        # "http://something" -- Full url to repost incoming requests
    queue_length: int  # Number of incoming events to hold before the queue will cycle items off the end
    max_parallel: int  # Number of simultaneous requests to make to the URL end point
  outbound2:
    url: http://otherplace.com
    queue_length: 5
    max_parallel: 1        
```
## FAQ

**What if a target endpoint is down?**

`Capacitor` allows for a small buffer to account for endpoints which are temporarily down. Keep in mind that `capacitor` is _not_ a permanent queue solution -- it only holds items in memory! As memory is a farily finite resource, some level of failure must be set and action taken. `Capacitor` utilizes buffered channels for each queue/endpoint combination and when a channel fills up, the oldest messages will be dropped. Since each queue/endpoint gets its own buffer, one service being down does not affect others. `Capacitor` will log when messages are dropped from the queue, but not the content of the message.

The amount of buffer you choose is highly variable depending on your specific situation. You must take into account how many queue/endpoint combos you have (for overall, max memory usage), how fast events are being pushed into `capacitor`, the size of the average event, and even how fast the downstream endpoints can process incoming requests. All of this will factor into how large a channel buffer you can setup; _caveat emptor_ on this front, my friends, test and measure.

**How do I reload the config?**

You don't. Adding run-time reload of the configuration would lead to a lot of tough to answer questions about things like configuration compatibility and how to deal with non-zero queues and endpoints. Adding support for generation configurations and layers of queues would add a great deal of complexity to the application.

My recommendataion is to, instead, place a HTTP proxy (nginx, apache, caddy, etc) in front of `capacitor`. When you wish to change the configuration, you would start a _new_ instance of `capacitor` and reconfigure the proxy to seamlessly shift traffic to the new `capacitor` instance. This allows for near immediate upgrades and allows for the old instance to finish out any open queues it may have.

This does come with the side effect that, for a short duration, you will have more parallel requests in-flight, so keep that in mind.

**How does `Capacitor` handle HA?**

It does not. `Capacitor` does not cache events to multiple agents across a network divide, and if you wanted to think about running multiple instance from a CAP perspective, it would operate as AP. Running multiple instanecs and balancing traffic between them will provide horizontal scalability, but losing a node means all messages on that node are gone as `Capacitor` is not durable in any fashion.

The other thing you'd need to watch out for is that you'll want to distribute `max_parallel` among the agents. For example, if you ultimately desire `max_parallel=15` and you have 3 agents, you must tune each config to `max_parallel=5`.
