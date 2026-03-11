package node

import (
	"fmt"
	"sync/atomic"
)

var (
	nodeID     string
	sequenceID uint64
)

// SetNodeID sets the unique identifier for this node.
func SetNodeID(id string) {
	nodeID = id
}

// ID returns the unique identifier for this node.
func ID() string {
	return nodeID
}

// NextTaskID generates the next task ID following the pattern "<node-id>-<seq>".
func NextTaskID() string {
	seq := atomic.AddUint64(&sequenceID, 1)
	return fmt.Sprintf("%s-%d", nodeID, seq)
}

