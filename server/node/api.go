package node

import (
	"time"

	"github.com/hashicorp/raft"
)

// The timeout to execute Raft commands
var Timeout = 5 * time.Second

// Command is what we will use to change the state of the Replicate state
// machine.
// The type field defines how the message is going to be interpreted by the
// system.
type Command struct {
	Type string `json:"type"`
	Key  string `json:"key"`
	// some commands don't have a value such as `DELETE` and `GET`
	Value string `json:"value,omitempty"`
}

// A storage engine abstraction over the key-value store.
//
// The engine can be thought of as a Finite state machine that can be fed into
// the Raft cluster.
type StorageEngine interface {
	// Inherit the behaviour of a Raft state machine.
	raft.FSM

	// Put the key-value pair into the underlying storage, and return an error if it's not possible to
	// finish the operation.
	Put(string, string) error

	// Get the value identified by the given key.
	Get(string) (string, error)

	// Delete the value identifier by the given key from the underlying storage,
	// and return it.
	Delete(string) (string, error)
}

// JoinRequest is the message received by the API to handle new nodes joining
// the cluster.
type JoinRequest struct {
	Id   string `json:"id"`
	Addr string `json:"addr"`
}
