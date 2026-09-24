# raft-poc

A small, dependency-free implementation of the [Raft consensus algorithm](https://raft.github.io/raft.pdf)
in Go: leader election, log replication, and a replicated key/value store, with
an in-memory network that can simulate crashes and partitions.

The whole point of Raft is to make a **cluster of machines agree on an ordered
log of commands**, so that even as machines crash and leaders change, every
node's state machine ends up identical. This POC lets you watch that happen.

## Run it

```bash
go run .          # narrated 5-node demo: elect, replicate, kill leader, recover
go test -race     # election, re-election, replication, and catch-up tests
```

`go run .` walks through five stages and prints each node's KV state so you can
see them agree. `go test -race` proves the properties hold under the race
detector.

## The idea in one paragraph

At any moment each node is a **Follower**, **Candidate**, or **Leader**. Followers
just respond to messages. If a follower hears nothing from a leader for a
randomized timeout, it becomes a candidate and starts an **election** (term + 1,
vote for self, ask everyone for votes). Win a majority and you're the leader.
The leader takes client commands, appends them to its **log**, and replicates
them to followers. Once an entry is stored on a majority it's **committed**, and
every node applies it to its state machine in the same order. Randomized
election timeouts keep two nodes from campaigning at once; the "term" number
lets any node instantly recognize and defer to newer leadership.

## Files (read them in this order)

| File | What it teaches |
|------|-----------------|
| `types.go`   | The **only two RPCs** in Raft — `RequestVote` and `AppendEntries` — and the log entry format. |
| `kv.go`      | The state machine on top of Raft. Raft agrees on *order*; this gives commands *meaning*. |
| `node.go`    | The heart: election timer, becoming candidate/leader/follower, log replication, commit rules, and applying entries. |
| `network.go` | An in-memory transport that can **drop messages** to simulate crash/partition — how the demo kills the leader deterministically. |
| `cluster.go` | Convenience wiring for N nodes: find the leader, submit commands, disconnect/reconnect. |
| `main.go`    | The narrated demo scenario. |
| `raft_test.go` | Property tests: single leader, re-election, convergence, catch-up. |

## The three safety rules worth remembering

These are the subtle parts of `node.go` that make Raft *correct*, not just
plausible:

1. **Election restriction** (`HandleRequestVote`): a node only votes for a
   candidate whose log is *at least as up-to-date* as its own. This is what
   guarantees a new leader already has every committed entry.
2. **Log matching** (`HandleAppendEntries`): a follower rejects an
   `AppendEntries` unless the entry just before the new ones matches in both
   index *and* term. Divergent tails get truncated and overwritten.
3. **Commit only current-term entries** (`maybeAdvanceCommit`): a leader
   advances the commit index by counting replicas *only* for entries from its
   own term. Committing an older-term entry by replica count alone can be unsafe
   (Raft paper §5.4.2).

## What this POC leaves out (on purpose)

Real Raft also needs **persistence** (writing `currentTerm`, `votedFor`, and the
log to disk before responding), **log compaction / snapshots** (so the log
doesn't grow forever), and **cluster membership changes**. They're described in
the paper; the core protocol here is the foundation they build on.
