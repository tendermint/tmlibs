package db

import (
	"fmt"
	"sort"
	"sync"
)

func init() {
	registerDBCreator(MemDBBackendStr, func(name string, dir string) (DB, error) {
		return NewMemDB(), nil
	}, false)
}

type MemDB struct {
	mtx sync.Mutex
	db  map[string][]byte
}

func NewMemDB() *MemDB {
	database := &MemDB{db: make(map[string][]byte)}
	return database
}

func (db *MemDB) Get(key []byte) []byte {
	db.mtx.Lock()
	defer db.mtx.Unlock()
	return db.db[string(key)]
}

func (db *MemDB) GetOK(key []byte) ([]byte, bool) {
	db.mtx.Lock()
	defer db.mtx.Unlock()
	value, ok := db.db[string(key)]
	return value, ok
}

func (db *MemDB) Set(key []byte, value []byte) {
	db.mtx.Lock()
	defer db.mtx.Unlock()
	db.SetNoLock(key, value)
}

func (db *MemDB) SetSync(key []byte, value []byte) {
	db.mtx.Lock()
	defer db.mtx.Unlock()
	db.SetNoLock(key, value)
}

// NOTE: Implements atomicSetDeleter
func (db *MemDB) SetNoLock(key []byte, value []byte) {
	if value == nil {
		value = []byte{}
	}
	db.db[string(key)] = value
}

func (db *MemDB) Delete(key []byte) {
	db.mtx.Lock()
	defer db.mtx.Unlock()
	delete(db.db, string(key))
}

func (db *MemDB) DeleteSync(key []byte) {
	db.mtx.Lock()
	defer db.mtx.Unlock()
	delete(db.db, string(key))
}

// NOTE: Implements atomicSetDeleter
func (db *MemDB) DeleteNoLock(key []byte) {
	delete(db.db, string(key))
}

func (db *MemDB) Close() {
	// Close is a noop since for an in-memory
	// database, we don't have a destination
	// to flush contents to nor do we want
	// any data loss on invoking Close()
	// See the discussion in https://github.com/tendermint/tmlibs/pull/56
}

func (db *MemDB) Print() {
	db.mtx.Lock()
	defer db.mtx.Unlock()
	for key, value := range db.db {
		fmt.Printf("[%X]:\t[%X]\n", []byte(key), value)
	}
}

func (db *MemDB) Stats() map[string]string {
	stats := make(map[string]string)
	stats["database.type"] = "memDB"
	stats["database.size"] = fmt.Sprintf("%d", len(db.db))
	return stats
}

type memDBIterator struct {
	last int
	keys []string
	db   *MemDB
}

func newMemDBIterator() *memDBIterator {
	return &memDBIterator{}
}

func (it *memDBIterator) Next() bool {
	if it.last >= len(it.keys)-1 {
		return false
	}
	it.last++
	return true
}

func (it *memDBIterator) Key() []byte {
	return []byte(it.keys[it.last])
}

func (it *memDBIterator) Value() []byte {
	return it.db.Get(it.Key())
}

func (it *memDBIterator) Release() {
	it.db = nil
	it.keys = nil
}

func (it *memDBIterator) Error() error {
	return nil
}

func (db *MemDB) Iterator() Iterator {
	it := newMemDBIterator()
	it.db = db
	it.last = -1

	db.mtx.Lock()
	defer db.mtx.Unlock()

	// We need a copy of all of the keys.
	// Not the best, but probably not a bottleneck depending.
	for key, _ := range db.db {
		it.keys = append(it.keys, key)
	}
	sort.Strings(it.keys)
	return it
}

func (db *MemDB) NewBatch() Batch {
	return &memBatch{db, nil}
}

func (db *MemDB) Mutex() *sync.Mutex {
	return &(db.mtx)
}

func (db *MemDB) CacheWrap() interface{} {
	return NewCacheDB(db)
}
