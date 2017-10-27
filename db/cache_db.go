package db

import (
	"fmt"
	"sort"
	"sync"
)

// If value is nil but deleted is false,
// it means the parent doesn't have the key.
// (No need to delete upon Commit())
type cDBValue struct {
	value   []byte
	deleted bool
}

// CacheDB wraps an in-memory cache around an underlying DB.
type CacheDB struct {
	mtx    sync.Mutex
	cache  map[string]cDBValue
	parent DB
}

// NOTE: Needed by MultiStore.CacheWrap().
var _ atomicSetDeleter = (*CacheDB)(nil)

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
	if !ok {
		data := db.parent.Get(key)
		value = cDBValue{value: data}
		db.cache[string(key)] = value
	}
	return value.value
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

func (db *CacheDB) SetNoLock(key []byte, value []byte) {
	db.cache[string(key)] = cDBValue{value, false}
}

func (db *CacheDB) Delete(key []byte) {
	db.mtx.Lock()
	defer db.mtx.Unlock()

	db.DeleteNoLock(key)
}

func (db *CacheDB) DeleteSync(key []byte) {
	db.mtx.Lock()
	defer db.mtx.Unlock()

	db.DeleteNoLock(key)
}

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

	fmt.Println("CacheDB\ncache:")
	for key, value := range db.cache {
		fmt.Printf("[%X]:\t[%v]\n", []byte(key), value)
	}
	fmt.Println("\nparent:")
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

// Commit writes pending updates to the parent database and clears the cache.
func (db *CacheDB) Commit() {
	db.mtx.Lock()
	defer db.mtx.Unlock()

	// We need a copy of all of the keys.
	// Not the best, but probably not a bottleneck depending.
	keys := make([]string, 0, len(db.cache))
	for key := range db.cache {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	batch := db.parent.NewBatch()
	for _, key := range keys {
		value := db.cache[key]
		if value.deleted {
			batch.Delete([]byte(key))
		} else if value.value == nil {
			// Skip, it already doesn't exist in parent.
		} else {
			batch.Set([]byte(key), value.value)
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
