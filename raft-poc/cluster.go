package main

import "time"

// Cluster wires up N nodes on one in-memory network and gives the demo/tests a
// handful of convenience methods (find the leader, submit a command, kill and
// revive nodes).
type Cluster struct {
	net   *Network
	nodes map[int]*Node
	kvs   map[int]*KVStore
	ids   []int
}

// NewCluster builds (but does not start) a cluster of the given size.
func NewCluster(size int) *Cluster {
	c := &Cluster{
		net:   NewNetwork(),
		nodes: make(map[int]*Node),
		kvs:   make(map[int]*KVStore),
	}
	for id := 0; id < size; id++ {
		c.ids = append(c.ids, id)
	}
	for _, id := range c.ids {
		peers := make([]int, 0, size-1)
		for _, other := range c.ids {
			if other != id {
				peers = append(peers, other)
			}
		}
		kv := NewKVStore()
		n := NewNode(id, peers, c.net.TransportFor(id), kv)
		c.net.Register(n)
		c.nodes[id] = n
		c.kvs[id] = kv
	}
	return c
}

// Start brings every node to life.
func (c *Cluster) Start() {
	for _, id := range c.ids {
		c.nodes[id].Start()
	}
}

// Leader returns the currently connected node that believes it is leader in the
// highest term, requiring it to be unique so we don't report a transient split.
func (c *Cluster) Leader() (*Node, bool) {
	bestTerm := -1
	var leader *Node
	count := 0
	for _, id := range c.ids {
		if !c.net.connected[id] {
			continue
		}
		n := c.nodes[id]
		n.mu.Lock()
		isLeader := n.state == Leader
		term := n.currentTerm
		n.mu.Unlock()
		if isLeader {
			if term > bestTerm {
				bestTerm, leader, count = term, n, 1
			} else if term == bestTerm {
				count++
			}
		}
	}
	if leader != nil && count == 1 {
		return leader, true
	}
	return nil, false
}

// WaitForLeader polls until a stable leader emerges or the timeout elapses.
func (c *Cluster) WaitForLeader(timeout time.Duration) (*Node, bool) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if l, ok := c.Leader(); ok {
			return l, true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil, false
}

// Submit finds the leader and submits a command, returning false if there is no
// leader right now.
func (c *Cluster) Submit(command string) bool {
	l, ok := c.Leader()
	if !ok {
		return false
	}
	return l.Submit(command)
}

// Disconnect simulates crashing/partitioning a node (its messages are dropped).
func (c *Cluster) Disconnect(id int) { c.net.SetConnected(id, false) }

// Reconnect heals a previously disconnected node.
func (c *Cluster) Reconnect(id int) { c.net.SetConnected(id, true) }

// Stop shuts every node down.
func (c *Cluster) Stop() {
	for _, id := range c.ids {
		c.nodes[id].Stop()
	}
}
