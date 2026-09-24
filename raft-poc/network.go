package main

import (
	"math/rand"
	"sync"
	"time"
)

// Network is an in-memory message bus standing in for a real network. Because
// the Raft logic only talks to the Transport interface, we can deliver RPCs by
// direct method calls here - and, crucially, we can DROP them to simulate the
// two failures Raft is built to survive:
//
//   - crash/partition: a node whose messages are all dropped (SetConnected off)
//   - flaky links: a small random delay on every message
//
// This is what lets the demo "kill the leader" deterministically without Docker
// or real sockets.
type Network struct {
	mu        sync.Mutex
	nodes     map[int]*Node
	connected map[int]bool
}

func NewNetwork() *Network {
	return &Network{
		nodes:     make(map[int]*Node),
		connected: make(map[int]bool),
	}
}

// Register adds a node to the bus, connected by default.
func (net *Network) Register(n *Node) {
	net.mu.Lock()
	defer net.mu.Unlock()
	net.nodes[n.id] = n
	net.connected[n.id] = true
}

// TransportFor returns the Transport a given node should use to reach peers.
func (net *Network) TransportFor(id int) Transport {
	return &busTransport{net: net, from: id}
}

// SetConnected attaches or detaches a node from the bus. Detaching simulates a
// crash or a network partition: every message to and from that node is lost.
func (net *Network) SetConnected(id int, up bool) {
	net.mu.Lock()
	defer net.mu.Unlock()
	net.connected[id] = up
}

// deliverable reports whether a message between from and to can pass, and
// returns the destination node.
func (net *Network) deliverable(from, to int) (*Node, bool) {
	net.mu.Lock()
	defer net.mu.Unlock()
	if !net.connected[from] || !net.connected[to] {
		return nil, false
	}
	dst, ok := net.nodes[to]
	return dst, ok
}

// busTransport is one node's view of the network.
type busTransport struct {
	net  *Network
	from int
}

func (t *busTransport) SendRequestVote(to int, args RequestVoteArgs) (RequestVoteReply, bool) {
	dst, ok := t.net.deliverable(t.from, to)
	if !ok {
		return RequestVoteReply{}, false
	}
	randomDelay()
	reply := dst.HandleRequestVote(args)
	// The reply travels back over the same link; if it dropped meanwhile, lose it.
	if _, ok := t.net.deliverable(to, t.from); !ok {
		return RequestVoteReply{}, false
	}
	return reply, true
}

func (t *busTransport) SendAppendEntries(to int, args AppendEntriesArgs) (AppendEntriesReply, bool) {
	dst, ok := t.net.deliverable(t.from, to)
	if !ok {
		return AppendEntriesReply{}, false
	}
	randomDelay()
	reply := dst.HandleAppendEntries(args)
	if _, ok := t.net.deliverable(to, t.from); !ok {
		return AppendEntriesReply{}, false
	}
	return reply, true
}

// randomDelay adds a little jitter so goroutines interleave the way they would
// on a real network - shaking out races the code must tolerate.
func randomDelay() {
	time.Sleep(time.Duration(rand.Intn(5)) * time.Millisecond)
}
