package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// Verbose turns on the running commentary that makes the demo readable.
var Verbose = true

// State is which of the three Raft roles a node currently plays.
type State int

const (
	Follower State = iota
	Candidate
	Leader
	Dead // stopped (simulates a crashed process)
)

func (s State) String() string {
	switch s {
	case Follower:
		return "Follower"
	case Candidate:
		return "Candidate"
	case Leader:
		return "Leader"
	default:
		return "Dead"
	}
}

// Timing. Real Raft uses election timeouts an order of magnitude larger than
// the network round-trip, and heartbeats well under the election timeout.
// These small values keep the demo snappy while preserving that ratio.
const (
	heartbeatInterval = 40 * time.Millisecond
	electionTimeoutLo = 150 * time.Millisecond
	electionTimeoutHi = 300 * time.Millisecond
)

// Node is a single Raft server: its persistent state, volatile state, and the
// goroutines that drive elections, replication, and applying committed entries.
type Node struct {
	mu        sync.Mutex
	id        int
	peers     []int // ids of the OTHER nodes
	transport Transport
	kv        *KVStore

	// --- Persistent state (would survive a restart in a real system) ---
	currentTerm int
	votedFor    int        // -1 == "haven't voted this term"
	log         []LogEntry // log[0] is a sentinel; real entries start at index 1

	// --- Volatile state on all nodes ---
	state       State
	commitIndex int // highest log index known to be committed
	lastApplied int // highest log index applied to the KV store

	// --- Volatile state on leaders (reset after each election) ---
	nextIndex  map[int]int // for each peer, index of the next entry to send
	matchIndex map[int]int // for each peer, highest index known replicated

	// electionResetEvent is the last time we heard from a valid leader or
	// granted a vote. The election timer fires if too long passes with silence.
	electionResetEvent time.Time

	applyCond *sync.Cond // signaled when commitIndex advances
	quit      chan struct{}
}

// NewNode builds a node. Call Start to bring it to life.
func NewNode(id int, peers []int, transport Transport, kv *KVStore) *Node {
	n := &Node{
		id:          id,
		peers:       peers,
		transport:   transport,
		kv:          kv,
		votedFor:    -1,
		log:         []LogEntry{{Term: 0}}, // sentinel at index 0
		state:       Follower,
		nextIndex:   make(map[int]int),
		matchIndex:  make(map[int]int),
		commitIndex: 0,
		lastApplied: 0,
		quit:        make(chan struct{}),
	}
	n.applyCond = sync.NewCond(&n.mu)
	return n
}

// Start launches the background loops: the election timer and the apply loop.
func (n *Node) Start() {
	n.mu.Lock()
	n.electionResetEvent = time.Now()
	n.mu.Unlock()
	go n.runElectionTimer()
	go n.applyLoop()
}

// Stop simulates a crash: the node stops driving any loops. The network can
// also just disconnect a node to simulate a partition (node stays "alive" but
// its messages are dropped). See network.go.
func (n *Node) Stop() {
	n.mu.Lock()
	n.state = Dead
	n.mu.Unlock()
	select {
	case <-n.quit:
	default:
		close(n.quit)
	}
	n.applyCond.Broadcast() // wake the apply loop so it can exit
	n.dlog("stopped (crash simulated)")
}

func (n *Node) dlog(format string, args ...any) {
	if Verbose {
		fmt.Printf("[node %d | term %d | %-9s] %s\n",
			n.id, n.currentTerm, n.state, fmt.Sprintf(format, args...))
	}
}

// ---------------------------------------------------------------------------
// Log helpers (must be called with n.mu held)
// ---------------------------------------------------------------------------

func (n *Node) lastLogIndex() int { return len(n.log) - 1 }
func (n *Node) lastLogTerm() int  { return n.log[len(n.log)-1].Term }

// ---------------------------------------------------------------------------
// Election
// ---------------------------------------------------------------------------

