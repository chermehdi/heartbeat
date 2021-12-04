package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

var (
	leader  = flag.String("leader", "", "The host:port address of the leader http server")
	service = flag.String("service", "test-1", "Name of the logical service for this instance")
	port    = flag.Int("port", 0, "Port where to start the http server")
)

type RegisterRequest struct {
	ServiceName string `json:"service"`
	Host        string `json:"host"`
	Port        uint16 `json:"port"`
}

type Client struct {
	leader      string
	serviceName string
	host        string
	port        uint16
}

func (c *Client) Start() {
	var req bytes.Buffer
	reg := RegisterRequest{
		ServiceName: c.serviceName,
		Host:        c.host,
		Port:        c.port,
	}

	if err := json.NewEncoder(&req).Encode(reg); err != nil {
		log.Fatalf("Cannot encode the registration request: %s", err)
	}

	bts := req.Bytes()
	for {
		_, err := http.Post(fmt.Sprintf("http://%s/heartbeat", c.leader), "application/json", bytes.NewReader(bts))
		if err != nil {
			log.Printf("Failed to contact the register server's leader, will retry in 2 seconds: %s", err)
		}
		// Perform requests every 2 seconds
		time.Sleep(time.Second * 2)
	}
}

func HandleHttp(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK!"))
}

func main() {
	flag.Parse()

	http.HandleFunc("/", HandleHttp)

	c := &Client{
		leader:      *leader,
		serviceName: *service,
		port:        uint16(*port),
		host:        "127.0.0.1",
	}

	go c.Start()

	log.Fatal(http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", *port), nil))
}
