package main

// This here is the meat-n-potatoes of the simple event processor
// This is a mapping of environment to target list, each target list
// deontes what endpoints gosep should listen for and where, upon
// reception, the events should be replayed to.

var allTargets = map[string]TargetList{
	"default": {
		"registration": {
			{"http://127.0.0.1:8001/brokenville"},
			{"http://127.0.0.1:8000/ep"},
		},
		"score-pulled": {
			{"http://marketing.service.consul/events/score-pull"},
			{"http://metrics.service.consul/event-post?eventId=score-pull"},
		},
		"clicked-ad": {
			{"http://marketing.service.consul/events/adclick"},
			{"http://sales.service.consul/event.php?id=clickedAd"},
		},
	},
	"dev": {
		//"registration": {{"http://127.0.0.1:9000/brokenville"}},
		"r0": {{"http://127.0.0.1:9000/brokenville"}},
		"r1": {{"http://127.0.0.1:9000/brokenville"}},
		"r2": {{"http://127.0.0.1:9000/brokenville"}},
		"r3": {{"http://127.0.0.1:9000/brokenville"}},
		"r4": {{"http://127.0.0.1:9000/brokenville"}},
		"r5": {{"http://127.0.0.1:9000/brokenville"}},
		"r6": {{"http://127.0.0.1:9000/brokenville"}},
		"r7": {{"http://127.0.0.1:9000/brokenville"}},
		"r8": {{"http://127.0.0.1:9000/brokenville"}},
		"r9": {{"http://127.0.0.1:9000/brokenville"}},
		//"loopback": {{"http://127.0.0.1:8000/non-existant"}},
	},
}

