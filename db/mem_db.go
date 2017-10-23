package db

import (
	"fmt"
	"strings"
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
	db.db[string(key)] = value
}

func (db *MemDB) SetSync(key []byte, value []byte) {
	db.mtx.Lock()
	defer db.mtx.Unlock()
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
	return db.IteratorPrefix([]byte{})
}

func (db *MemDB) IteratorPrefix(prefix []byte) Iterator {
	it := newMemDBIterator()
	it.db = db
	it.last = -1

	db.mtx.Lock()
	defer db.mtx.Unlock()

	// unfortunately we need a copy of all of the keys
	for key, _ := range db.db {
		if strings.HasPrefix(key, string(prefix)) {
			it.keys = append(it.keys, key)
		}
	}
	return it
}

func (db *MemDB) NewBatch() Batch {
	return &memDBBatch{db, nil}
}
