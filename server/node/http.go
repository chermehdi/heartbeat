package node

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
)

// HttpServer is the component that will interact with the outside world through
// a REST API to perform different operations:
//   1. Perform `Join` requests from other nodes that will later join the
//      cluster (if this is a leader)
//   2. Reply to client requests to investigate the state of the key-value store.
type HttpServer struct {
	addr     string
	listener net.Listener

	node   *Node
	logger *log.Logger
}

// NewServer will create a new `HttpServer` that will listen on `addr` later on
// after `Start` is Called.
func NewServer(addr string, node *Node) *HttpServer {
	return &HttpServer{
		addr:   addr,
		node:   node,
		logger: log.New(os.Stderr, "(Server) ", log.LstdFlags),
	}
}

// Start will start listening for incoming client requests.
// The HttpServer will create and run in it's own goroutine.
func (s *HttpServer) Start() error {
	s.logger.Printf("Starting Http server on: '%s'", s.addr)

	sv := http.Server{
		Handler: s,
	}

	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		s.logger.Fatalf("Cannot start a listener: %s", err)
		return err
	}

	http.Handle("/", s)

	s.listener = listener
	go func() {
		err := sv.Serve(listener)
		if err != nil {
			s.logger.Fatalf("Unexpected error happened: %s", err)
		}
	}()

	return nil
}

// Shutdown will be used for graceful shutdown.
func (s *HttpServer) Shutdown() {
	s.listener.Close()
}

// ServeHTTP is an implementation of the `http.Handler` interface to process
// incoming client requets.
func (s *HttpServer) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if req.URL.Path == "/join" {
		s.handleJoin(req, res)
	} else if req.URL.Path == "/services" {
		s.handleServices(req, res)
	} else if req.URL.Path == "/heartbeat" {
		s.handleHeartbeat(req, res)
	} else {
		s.badRequest(res)
	}
}

func (s *HttpServer) handleJoin(req *http.Request, res http.ResponseWriter) {
	s.logger.Printf("Join request received")
	var jr JoinRequest
	if err := json.NewDecoder(req.Body).Decode(&jr); err != nil {
		s.badRequest(res)
		return
	}
	s.logger.Printf("Node '%s' trying to join with address '%s'", jr.Id, jr.Addr)
	if err := s.node.AddPeer(jr.Id, jr.Addr); err != nil {
		s.logger.Printf("Failed to join node '%s': %s", jr.Id, err)
		s.badRequest(res)
		return
	}
	res.WriteHeader(200)
}

func (s *HttpServer) handleServices(req *http.Request, res http.ResponseWriter) {
	services := s.node.store.GetServices()
	if err := json.NewEncoder(res).Encode(services); err != nil {
		res.Write([]byte(fmt.Sprintf("Server error occured: %s", err)))
		res.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *HttpServer) handleHeartbeat(req *http.Request, res http.ResponseWriter) {
	var reg InstanceRegistration
	if err := json.NewDecoder(req.Body).Decode(&reg); err != nil {
		s.logger.Printf("Could not parse heartbeat request: %s", err)
		s.badRequest(res)
		return
	}

	s.logger.Printf("Staring instance registration for service='%s' host='%s' port='%d'", reg.ServiceName, reg.Host, reg.Port)
	s.node.store.RegisterInstance(reg)

	res.WriteHeader(http.StatusOK)
}

func (s *HttpServer) badRequest(res http.ResponseWriter) {
	res.WriteHeader(http.StatusBadRequest)
}
