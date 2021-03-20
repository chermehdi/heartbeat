package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/chermehdi/heartbeat/server/node"
)

var (
	port       = flag.Int("port", 9000, "Port used by the client to connect")
	rport      = flag.Int("rport", 9999, "Port used by the underlying Raft protocol")
	leaderAddr = flag.String("leader", "", "The leader's join address, if this node is the leader when bootstrapping the cluster, this should be empty")
	storageDir = flag.String("sdir", "/tmp/heartbeat/data", "A path to the storage directory")
	id         = flag.String("id", "node-1", "Node identifier")
)

func main() {
	flag.Parse()
	log.SetOutput(os.Stderr)

	log.Printf("Starting the server at '127.0.0.1:%d' with id='%s'", *port, *id)

	os.Mkdir(*storageDir, 0775)

	storage := node.NewInMemStore()
	nd := node.NewNode(*id, *storageDir, fmt.Sprintf("127.0.0.1:%d", *rport), storage)

	storage.Node = nd

	httpServer := node.NewServer(fmt.Sprintf(":%d", *port), nd)

	if err := nd.Bootstrap(*leaderAddr == ""); err != nil {
		log.Fatalf("Bootrapping finished with errors: %s", err)
	}

	if err := httpServer.Start(); err != nil {
		log.Fatalf("Could not start the http server: %s", err)
	}

	if *leaderAddr != "" {
		b, err := json.Marshal(node.JoinRequest{Id: *id, Addr: fmt.Sprintf(":%d", *rport)})
		if err != nil {
			log.Fatalf("Could not marshal join request: %s", err)
		}
		res, err := http.Post(fmt.Sprintf("http://%s/join", *leaderAddr), "application/json", bytes.NewReader(b))
		if err != nil {
			log.Fatalf("Failed to join the leader: %s", err)
		}
		res.Body.Close()
	}

	time.Sleep(300 * time.Second)
}
