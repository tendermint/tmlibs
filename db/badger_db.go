// Copyright 2017 Tendermint. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// BadgerDB enables us the use of BadgerDB a fast key value store in pure Go.
// Typical usage follows:
//  db, err := db.NewBadgerDB("badger", targetDir)
//  if err != nil {
//	log.Fatalf("badgerDB initialization: %v", err)
//  }
//  defer db.Close()
//
//  For the asynchronous set
//  db.Set([]byte("foo"), []byte("bar"))
//  db.SetSync([]byte("true"), []byte("tendermint"))
//  db.Delete([]byte("true"))
//  db.Deletesync([]byte("true"))
//  iter := db.Iterator()
//  for {
//     key, value := iter.Key(), iter.Value()
//     fmt.Printf("key: %v value: %v\n", key, value)
//  }
//
// For better performance, perform Sets/Writes in batches
//  batch := db.NewBatch()
//  for i := 0; i < 100; i++ {
//	batch.Set(cmn.RandBytes(100), []byte("true!!!"))
//  }
//  batch.Write()

package db

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/dgraph-io/badger"

	"github.com/tendermint/tmlibs/common"
)

func init() {
	registerDBCreator(BadgerDBBackendStr, badgerDBCreator, true)
}

type Options badger.Options

func badgerDBCreator(dbName, dir string) (DB, error) {
	return NewBadgerDB(dbName, dir)
}

var (
	_KB = int64(1024)
	_MB = 1024 * _KB
	_GB = 1024 * _MB
)

// NewBadgerDB creates a Badger key-value store backed to the
// directory dir supplied. If dir does not exist, we create it.
func NewBadgerDB(dbName, dir string) (*BadgerDB, error) {
	// BadgerDB doesn't expose a way for us to
	// create  a DB with the user's supplied name.
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	opts := new(badger.Options)
	*opts = badger.DefaultOptions
	// Arbitrary size given that at Tendermint
	// we'll need huge KeyValue stores.
	opts.ValueLogFileSize = 1 * _GB
	opts.Dir = dir
	opts.ValueDir = dir

	return NewBadgerDBWithOptions((*Options)(opts))
}

// NewBadgerDBWithOptions creates a BadgerDB key value store
// gives the flexibility of initializing a database with the
// respective options.
func NewBadgerDBWithOptions(opts *Options) (*BadgerDB, error) {
	kv, err := badger.Open(*(*badger.Options)(opts))
	if err != nil {
		return nil, err
	}
	return &BadgerDB{kv: kv}, nil
}

type BadgerDB struct {
	kv *badger.DB

	// closeOnce ensure that Close is invoked
	// only once since BadgerDB uses a channel
	// that when already closed will panic
	// yet serving no other purpose to the DB.
	closeOnce sync.Once
}

var _ DB = (*BadgerDB)(nil)

const (
	readOnly  = false
	readWrite = true
)

func (b *BadgerDB) Get(key []byte) []byte {
	tx := b.kv.NewTransaction(readOnly)
	defer tx.Discard()

	kvItem, err := tx.Get(key)
	if err != nil {
		switch err {
		case badger.ErrKeyNotFound:
			return nil
		default:
			// Unfortunate that Get can't return errors
			// TODO: Propose allowing DB's Get to return errors too.
			common.PanicCrisis(err)
		}
	}
	value, err := kvItem.Value()
	if err != nil {
		// TODO: ditto:: Propose allowing DB's Get to return errors too.
		common.PanicCrisis(err)
	}
	return value
}

func (b *BadgerDB) Set(key, value []byte) {
	b.kv.Update(func(tx *badger.Txn) error {
		return tx.Set(key, value)
	})
}

func (b *BadgerDB) SetSync(key, value []byte) {
	b.Set(key, value)
}

func (b *BadgerDB) Delete(key []byte) {
	b.kv.Update(func(tx *badger.Txn) error {
		return tx.Delete(key)
	})
}

func (b *BadgerDB) DeleteSync(key []byte) {
	b.kv.Update(func(tx *badger.Txn) error {
		return tx.Delete(key)
	})
}

func (b *BadgerDB) Close() {
	b.closeOnce.Do(func() {
		if err := b.kv.Close(); err != nil {
			common.PanicCrisis(err)
		}
	})
}

