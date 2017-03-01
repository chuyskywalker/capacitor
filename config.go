package main

/*
Configuration Syntax:

inbound:               # string of the incoming endpoint [a-zA-Z]+[a-zA-Z0-9-]*
  outbound1:           # A stats friendly name [a-zA-Z]+[a-zA-Z0-9-]*
    url: string        # "http://something" -- Full url to repost incoming requests
    queue_length: int  # Number of incoming events to hold before the queue will cycle items off the end
    max_parallel: int  # Number of simultaneous requests to make to the URL end point
  outbound2:
    url: http://otherplace.com
    queue_length: 5
    max_parallel: 1

 */

// yaml config ingestion structures
type QueueInfo struct {
	URL         string `yaml:"url"`
	QueueLength uint   `yaml:"queue_length"`
	MaxParallel uint   `yaml:"max_parallel"`
}

// A map of the "outbound name" -> outbound info
type QueueItems map[string]QueueInfo

// A map of inbound urls to outbound sets
type Config map[string]QueueItems
