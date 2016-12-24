package main

var allTargets = map[string]TargetList{
	"default": defaultTargets,
	"dev": devTargets,
}

// This here is the meat-n-potatoes. The map of eventids -> service endpoints
var defaultTargets = TargetList{
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
}

var devTargets = TargetList{
	"registration": {
		{"http://127.0.0.1:9000/brokenville"},
	},
}
