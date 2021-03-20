package node

import (
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/hashicorp/raft"
)

type Node struct {
	dataDir  string
	raftAddr string
	id       string

	store  StorageEngine
	raft   *raft.Raft
	logger *log.Logger
}

func NewNode(id, dataDir, raftAddr string, store StorageEngine) *Node {
	return &Node{
		dataDir:  dataDir,
		raftAddr: raftAddr,
		store:    store,
		id:       id,
		// Adding a prefix to the logger will help trace which calls came in from
		// withing a node method, and which from other parts of the code.
		logger: log.New(os.Stderr, "(Node) ", log.LstdFlags),
	}
}

// Bootstrap creates the underlying Raft node and configures it according to the
// supplied parameters / node state.
//
// If the `isLeader` param is set, it will initiate a cluster with size `1` with
// the current node as the leader.
func (n *Node) Bootstrap(isLeader bool) error {
	n.logger.Printf("Bootsrapping the cluster with the default configuration...")
	conf := raft.DefaultConfig()
	conf.LocalID = raft.ServerID(n.id)

	addr, err := net.ResolveTCPAddr("tcp", n.raftAddr)
	if err != nil {
		return err
	}

	// Create the transport for the Raft RPCs
	transport, err := raft.NewTCPTransport(n.raftAddr, addr, 5, 20*time.Second, os.Stderr)
	if err != nil {
		return err
	}

	n.logger.Printf("Created transport at '%s'", transport.LocalAddr())

	// Create a snapshoter to truncate the logs.
	snapshots, err := raft.NewFileSnapshotStore(n.dataDir, 3, os.Stderr)
	if err != nil {
		return err
	}
	n.logger.Printf("Created snapshotter in '%s'", n.dataDir)

	n.logger.Printf("Creating the log store")
	logStore := raft.NewInmemStore()

	n.logger.Printf("Creating the stable store")
	stableStore := raft.NewInmemStore()

	rft, err := raft.NewRaft(conf, n.store, logStore, stableStore, snapshots, transport)
	if err != nil {
		return err
	}

	n.logger.Printf("Initiliazed the Raft node %v", isLeader)
	n.raft = rft

	if isLeader {
		n.logger.Printf("This node is supposed to be a leader, bootstrapping a single node cluster")
		// Bootstrapping the leader to create a single node cluster.
		// Later on, we will add voters to the same node's cluster to populate the
		// Raft cluster.
		ft := rft.BootstrapCluster(raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      conf.LocalID,
					Address: transport.LocalAddr(),
				},
			},
		})
		if err = ft.Error(); err != nil {
			n.logger.Printf("Bootrapping the cluster finished with error '%v", err)
		}
	}
	return nil
}

func (n *Node) AddPeer(id, addr string) error {
	n.logger.Printf("Adding a peer to the Raft cluster '%s' at '%s'", id, addr)

	confFt := n.raft.GetConfiguration()
	if err := confFt.Error(); err != nil {
		n.logger.Printf("Failed to get the raft configuration: %s", err)
		return err
	}

	conf := confFt.Configuration()
	return n.addVoter(conf, id, addr)
}

func (n *Node) addVoter(conf raft.Configuration, id, addr string) error {
	for _, srv := range conf.Servers {
		if srv.ID == raft.ServerID(id) || srv.Address == raft.ServerAddress(addr) {
			if srv.ID == raft.ServerID(id) && srv.Address == raft.ServerAddress(addr) {
				n.logger.Printf("Node '%s' is already a member in the cluster,  AddPeer request ignored", id)
				return nil
			}
			ft := n.raft.RemoveServer(srv.ID, 0, 0)
			if err := ft.Error(); err != nil {
				return fmt.Errorf("Error removing node '%s' from the Raft cluster: %s", id, err)
			}
		}
	}
	ft := n.raft.AddVoter(raft.ServerID(id), raft.ServerAddress(addr), 0, 0)
	if err := ft.Error(); err != nil {
		return fmt.Errorf("Error adding node '%s' as a voter in the Raft cluster: %s", id, err)
	}

	n.logger.Printf("Node '%s' at '%s' joined the cluster successfully", id, addr)
	return nil
}

func (n *Node) ServeHTTP() {

}
