package main

import (
	"fmt"
	"strings"
	"time"
)

// This demo runs a 5-node Raft cluster entirely in one process and walks
// through the two things Raft guarantees:
//
//  1. Leader election - the cluster picks exactly one leader, and re-elects a
//     new one within a couple hundred milliseconds if the leader disappears.
//  2. Log replication + safety - committed commands survive leader changes, and
//     every node's state machine converges to the same result.
//
// Run it:  go run .
func main() {
	fmt.Println("=== Raft POC: 5-node cluster, leader election + replicated KV store ===")

	c := NewCluster(5)
	c.Start()
	defer c.Stop()

	// --- 1. First election -------------------------------------------------
	banner("1. Waiting for the cluster to elect a leader")
	leader, ok := c.WaitForLeader(2 * time.Second)
	if !ok {
		fmt.Println("no leader emerged - something is wrong")
		return
	}
	fmt.Printf(">> node %d is the leader\n", leader.id)

	// --- 2. Replicate some commands ---------------------------------------
	banner("2. Client submits commands to the leader")
	for _, cmd := range []string{"set name raft", "set lang go", "set stars 3"} {
		c.Submit(cmd)
	}
	time.Sleep(300 * time.Millisecond) // let replication + apply settle
	showState(c, "after initial writes (all nodes should match)")

	// --- 3. Kill the leader ------------------------------------------------
	banner("3. Killing the leader - the cluster must re-elect")
	fmt.Printf(">> disconnecting node %d (the leader)\n", leader.id)
	c.Disconnect(leader.id)

	newLeader, ok := c.WaitForLeader(2 * time.Second)
	if !ok {
		fmt.Println("no new leader emerged - something is wrong")
		return
	}
	fmt.Printf(">> node %d took over as the new leader\n", newLeader.id)

	// --- 4. Keep serving on the majority side -----------------------------
	banner("4. New leader keeps accepting writes")
	for _, cmd := range []string{"set stars 4", "del name", "set status re-elected"} {
		c.Submit(cmd)
	}
	time.Sleep(300 * time.Millisecond)
	showState(c, "after writes under the new leader (old leader still down)")

	// --- 5. Revive the old leader -----------------------------------------
	banner("5. Old leader comes back and catches up")
	fmt.Printf(">> reconnecting node %d\n", leader.id)
	c.Reconnect(leader.id)
	time.Sleep(600 * time.Millisecond) // give it time to sync missed entries
	showState(c, "final state (the revived node should have caught up)")

	if converged(c) {
		fmt.Println("\nSUCCESS: every node converged to identical state.")
	} else {
		fmt.Println("\nMISMATCH: nodes disagree (unexpected).")
	}
}

func banner(title string) {
	fmt.Printf("\n%s\n%s\n", title, strings.Repeat("-", len(title)))
}

// showState prints each node's KV snapshot so you can watch them agree.
func showState(c *Cluster, label string) {
	fmt.Printf("\nKV state %s:\n", label)
	for _, id := range c.ids {
		status := "up"
		if !c.net.connected[id] {
			status = "DOWN"
		}
		fmt.Printf("  node %d [%-4s]: %s\n", id, status, c.kvs[id].Snapshot())
	}
}

// converged returns true if every connected node has the same KV snapshot.
func converged(c *Cluster) bool {
	var want string
	first := true
	for _, id := range c.ids {
		if !c.net.connected[id] {
			continue
		}
		got := c.kvs[id].Snapshot()
		if first {
			want, first = got, false
		} else if got != want {
			return false
		}
	}
	return true
}
