package main

// This here is the meat-n-potatoes of the simple event processor
// This is a mapping of environment to target list, each target list
// deontes what endpoints gosep should listen for and where, upon
// reception, the events should be replayed to.

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

