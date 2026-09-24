package main

// This file defines the on-the-wire data structures for Raft.
//
// Raft only has TWO RPCs. That is the whole protocol:
//
//   RequestVote   - sent by candidates to win an election
//   AppendEntries - sent by the leader to replicate log entries
//                   (an AppendEntries with no entries is a heartbeat)
//
// Everything else - leader election, log replication, safety - falls out of
// how nodes REACT to these two messages. See node.go for the reactions.

// LogEntry is one command in the replicated log.
//
// The core idea of Raft: if every node applies the same commands, in the same
// order, to the same starting state, they all end up in the same final state.
// The log is that ordered list of commands. Term is stamped on so nodes can
// detect and resolve inconsistencies (an entry is only "the same" if both its
// index AND its term match on two nodes).
type LogEntry struct {
	Term    int    // term in which the leader created this entry
	Command string // opaque command for the state machine, e.g. "set x 1"
}

// --- RequestVote RPC -------------------------------------------------------

// RequestVoteArgs is sent by a candidate asking a peer for its vote.
type RequestVoteArgs struct {
	Term         int // candidate's term
	CandidateID  int // who is asking
	LastLogIndex int // index of candidate's last log entry
	LastLogTerm  int // term of candidate's last log entry
}

// RequestVoteReply is the peer's answer.
type RequestVoteReply struct {
	Term        int  // responder's currentTerm, so a stale candidate can update itself
	VoteGranted bool // true means the candidate got this vote
}

// --- AppendEntries RPC -----------------------------------------------------

// AppendEntriesArgs is sent by the leader to replicate entries (or as an empty
// heartbeat to assert leadership and prevent new elections).
type AppendEntriesArgs struct {
	Term         int        // leader's term
	LeaderID     int        // so followers can redirect clients
	PrevLogIndex int        // index of log entry immediately preceding the new ones
	PrevLogTerm  int        // term of that PrevLogIndex entry
	Entries      []LogEntry // entries to store (empty for heartbeat)
	LeaderCommit int        // leader's commitIndex
}

// AppendEntriesReply is the follower's answer.
type AppendEntriesReply struct {
	Term    int  // responder's currentTerm, for the leader to update itself
	Success bool // true if follower's log matched PrevLogIndex/PrevLogTerm

	// ConflictIndex lets the leader back up nextIndex quickly instead of one
	// entry at a time. This is an optimization from the Raft paper (§5.3) that
	// makes recovery after a long partition fast.
	ConflictIndex int
}

// Transport is how a node sends RPCs to its peers. Abstracting this lets us
// swap a real network for an in-memory one that can simulate crashes and
// network partitions (see network.go). The Raft logic in node.go never knows
// which is behind the interface.
//
// ok == false means "the message was lost" (peer down or partitioned). Raft is
// designed to tolerate exactly this: lost messages are indistinguishable from a
// slow peer, and the protocol simply retries.
type Transport interface {
	SendRequestVote(to int, args RequestVoteArgs) (RequestVoteReply, bool)
	SendAppendEntries(to int, args AppendEntriesArgs) (AppendEntriesReply, bool)
}
