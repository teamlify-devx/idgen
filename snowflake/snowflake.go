package snowflake

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
)

// Node represents a single producer node (machine/pod).
type Node struct {
	mu             sync.Mutex
	nodeID         int64
	sequence       int64
	lastMs         int64
	epoch          int64
	maxNodeID      int64
	maxSequence    int64
	nodeShift      int64
	timestampShift int64
}

/*
NewNode creates a new Node using configuration loaded from Viper.

Config keys:
  - snowflake.EPOCH: custom epoch timestamp in milliseconds
  - snowflake.NODE_BITS: number of bits for node ID (1–31) DEFAULT: 14
  - snowflake.SEQUENCE_BITS: number of bits for sequence (1–31) DEFAULT: 8
  - snowflake.NODE_TYPE: "single" (default), "k8s" (Kubernetes), "manual" (config.yaml)
  - snowflake.NODE_ID: used when NODE_TYPE is "manual"

Usage

	node, err := snowflake.NewNode()
	id, err := node.Generate()
*/
func NewNode() (*Node, error) {
	var epoch, nodeBits, sequenceBits, nodeID int64
	var nodeType string
	var err error

	// Setup Viper to read from environment variables
	viper.SetEnvPrefix("SNOWFLAKE")
	viper.AutomaticEnv()

	// Get config values with defaults - only use defaults if not explicitly set
	if epochStr := os.Getenv("SNOWFLAKE_EPOCH"); epochStr != "" {
		epoch = viper.GetInt64("EPOCH")
	} else {
		epoch = 1700000000000 // Default epoch: 2024-11-14T16:53:20Z
	}

	if nodeBitsStr := os.Getenv("SNOWFLAKE_NODE_BITS"); nodeBitsStr != "" {
		nodeBits = viper.GetInt64("NODE_BITS")
	} else {
		nodeBits = 14 // Default node bits
	}

	if sequenceBitsStr := os.Getenv("SNOWFLAKE_SEQUENCE_BITS"); sequenceBitsStr != "" {
		sequenceBits = viper.GetInt64("SEQUENCE_BITS")
	} else {
		sequenceBits = 8 // Default sequence bits
	}

	nodeType = viper.GetString("NODE_TYPE")
	if nodeType == "" {
		nodeType = "single" // Default node type
	}

	// Validate bit layout before resolving nodeID so getNodeIDFromPod
	// receives a correct maxNodeID derived from the actual config.

	if nodeBits <= 0 || nodeBits > 31 {
		return nil, errors.New("snowflake: nodeBits must be between 1 and 31")
	}

	if sequenceBits <= 0 || sequenceBits > 31 {
		return nil, errors.New("snowflake: sequenceBits must be between 1 and 31")
	}

	if nodeBits+sequenceBits >= 63 {
		return nil, errors.New("snowflake: nodeBits + sequenceBits must be less than 63")
	}

	if epoch < 0 {
		return nil, errors.New("snowflake: epoch must be non-negative")
	}

	maxNodeID := int64(-1 ^ (-1 << nodeBits))
	maxSequence := int64(-1 ^ (-1 << sequenceBits))
	nodeShift := sequenceBits
	timestampShift := nodeBits + sequenceBits

	if nodeType == "k8s" {
		nodeID, err = getNodeIDFromPod(maxNodeID)
		if err != nil {
			return nil, fmt.Errorf("failed to get nodeID from pod: %w", err)
		}
	} else if nodeType == "manual" {
		nodeID = viper.GetInt64("NODE_ID")
	} else {
		// default: single node
		nodeID = 0
	}

	if nodeID < 0 || nodeID > maxNodeID {
		return nil, fmt.Errorf("snowflake: nodeID must be between 0 and %d", maxNodeID)
	}

	return &Node{
		nodeID:         nodeID,
		epoch:          epoch,
		maxNodeID:      maxNodeID,
		maxSequence:    maxSequence,
		nodeShift:      nodeShift,
		timestampShift: timestampShift,
	}, nil
}

// Generate generates a new Snowflake ID in a thread-safe manner.
func (n *Node) Generate() (int64, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	now := currentMs()

	if now < n.lastMs {
		wait := time.Duration(n.lastMs-now) * time.Millisecond
		if wait > 10*time.Millisecond {
			return 0, errors.New("snowflake: clock moved backwards too far, refusing to generate ID")
		}
		n.mu.Unlock()
		time.Sleep(wait)
		n.mu.Lock()
		now = currentMs()
	}

	if now == n.lastMs {
		n.sequence = (n.sequence + 1) & n.maxSequence
		if n.sequence == 0 {
			// This ms is full, wait for next ms
			for now <= n.lastMs {
				now = currentMs()
			}
		}
	} else {
		n.sequence = 0
	}

	n.lastMs = now

	id := ((now - n.epoch) << n.timestampShift) | (n.nodeID << n.nodeShift) | n.sequence

	return id, nil
}

// Time returns the embedded timestamp of the ID.
func (n *Node) Time(id int64) time.Time {
	ms := (id >> n.timestampShift) + n.epoch
	return time.Unix(ms/1000, (ms%1000)*int64(time.Millisecond))
}

// NodeID returns the node that generated the ID.
func (n *Node) NodeID(id int64) int64 {
	return (id >> n.nodeShift) & n.maxNodeID
}

// Sequence returns the sequence number of the ID.
func (n *Node) Sequence(id int64) int64 {
	return id & n.maxSequence
}

// getNodeIDFromPod parses the StatefulSet ordinal from POD_NAME (e.g. "myapp-3")
// and validates it against maxNodeID derived from the configured nodeBits.
func getNodeIDFromPod(maxNodeID int64) (int64, error) {
	podName := os.Getenv("POD_NAME")
	if podName == "" {
		return 0, fmt.Errorf("POD_NAME environment variable is not set")
	}

	parts := strings.Split(podName, "-")
	ordinal := parts[len(parts)-1]
	if ordinal == "" {
		return 0, fmt.Errorf("POD_NAME %q does not end with a numeric ordinal", podName)
	}

	nodeID, err := strconv.ParseInt(ordinal, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse ordinal from POD_NAME %q: %w", podName, err)
	}

	if nodeID > maxNodeID {
		return 0, fmt.Errorf("nodeID %d exceeds max allowed %d for configured nodeBits", nodeID, maxNodeID)
	}

	return nodeID, nil
}

func currentMs() int64 {
	return time.Now().UnixMilli()
}
