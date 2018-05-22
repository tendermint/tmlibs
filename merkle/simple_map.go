package merkle

import (
	cmn "github.com/tendermint/tmlibs/common"
	"github.com/tendermint/tmlibs/merkle/tmhash"
)

// Merkle tree from a map.
// Leaves are `hash(key) | hash(value)`.
// Leaves are sorted before Merkle hashing.
type simpleMap struct {
	kvs    cmn.KVPairs
	sorted bool
}

func newSimpleMap() *simpleMap {
	return &simpleMap{
		kvs:    nil,
		sorted: false,
	}
}

// Hash the key and value and append to the kv pairs
func (sm *simpleMap) Set(key string, value Hasher) {
	sm.sorted = false

	// Hash the key to blind it... why not?
	khash := tmhash.Sum([]byte(key))

	// And the value is hashed too, so you can
	// check for equality with a cached value (say)
	// and make a determination to fetch or not.
	vhash := value.Hash()

	sm.kvs = append(sm.kvs, cmn.KVPair{
		Key:   khash,
		Value: vhash,
	})
}

// Merkle root hash of items sorted by key
// (UNSTABLE: and by value too if duplicate key).
func (sm *simpleMap) Hash() []byte {
	sm.Sort()
	return hashKVPairs(sm.kvs)
}

func (sm *simpleMap) Sort() {
	if sm.sorted {
		return
	}
	sm.kvs.Sort()
	sm.sorted = true
}

// Returns a copy of sorted KVPairs.
// NOTE these contain the hashed key and value.
func (sm *simpleMap) KVPairs() cmn.KVPairs {
	sm.Sort()
	kvs := make(cmn.KVPairs, len(sm.kvs))
	copy(kvs, sm.kvs)
	return kvs
}

//----------------------------------------

// A local extension to KVPair that can be hashed.
// XXX: key and value must already be hashed -
// otherwise the kvpair ("abc", "def") would give the same result
// as ("ab", "cdef") since we're not using length-prefixing.
type kvPair cmn.KVPair

func (kv kvPair) Hash() []byte {
	return SimpleHashFromTwoHashes(kv.Key, kv.Value)
}

func hashKVPairs(kvs cmn.KVPairs) []byte {
	kvsH := make([]Hasher, len(kvs))
	for i, kvp := range kvs {
		kvsH[i] = kvPair(kvp)
	}
	return SimpleHashFromHashers(kvsH)
}
