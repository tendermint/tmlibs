package db

import (
	"bytes"
)

// IteratePrefix is a convenience function for iterating over a key domain
// restricted by prefix.
func IteratePrefix(db DB, prefix []byte) Iterator {
	var start, end []byte
	if len(prefix) == 0 {
		start = nil
		end = nil
	} else {
		start = cp(prefix)
		end = cpIncr(prefix)
	}
	return db.Iterator(start, end)
}

/*
TODO: Make test, maybe rename.
// Like IteratePrefix but the iterator strips the prefix from the keys.
func IteratePrefixStripped(db DB, prefix []byte) Iterator {
	return newUnprefixIterator(prefix, IteratePrefix(db, prefix))
}
*/

//----------------------------------------
// prefixDB

type prefixDB struct {
	prefix []byte
	db     DB
}

// NewPrefixDB lets you namespace multiple DBs within a single DB.
func NewPrefixDB(db DB, prefix []byte) prefixDB {
	return prefixDB{
		prefix: prefix,
		db:     db,
	}
}

func (pdb prefixDB) Get(key []byte) []byte {
	return pdb.db.Get(pdb.prefixed(key))
}

func (pdb prefixDB) Has(key []byte) bool {
	return pdb.db.Has(pdb.prefixed(key))
}

func (pdb prefixDB) Set(key []byte, value []byte) {
	pdb.db.Set(pdb.prefixed(key), value)
}

func (pdb prefixDB) SetSync(key []byte, value []byte) {
	pdb.db.SetSync(pdb.prefixed(key), value)
}

func (pdb prefixDB) Delete(key []byte) {
	pdb.db.Delete(pdb.prefixed(key))
}

func (pdb prefixDB) DeleteSync(key []byte) {
	pdb.db.DeleteSync(pdb.prefixed(key))
}

func (pdb prefixDB) Iterator(start, end []byte) Iterator {
	pstart := append([]byte(pdb.prefix), start...)
	pend := []byte(nil)
	if end != nil {
		pend = append([]byte(pdb.prefix), end...)
	}
	return newUnprefixIterator(
		pdb.prefix,
		pdb.db.Iterator(
			pstart,
			pend,
		),
	)
}

func (pdb prefixDB) prefixed(key []byte) []byte {
	return append([]byte(pdb.prefix), key...)
}

//----------------------------------------

// Strips prefix while iterating from Iterator.
type unprefixIterator struct {
	prefix []byte
	source Iterator
}

func newUnprefixIterator(prefix []byte, source Iterator) unprefixIterator {
	return unprefixIterator{
		prefix: prefix,
		source: source,
	}
}

func (iter unprefixIterator) Domain() (start []byte, end []byte) {
	start, end = iter.source.Domain()
	if len(start) > 0 {
		start = stripPrefix(start, iter.prefix)
	}
	if len(end) > 0 {
		end = stripPrefix(end, iter.prefix)
	}
	return
}

func (iter unprefixIterator) Valid() bool {
	return iter.source.Valid()
}

func (iter unprefixIterator) Next() {
	iter.source.Next()
}

func (iter unprefixIterator) Key() (key []byte) {
	return stripPrefix(iter.source.Key(), iter.prefix)
}

func (iter unprefixIterator) Value() (value []byte) {
	return iter.source.Value()
}

func (iter unprefixIterator) Close() {
	iter.source.Close()
}

//----------------------------------------

func stripPrefix(key []byte, prefix []byte) (stripped []byte) {
	if len(key) < len(prefix) {
		panic("should not happen")
	}
	if !bytes.Equal(key[:len(prefix)], prefix) {
		panic("should not happne")
	}
	return key[len(prefix):]
}
