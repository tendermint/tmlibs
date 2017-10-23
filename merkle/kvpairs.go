package merkle

import (
	"sort"

	wire "github.com/tendermint/go-wire"
	"golang.org/x/crypto/ripemd160"
)

// A list of KVPairs is hashed via `SimpleHashFromHashables`.
// NOTE: Behavior is undefined with dup keys.
type KVPair struct {
	Key   string
	Value interface{}
}

func (kv KVPair) Hash() []byte {
	hasher, n, err := ripemd160.New(), new(int), new(error)
	wire.WriteString(kv.Key, hasher, n, err)
	if kvH, ok := kv.Value.(Hashable); ok {
		wire.WriteByteSlice(kvH.Hash(), hasher, n, err)
	} else {
		wire.WriteBinary(kv.Value, hasher, n, err)
	}
	if *err != nil {
		panic(*err)
	}
	return hasher.Sum(nil)
}

type KVPairs []KVPair

func (kvps KVPairs) Len() int           { return len(kvps) }
func (kvps KVPairs) Less(i, j int) bool { return kvps[i].Key < kvps[j].Key }
func (kvps KVPairs) Swap(i, j int)      { kvps[i], kvps[j] = kvps[j], kvps[i] }
func (kvps KVPairs) Sort()              { sort.Sort(kvps) }

func MakeSortedKVPairs(m map[string]interface{}) []Hashable {
	kvPairs := []KVPair{}
	for k, v := range m {
		kvPairs = append(kvPairs, KVPair{k, v})
	}
	KVPairs(kvPairs).Sort()
	kvPairsH := []Hashable{}
	for _, kvp := range kvPairs {
		kvPairsH = append(kvPairsH, kvp)
	}
	return kvPairsH
}