// runElectionTimer waits out a randomized timeout and, if the node hasn't heard
// from a leader (or become one) by then, starts an election. The randomization
// is the trick that prevents split votes: nodes rarely time out simultaneously.
func (n *Node) runElectionTimer() {
	timeout := electionTimeoutLo + time.Duration(rand.Int63n(int64(electionTimeoutHi-electionTimeoutLo)))
	n.mu.Lock()
	termStarted := n.currentTerm
	n.mu.Unlock()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-n.quit:
			return
		case <-ticker.C:
		}

		n.mu.Lock()
		// A leader doesn't run an election timer; if we became one, stop.
		if n.state == Leader || n.state == Dead {
			n.mu.Unlock()
			return
		}
		// If the term moved on, this timer is stale; a fresh one was started.
		if termStarted != n.currentTerm {
			n.mu.Unlock()
			return
		}
		if elapsed := time.Since(n.electionResetEvent); elapsed >= timeout {
			n.startElection()
			n.mu.Unlock()
			return
		}
		n.mu.Unlock()
	}
}

// startElection transitions this node to Candidate and campaigns for votes.
// Called with n.mu held.
func (n *Node) startElection() {
	n.state = Candidate
	n.currentTerm++
	n.votedFor = n.id
	n.electionResetEvent = time.Now()
	savedTerm := n.currentTerm
	lastIdx, lastTerm := n.lastLogIndex(), n.lastLogTerm()
	n.dlog("election timeout -> becoming candidate, requesting votes")

	votes := 1 // vote for self
	for _, peer := range n.peers {
		go func(peer int) {
			args := RequestVoteArgs{
				Term:         savedTerm,
				CandidateID:  n.id,
				LastLogIndex: lastIdx,
				LastLogTerm:  lastTerm,
			}
			reply, ok := n.transport.SendRequestVote(peer, args)
			if !ok {
				return // message lost (peer down / partitioned)
			}
			n.mu.Lock()
			defer n.mu.Unlock()

			// We may have moved on since sending (newer term, or already lost).
			if n.state != Candidate || n.currentTerm != savedTerm {
				return
			}
			if reply.Term > n.currentTerm {
				n.becomeFollower(reply.Term)
				return
			}
			if reply.VoteGranted {
				votes++
				if votes*2 > len(n.peers)+1 { // strict majority of the cluster
					n.dlog("won election with %d votes -> becoming LEADER", votes)
					n.becomeLeader()
				}
			}
		}(peer)
	}

	// If this election fizzles (split vote), a new timer will fire another.
	go n.runElectionTimer()
}

// becomeFollower steps down to follower in a (possibly new) term.
// Called with n.mu held.
func (n *Node) becomeFollower(term int) {
	n.dlog("stepping down to follower (term %d)", term)
	n.state = Follower
	n.currentTerm = term
	n.votedFor = -1
	n.electionResetEvent = time.Now()
	go n.runElectionTimer()
}

// becomeLeader initializes leader state and starts sending heartbeats.
// Called with n.mu held.
func (n *Node) becomeLeader() {
	n.state = Leader
	for _, peer := range n.peers {
		n.nextIndex[peer] = n.lastLogIndex() + 1
		n.matchIndex[peer] = 0
	}
	go n.leaderLoop()
}

// leaderLoop sends heartbeats/replication to all peers on a fixed interval for
// as long as this node remains leader.
func (n *Node) leaderLoop() {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	for {
		n.replicateToAll()
		select {
		case <-n.quit:
			return
		case <-ticker.C:
		}
		n.mu.Lock()
		isLeader := n.state == Leader
		n.mu.Unlock()
		if !isLeader {
			return
		}
	}
}

// ---------------------------------------------------------------------------
// Log replication (leader side)
// ---------------------------------------------------------------------------

