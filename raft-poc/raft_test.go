package main

import (
	"fmt"
	"testing"
	"time"
)

// Tests run quietly.
func init() { Verbose = false }

// waitUntil polls cond until it's true or the timeout elapses.
func waitUntil(timeout time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return cond()
}

// A healthy cluster must elect exactly one leader.
func TestElectsSingleLeader(t *testing.T) {
	c := NewCluster(5)
	c.Start()
	defer c.Stop()

	leader, ok := c.WaitForLeader(2 * time.Second)
	if !ok {
		t.Fatal("no leader elected")
	}
	if leader.state != Leader {
		t.Fatalf("reported leader node %d is not in Leader state", leader.id)
	}
}

// Killing the leader must trigger a re-election among the survivors.
func TestReElectionAfterLeaderFailure(t *testing.T) {
	c := NewCluster(5)
	c.Start()
	defer c.Stop()

	old, ok := c.WaitForLeader(2 * time.Second)
	if !ok {
		t.Fatal("no initial leader")
	}
	c.Disconnect(old.id)

	newLeader, ok := c.WaitForLeader(2 * time.Second)
	if !ok {
		t.Fatal("no new leader after failure")
	}
	if newLeader.id == old.id {
		t.Fatalf("new leader is the disconnected node %d", old.id)
	}
}

// Committed commands must replicate to every node and produce identical state.
func TestReplicationConverges(t *testing.T) {
	c := NewCluster(5)
	c.Start()
	defer c.Stop()

	if _, ok := c.WaitForLeader(2 * time.Second); !ok {
		t.Fatal("no leader")
	}
	for i := 0; i < 10; i++ {
		if !c.Submit(fmt.Sprintf("set k%d v%d", i, i)) {
			t.Fatalf("submit %d rejected (no leader?)", i)
		}
	}
	ok := waitUntil(2*time.Second, func() bool { return converged(c) })
	if !ok {
		t.Fatalf("nodes did not converge; states:\n%s", dumpStates(c))
	}
}

// A node partitioned during writes must catch up once it rejoins - the key
// safety/liveness property that makes Raft useful.
func TestPartitionedNodeCatchesUp(t *testing.T) {
	c := NewCluster(5)
	c.Start()
	defer c.Stop()

	if _, ok := c.WaitForLeader(2 * time.Second); !ok {
		t.Fatal("no leader")
	}

	// Partition off a follower (not the leader) and keep writing.
	leader, _ := c.Leader()
	var victim int = -1
	for _, id := range c.ids {
		if id != leader.id {
			victim = id
			break
		}
	}
	c.Disconnect(victim)
	for i := 0; i < 8; i++ {
		c.Submit(fmt.Sprintf("set p%d v%d", i, i))
	}
	time.Sleep(200 * time.Millisecond)

	// Heal the partition; the victim must catch up to everyone else.
	c.Reconnect(victim)
	ok := waitUntil(3*time.Second, func() bool { return converged(c) })
	if !ok {
		t.Fatalf("victim did not catch up; states:\n%s", dumpStates(c))
	}
}

func dumpStates(c *Cluster) string {
	s := ""
	for _, id := range c.ids {
		s += fmt.Sprintf("  node %d: %q\n", id, c.kvs[id].Snapshot())
	}
	return s
}
