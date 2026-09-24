package main

import (
	"fmt"
	"strings"
	"sync"
)

// KVStore is the "state machine" that sits on top of Raft.
//
// Raft's job is only to agree on an ORDER of commands. It doesn't care what the
// commands mean. Here we interpret each committed command as a simple key/value
// operation. Because Raft guarantees every node applies the same commands in
// the same order, every node's KVStore ends up identical - that is the whole
// point of the exercise.
//
// Supported commands (space separated strings):
//
//	set <key> <value>
//	del <key>
type KVStore struct {
	mu   sync.Mutex
	data map[string]string
}

func NewKVStore() *KVStore {
	return &KVStore{data: make(map[string]string)}
}

// Apply executes one committed command against the store. It is called by the
// Raft node's apply loop, in log order, once an entry is known to be committed.
func (kv *KVStore) Apply(command string) {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return
	}
	kv.mu.Lock()
	defer kv.mu.Unlock()
	switch fields[0] {
	case "set":
		if len(fields) >= 3 {
			kv.data[fields[1]] = strings.Join(fields[2:], " ")
		}
	case "del":
		if len(fields) >= 2 {
			delete(kv.data, fields[1])
		}
	}
}

// Get returns the value for a key (for inspection in the demo/tests).
func (kv *KVStore) Get(key string) (string, bool) {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	v, ok := kv.data[key]
	return v, ok
}

// Snapshot returns a stable string of the whole store so we can compare two
// nodes and prove they converged to the same state.
func (kv *KVStore) Snapshot() string {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	keys := make([]string, 0, len(kv.data))
	for k := range kv.data {
		keys = append(keys, k)
	}
	// simple insertion sort keeps this file dependency-free
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j-1] > keys[j]; j-- {
			keys[j-1], keys[j] = keys[j], keys[j-1]
		}
	}
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s=%s ", k, kv.data[k])
	}
	return strings.TrimSpace(b.String())
}
