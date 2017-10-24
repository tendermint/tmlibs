package db

import (
	"fmt"
	"sort"
	"sync"
)

type cDBValue struct {
	value   []byte
	deleted bool
}

// NOTE: Needed by MultiStore.CacheWrap().
type CacheDB struct {
	mtx    sync.Mutex
	cache  map[string]cDBValue
	parent DB
}

func NewCacheDB(parent DB) *CacheDB {
	db := &CacheDB{
		cache:  make(map[string]cDBValue),
		parent: parent,
	}
	return db
}

func (db *CacheDB) Get(key []byte) []byte {
	db.mtx.Lock()
	defer db.mtx.Unlock()

	value, ok := db.cache[string(key)]
	if ok {
		return value.value
	}
	return db.parent.Get(key)
}

func (db *CacheDB) Set(key []byte, value []byte) {
	db.mtx.Lock()
	defer db.mtx.Unlock()

	db.SetNoLock(key, value)
}

func (db *CacheDB) SetSync(key []byte, value []byte) {
	db.mtx.Lock()
	defer db.mtx.Unlock()

	db.SetNoLock(key, value)
}

// NOTE: Implements atomicSetDeleter
func (db *CacheDB) SetNoLock(key []byte, value []byte) {
	db.cache[string(key)] = cDBValue{value, false}
}

func (db *CacheDB) Delete(key []byte) {
	db.mtx.Lock()
	defer db.mtx.Unlock()

	db.cache[string(key)] = cDBValue{nil, true}
}

func (db *CacheDB) DeleteSync(key []byte) {
	db.mtx.Lock()
	defer db.mtx.Unlock()

	db.cache[string(key)] = cDBValue{nil, true}
}

// NOTE: Implements atomicSetDeleter
func (db *CacheDB) DeleteNoLock(key []byte) {
	db.cache[string(key)] = cDBValue{nil, true}
}

func (db *CacheDB) Close() {
	db.mtx.Lock()
	defer db.mtx.Unlock()

	db.parent.Close()
}

func (db *CacheDB) Print() {
	db.mtx.Lock()
	defer db.mtx.Unlock()

	fmt.Println("CacheDB\ncache:\n")
	for key, value := range db.cache {
		fmt.Printf("[%X]:\t[%X]\n", []byte(key), value)
	}
	fmt.Println("\nparent:\n")
	db.parent.Print()
}

func (db *CacheDB) Stats() map[string]string {
	db.mtx.Lock()
	defer db.mtx.Unlock()

	stats := make(map[string]string)
	stats["cache.size"] = fmt.Sprintf("%d", len(db.cache))
	mergeStats(db.parent.Stats(), stats, "parent.")
	return stats
}

func (db *CacheDB) Iterator() Iterator {
	panic("CacheDB.Iterator() not yet supported")
}

func (db *CacheDB) NewBatch() Batch {
	return &memBatch{db, nil}
}

// Writes pending updates to the parent DB.
func (db *CacheDB) Commit() {
	db.mtx.Lock()
	defer db.mtx.Unlock()

	// We need a copy of all of the keys.
	// Not the best, but probably not a bottleneck depending.
	keys := []string{}
	for key, _ := range db.cache {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	batch := db.parent.NewBatch()
	for _, key := range keys {
		value := db.cache[key]
		if value.deleted {
			batch.Delete([]byte(key))
		} else {
			batch.Set([]byte(key), []byte(value.value))
		}
	}
	batch.Write()

	// Clear the cache
	db.cache = make(map[string]cDBValue)
}

func (db *CacheDB) Mutex() *sync.Mutex {
	return &(db.mtx)
}

func (db *CacheDB) CacheWrap() interface{} {
	return NewCacheDB(db)
}
