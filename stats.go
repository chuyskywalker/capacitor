package main

import (
	"runtime"
	"time"
	log "github.com/Sirupsen/logrus"
)

func stats() {
	var mem runtime.MemStats
	for {
		runtime.ReadMemStats(&mem)
		log.WithFields(log.Fields{
			"mem.Alloc":            mem.Alloc,
			"mem.TotalAlloc":       mem.TotalAlloc,
			"mem.HeapAlloc":        mem.HeapAlloc,
			"mem.HeapSys":          mem.HeapSys,
			"runtime.NumGoroutine": runtime.NumGoroutine(),
		}).Info("metrics-mem")
		for cKeys, cVals := range counters {
			log.WithFields(log.Fields{
				"queue":   cKeys.InboundName,
				"outname": cKeys.OutboundName,
				"current": cVals.Current,
				"total":   cVals.Total,
				"success": cVals.Success,
				"failure": cVals.Failure,
				"lost":    cVals.Lost,
				"chanlen": len(requestBuffers[cKeys]),
				"chanmax": cap(requestBuffers[cKeys]),
			}).Info("metrics-queue")
		}
		time.Sleep(time.Second * 5)
	}
}