func (b *BadgerDB) Fprint(w io.Writer) {
	bIter := b.Iterator().(*badgerDBIterator)
	defer bIter.Release()

	var bw *bufio.Writer
	if bbw, ok := w.(*bufio.Writer); ok {
		bw = bbw
	} else {
		bw = bufio.NewWriter(w)
	}
	defer bw.Flush()

	i := uint64(0)
	for bIter.rewind(); bIter.valid(); bIter.Next() {
		k, v := bIter.kv()
		fmt.Fprintf(bw, "[%X]:\t[%X]\n", k, v)
		i += 1
		if i%1024 == 0 {
			bw.Flush()
			i = 0
		}
	}
}

func (b *BadgerDB) Print() {
	bw := bufio.NewWriter(os.Stdout)
	b.Fprint(bw)
}

func (b *BadgerDB) Iterator() Iterator {
	tx := b.kv.NewTransaction(readOnly)
	dbIter := tx.NewIterator(badger.IteratorOptions{
		PrefetchValues: true,

		// Arbitrary PrefetchSize
		PrefetchSize: 10,
	})
	// Ensure that we are always at the zeroth item
	dbIter.Rewind()
	return &badgerDBIterator{iter: dbIter}
}

func (b *BadgerDB) IteratorPrefix(prefix []byte) Iterator {
	tx := b.kv.NewTransaction(readOnly)
	dbIter := tx.NewIterator(badger.IteratorOptions{
		PrefetchValues: true,

		// Arbitrary PrefetchSize
		PrefetchSize: 10,
	})
	// Ensure that we are always at the zeroth item
	dbIter.Rewind()
	return &badgerDBIterator{iter: dbIter}
}

func (b *BadgerDB) Stats() map[string]string {
	return nil
}

func (b *BadgerDB) NewBatch() Batch {
	return &badgerDBBatch{db: b.kv}
}

var _ Batch = (*badgerDBBatch)(nil)

type badgerDBBatch struct {
	txnMu sync.Mutex
	txn_  *badger.Txn

	db *badger.DB
}

func (bb *badgerDBBatch) txn() *badger.Txn {
	bb.txnMu.Lock()
	if bb.txn_ == nil {
		bb.txn_ = bb.db.NewTransaction(true)
	}
	bb.txnMu.Unlock()
	return bb.txn_
}

func (bb *badgerDBBatch) Set(key, value []byte) {
	bb.txn().Set(key, value)
}

func (bb *badgerDBBatch) Delete(key []byte) {
	bb.txn().Delete(key)
}

// Write commits all batch sets to the DB
func (bb *badgerDBBatch) Write() {
	bb.txnMu.Lock()
	txn := bb.txn_
	bb.txn_ = nil
	bb.txnMu.Unlock()

	txn.Commit(func(err error) {
		if err != nil {
			common.PanicCrisis(err)
		}
	})
}

type badgerDBIterator struct {
	mu sync.RWMutex

	lastErr error

	iter *badger.Iterator
}

var _ Iterator = (*badgerDBIterator)(nil)

func (bi *badgerDBIterator) Next() bool {
	bi.mu.Lock()
	defer bi.mu.Unlock()

	if !bi.iter.Valid() {
		return false
	}
	bi.iter.Next()
	return bi.iter.Valid()
}

func (bi *badgerDBIterator) Key() []byte {
	bi.mu.Lock()
	defer bi.mu.Unlock()

	return bi.iter.Item().Key()
}

func (bi *badgerDBIterator) Value() []byte {
	bi.mu.Lock()
	defer bi.mu.Unlock()

	bItem := bi.iter.Item()
	if bItem == nil {
		return nil
	}
	value, err := bItem.Value()
	if err != nil {
		bi.setLastError(err)
	}
	return value
}

func (bi *badgerDBIterator) kv() (key, value []byte) {
	bItem := bi.iter.Item()
	if bItem == nil {
		return nil, nil
	}
	key = bItem.Key()
	value, err := bItem.Value()
	if err != nil {
		bi.setLastError(err)
	}
	return key, value
}

func (bi *badgerDBIterator) Error() error {
	bi.mu.RLock()
	err := bi.lastErr
	bi.mu.RUnlock()
	return err
}

func (bi *badgerDBIterator) setLastError(err error) {
	bi.mu.Lock()
	bi.lastErr = err
	bi.mu.Unlock()
}

func (bi *badgerDBIterator) Release() {
	bi.mu.Lock()
	bi.iter.Close()
	bi.mu.Unlock()
}

func (bi *badgerDBIterator) rewind()     { bi.iter.Rewind() }
func (bi *badgerDBIterator) valid() bool { return bi.iter.Valid() }
