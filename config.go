package main

// This here is the meat-n-potatoes of the simple event push
// This starts as a mapping of environment -> target list.
// Each target list, in turn, denotes a list of valid endpoints
// and to which gosep should push.
//
// The second param in the target is the channel size for buffering

var allTargets = map[string]TargetList{
	"default": {
		"registration": {
			{"http://127.0.0.1:8001/brokenville", 1024},
			{"http://127.0.0.1:8000/ep", 1024},
		},
		"score-pulled": {
			{"http://marketing.service.consul/events/score-pull", 1024},
			{"http://metrics.service.consul/event-post?eventId=score-pull", 1024},
		},
		"clicked-ad": {
			{"http://marketing.service.consul/events/adclick", 1024},
			{"http://sales.service.consul/event.php?id=clickedAd", 1024},
		},
	},
	"dev": {
		"demo": {
			{"http://127.0.0.1:8001/event-inbound", 10},
			{"http://127.0.0.1:8002/event-inbound", 20},
		},
	},
}

