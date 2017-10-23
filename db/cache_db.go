package db

import (
	"fmt"
)

// NOTE: Needed by MultiStore.CacheWrap().
type CacheDB struct {
	cache  *MemDB
	parent DB
}

func NewCacheDB(parent DB) *CacheDB {
	db := &CacheDB{
		cache:  NewMemDB(),
		parent: parent,
	}
	return db
}

func (db *CacheDB) Get(key []byte) []byte {
	value, ok := db.cache.GetOK(key)
	if ok {
		return value
	}
	return db.parent.Get(key)
}

func (db *CacheDB) Set(key []byte, value []byte) {
	db.cache.Set(key, value)
}

func (db *CacheDB) SetSync(key []byte, value []byte) {
	db.cache.Set(key, value)
}

func (db *CacheDB) Delete(key []byte) {
	db.cache.Delete(key)
	db.parent.Delete(key)
}

func (db *CacheDB) DeleteSync(key []byte) {
	db.cache.DeleteSync(key)
	db.parent.DeleteSync(key)
}

func (db *CacheDB) Close() {
	db.cache.Close()
	db.parent.Close()
}

func (db *CacheDB) Print() {
	fmt.Println("CacheDB\ncache:\n")
	db.cache.Print()
	fmt.Println("\nparent:\n")
	db.parent.Print()
}

func (db *CacheDB) Stats() map[string]string {
	stats := make(map[string]string)
	mergeStats(db.cache.Stats(), stats, "cache.")
	mergeStats(db.parent.Stats(), stats, "parent.")
	return stats
}

func (db *CacheDB) Iterator() Iterator {
	panic("CacheDB.Iterator() not yet supported")
}

func (db *CacheDB) IteratorPrefix(prefix []byte) Iterator {
	panic("CacheDB.IteratorPrefix() not yet supported")
}

func (db *CacheDB) NewBatch() Batch {
	return &memDBBatch{db.cache, nil}
}