// Submit is the client entry point. If this node is the leader it appends the
// command to its log and kicks off replication, returning true. Otherwise it
// returns false and the client must find the leader.
func (n *Node) Submit(command string) bool {
	n.mu.Lock()
	if n.state != Leader {
		n.mu.Unlock()
		return false
	}
	n.log = append(n.log, LogEntry{Term: n.currentTerm, Command: command})
	n.dlog("accepted command %q at index %d", command, n.lastLogIndex())
	n.mu.Unlock()
	n.replicateToAll() // don't wait for the next heartbeat
	return true
}

// replicateToAll sends the appropriate AppendEntries to every peer.
func (n *Node) replicateToAll() {
	n.mu.Lock()
	if n.state != Leader {
		n.mu.Unlock()
		return
	}
	savedTerm := n.currentTerm
	n.mu.Unlock()
	for _, peer := range n.peers {
		go n.replicateTo(peer, savedTerm)
	}
}

// replicateTo sends one AppendEntries to a single peer and processes the reply,
// adjusting nextIndex/matchIndex and advancing the commit index.
func (n *Node) replicateTo(peer, savedTerm int) {
	n.mu.Lock()
	if n.state != Leader || n.currentTerm != savedTerm {
		n.mu.Unlock()
		return
	}
	ni := n.nextIndex[peer]
	prevLogIndex := ni - 1
	prevLogTerm := n.log[prevLogIndex].Term
	entries := append([]LogEntry(nil), n.log[ni:]...) // copy entries to send
	args := AppendEntriesArgs{
		Term:         savedTerm,
		LeaderID:     n.id,
		PrevLogIndex: prevLogIndex,
		PrevLogTerm:  prevLogTerm,
		Entries:      entries,
		LeaderCommit: n.commitIndex,
	}
	n.mu.Unlock()

	reply, ok := n.transport.SendAppendEntries(peer, args)
	if !ok {
		return
	}

	n.mu.Lock()
	defer n.mu.Unlock()
	if n.state != Leader || n.currentTerm != savedTerm {
		return
	}
	if reply.Term > n.currentTerm {
		n.becomeFollower(reply.Term)
		return
	}
	if reply.Success {
		n.matchIndex[peer] = prevLogIndex + len(entries)
		n.nextIndex[peer] = n.matchIndex[peer] + 1
		n.maybeAdvanceCommit()
	} else {
		// Follower's log didn't match. Back nextIndex up (using the follower's
		// hint) and we'll retry on the next heartbeat.
		if reply.ConflictIndex > 0 {
			n.nextIndex[peer] = reply.ConflictIndex
		} else if n.nextIndex[peer] > 1 {
			n.nextIndex[peer]--
		}
	}
}

// maybeAdvanceCommit moves commitIndex forward to the highest index that is
// replicated on a majority AND belongs to the current term. The current-term
// requirement is a subtle but critical safety rule from the paper (§5.4.2):
// a leader may not commit entries from previous terms by counting replicas.
// Called with n.mu held.
func (n *Node) maybeAdvanceCommit() {
	for idx := n.lastLogIndex(); idx > n.commitIndex; idx-- {
		if n.log[idx].Term != n.currentTerm {
			continue
		}
		count := 1 // this leader counts itself
		for _, peer := range n.peers {
			if n.matchIndex[peer] >= idx {
				count++
			}
		}
		if count*2 > len(n.peers)+1 {
			n.commitIndex = idx
			n.dlog("commitIndex advanced to %d", idx)
			n.applyCond.Broadcast()
			return
		}
	}
}

// ---------------------------------------------------------------------------
// RPC handlers (called by the transport when a message arrives)
// ---------------------------------------------------------------------------

