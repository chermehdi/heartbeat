package node

import (
	"log"
	"os"
	"time"
)

// SafetyDelta is the period of time that a heart beat should be late after the
// remthreshold for the cleaner to remove it.
var SafetyDelta = uint64(100 * time.Millisecond)

type Cleaner struct {
	period       time.Duration
	remThreshold time.Duration
	node         *Node
	logger       *log.Logger
}

func NewCleaner(period, remThreshold time.Duration, node *Node) *Cleaner {
	return &Cleaner{
		period:       period,
		node:         node,
		remThreshold: remThreshold,
		logger:       log.New(os.Stderr, "(Cleaner) ", log.LstdFlags),
	}
}

func (c *Cleaner) Start() {
	c.logger.Printf("Starting entries cleanup, cleaner will run every %s", c.period)
	for {
		// For each service instance, send commands to remove them from the cluster
		// registery if they are haven't renewed their lease for more than
		// `remThreshold + SafetyDelta`.
		services := c.node.store.GetResources()
		nowMs := uint64(time.Now().UnixNano()) / uint64(1e6)
		for _, v := range services {
			for _, instance := range v.Instances {
				if nowMs-instance.LastBeatMs > uint64(c.remThreshold.Milliseconds()) {
					// Send a delete request to remove the instance.
					c.logger.Printf("Sending a delete request for instance %s:%d, of service (%s) -- difference is %dms expected at most %dms", instance.Host, instance.Port, v.Name, nowMs-instance.LastBeatMs, c.remThreshold.Milliseconds())
					c.node.store.DeleteInstance(v.Name, *instance)
				}
			}
		}
		c.logger.Printf("Scanned all the services, sleeping until the next run")
		time.Sleep(c.period)
	}
}
