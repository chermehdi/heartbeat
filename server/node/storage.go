package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/hashicorp/raft"
)

type instanceEntry struct {
	port       uint16
	host       string
	lastBeatMs uint64
	created    time.Time
}

type serviceEntry struct {
	name      string
	instances []*instanceEntry
}

type inMemStore struct {
	mu sync.Mutex
	m  map[string]string

	ms       sync.Mutex
	services map[string]*serviceEntry

	Node   *Node
	logger *log.Logger
}

func NewInMemStore() *inMemStore {
	return &inMemStore{
		mu: sync.Mutex{},
		m:  make(map[string]string),

		ms:       sync.Mutex{},
		services: make(map[string]*serviceEntry),

		logger: log.New(os.Stderr, "(Store) ", log.LstdFlags),
	}
}

func (s *inMemStore) Put(key string, value string) error {
	if s.Node.raft.State() != raft.Leader {
		// TODO: add request forwarding
		return fmt.Errorf("Cannot execute a put operation on a none-leader node")
	}

	cmd := &Command{
		Type:  "PUT",
		Key:   key,
		Value: value,
	}

	return execCommand(cmd, s.Node.raft)
}

func (s *inMemStore) Get(key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.m[key], nil
}

func (s *inMemStore) GetServices() *ServicesResponse {
	s.ms.Lock()
	defer s.ms.Unlock()

	res := &ServicesResponse{
		Services: make([]Service, 0),
	}

	for k, v := range s.services {
		service := Service{
			Name:      k,
			Instances: make([]Instance, 0),
		}
		for _, inst := range v.instances {
			// Devide by a `1000` as the `Sub` call will return a `Duration` which is
			// a type alias of int64, giving the time in nanoseconds
			uptime := uint64(time.Now().Sub(inst.created)) / uint64(1000)

			service.Instances = append(service.Instances, Instance{
				Port:   inst.port,
				Host:   inst.host,
				Uptime: uptime,
			})
		}
		res.Services = append(res.Services, service)
	}

	return res
}

func (s *inMemStore) RegisterInstance(reg InstanceRegistration) {
	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(reg); err != nil {
		s.logger.Printf("Could not serialize the registration request (%v): %s", reg, err)
		return
	}
	cmd := &Command{
		Type:  "REG",
		Value: b.String(),
	}

	execCommand(cmd, s.Node.raft)
}

func (s *inMemStore) Delete(key string) (string, error) {
	cmd := &Command{
		Type: "DEL",
		Key:  key,
	}

	// Get the current value
	cur, _ := s.Get(key)

	return cur, execCommand(cmd, s.Node.raft)
}

func execCommand(cmd *Command, rft *raft.Raft) error {
	bytes, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	ft := rft.Apply(bytes, Timeout)
	return ft.Error()
}

func (s *inMemStore) Apply(l *raft.Log) interface{} {
	var cmd Command

	if err := json.Unmarshal(l.Data, &cmd); err != nil {
		s.logger.Fatalf("Cannot unmarchall command")
	}

	switch cmd.Type {
	case "PUT":
		return s.execPut(cmd.Key, cmd.Value)
	case "DEL":
		return s.execDel(cmd.Key, cmd.Value)
	case "REG":
		return s.execReg(cmd.Value)
	default:
		s.logger.Fatalf("Cannot unmarchall command")
		return nil
	}
}

func (s *inMemStore) execReg(value string) interface{} {
	var reg InstanceRegistration
	if err := json.NewDecoder(bytes.NewReader([]byte(value))).Decode(&reg); err != nil {
		s.logger.Printf("Failed executing a registration request (%s): %s", value, err)
		return err
	}
	s.ms.Lock()
	defer s.ms.Unlock()
	se, has := s.services[reg.ServiceName]
	if !has {
		s.logger.Printf("Registering the service '%s' for the first time", reg.ServiceName)
		se = &serviceEntry{
			name:      reg.ServiceName,
			instances: make([]*instanceEntry, 0),
		}
	}

	curTime := time.Now()
	// If an entry already exists with the same host:port pair
	// act as a lease renewal, updating the last time it got updated to prevent
	// the cleaner from removing it later on.
	for _, v := range se.instances {
		if v.host == reg.Host && v.port == reg.Port {
			v.lastBeatMs = uint64(curTime.UnixNano())
			return nil
		}
	}

	se.instances = append(se.instances, &instanceEntry{
		host:       reg.Host,
		port:       reg.Port,
		created:    curTime,
		lastBeatMs: uint64(curTime.UnixNano()),
	})

	s.services[reg.ServiceName] = se
	return nil
}

func (s *inMemStore) execPut(key, value string) interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key] = value
	return nil
}

func (s *inMemStore) execDel(key, value string) interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, key)
	return nil
}

func (s *inMemStore) Snapshot() (raft.FSMSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp := make(map[string]string)
	for k, v := range s.m {
		cp[k] = v
	}
	return &storeSnapshot{store: cp}, nil
}

func (s *inMemStore) Restore(rc io.ReadCloser) error {
	m := make(map[string]string)
	if err := json.NewDecoder(rc).Decode(&m); err != nil {
		return err
	}

	s.m = m
	return nil
}

type storeSnapshot struct {
	store map[string]string
}

func (f *storeSnapshot) Persist(sink raft.SnapshotSink) error {
	perFn := func() error {
		bytes, err := json.Marshal(f.store)
		if err != nil {
			return err
		}
		if _, err := sink.Write(bytes); err != nil {
			return err
		}

		return sink.Close()
	}

	err := perFn()
	if err != nil {
		sink.Cancel()
	}
	return err
}

// Nothing required for the snapshotter required to release.
func (f *storeSnapshot) Release() { /* NOP */ }
