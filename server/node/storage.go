package node

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/hashicorp/raft"
)

type inMemStore struct {
	mu sync.Mutex
	m  map[string]string

	Node   *Node
	logger *log.Logger
}

func NewInMemStore() *inMemStore {
	return &inMemStore{
		mu:     sync.Mutex{},
		m:      make(map[string]string),
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
	default:
		s.logger.Fatalf("Cannot unmarchall command")
		return nil
	}
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