// HandleRequestVote decides whether to grant a vote to a candidate.
func (n *Node) HandleRequestVote(args RequestVoteArgs) RequestVoteReply {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.state == Dead {
		return RequestVoteReply{}
	}
	if args.Term > n.currentTerm {
		n.becomeFollower(args.Term)
	}
	reply := RequestVoteReply{Term: n.currentTerm}

	// Grant the vote only if: same term, haven't voted for someone else, and
	// the candidate's log is at least as up-to-date as ours (election
	// restriction, §5.4.1 - this is what keeps committed entries safe).
	upToDate := args.LastLogTerm > n.lastLogTerm() ||
		(args.LastLogTerm == n.lastLogTerm() && args.LastLogIndex >= n.lastLogIndex())
	if args.Term == n.currentTerm &&
		(n.votedFor == -1 || n.votedFor == args.CandidateID) &&
		upToDate {
		n.votedFor = args.CandidateID
		reply.VoteGranted = true
		n.electionResetEvent = time.Now()
		n.dlog("granted vote to node %d", args.CandidateID)
	}
	return reply
}

// HandleAppendEntries processes replication/heartbeat from a leader.
func (n *Node) HandleAppendEntries(args AppendEntriesArgs) AppendEntriesReply {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.state == Dead {
		return AppendEntriesReply{}
	}
	if args.Term > n.currentTerm {
		n.becomeFollower(args.Term)
	}
	reply := AppendEntriesReply{Term: n.currentTerm}

	// Reject messages from a stale leader.
	if args.Term < n.currentTerm {
		reply.Success = false
		return reply
	}

	// Valid current leader: (re)acknowledge it and reset our election timer so
	// we don't start a needless election.
	if n.state != Follower {
		n.becomeFollower(args.Term)
	}
	n.electionResetEvent = time.Now()

	// --- Log consistency check ---
	// If we don't even have an entry at PrevLogIndex, tell the leader where our
	// log ends so it can back up quickly.
	if args.PrevLogIndex > n.lastLogIndex() {
		reply.Success = false
		reply.ConflictIndex = n.lastLogIndex() + 1
		return reply
	}
	// If the terms at PrevLogIndex disagree, our logs have diverged there.
	// Report the first index of the conflicting term so the leader rewinds past
	// the whole bad run in one step.
	if n.log[args.PrevLogIndex].Term != args.PrevLogTerm {
		conflictTerm := n.log[args.PrevLogIndex].Term
		i := args.PrevLogIndex
		for i > 1 && n.log[i-1].Term == conflictTerm {
			i--
		}
		reply.Success = false
		reply.ConflictIndex = i
		return reply
	}

	// --- Logs match up to PrevLogIndex: append the new entries ---
	// Walk the incoming entries; overwrite on the first conflict, otherwise
	// skip entries we already have (idempotent - heartbeats may resend).
	insertAt := args.PrevLogIndex + 1
	for i, entry := range args.Entries {
		idx := insertAt + i
		if idx <= n.lastLogIndex() && n.log[idx].Term != entry.Term {
			n.log = n.log[:idx] // truncate the divergent tail
		}
		if idx > n.lastLogIndex() {
			n.log = append(n.log, args.Entries[i:]...)
			break
		}
	}
	reply.Success = true

	// Adopt the leader's commit index (bounded by what we actually have).
	if args.LeaderCommit > n.commitIndex {
		last := args.PrevLogIndex + len(args.Entries)
		n.commitIndex = min(args.LeaderCommit, last)
		n.applyCond.Broadcast()
	}
	return reply
}

// ---------------------------------------------------------------------------
// Applying committed entries to the state machine
// ---------------------------------------------------------------------------

// applyLoop waits for commitIndex to advance and applies newly committed entries
// to the KV store, in order. This is where "agreed-upon log" becomes "actual
// state" that a client can read.
func (n *Node) applyLoop() {
	n.mu.Lock()
	defer n.mu.Unlock()
	for {
		if n.state == Dead {
			return
		}
		if n.commitIndex > n.lastApplied {
			n.lastApplied++
			entry := n.log[n.lastApplied]
			n.mu.Unlock()
			n.kv.Apply(entry.Command) // apply without holding the Raft lock
			n.mu.Lock()
		} else {
			n.applyCond.Wait()
		}
	}
}
