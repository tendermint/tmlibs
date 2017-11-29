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

package db_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/tendermint/tmlibs/common"
	"github.com/tendermint/tmlibs/db"
)

func setupBadgerDB(t *testing.T) (d db.DB, tearDown func()) {
	testDir, err := ioutil.TempDir("", "throwaway-db-tests")
	require.Nil(t, err, "expecting a nil error")
	ddb := db.NewDB("test_db", db.BadgerDBBackendStr, testDir)
	tearDown = func() {
		ddb.Close()
		os.RemoveAll(testDir)
	}
	return ddb, tearDown
}

func TestBadgerDB(t *testing.T) {
	t.Parallel()

	ddb, tearDown := setupBadgerDB(t)
	defer tearDown()

	switch ddb.(type) {
	case *db.BadgerDB:
	default:
		t.Fatalf("expected a proper DB initialization")
	}
}

func TestBadgerDBSetGet(t *testing.T) {
	t.Parallel()

	ddb, tearDown := setupBadgerDB(t)
	defer tearDown()

	tests := []struct {
		key, value []byte
	}{
		{key: []byte("foo"), value: []byte("bar")},
		{key: []byte("                                          "), value: []byte("")},
		{key: []byte(""), value: []byte("                                          ")},
		{key: []byte(" "), value: []byte("a                                         2")},
		{key: common.RandBytes(1024), value: []byte("right here ")},
	}

	for _, tt := range tests {
		ddb.SetSync(tt.key, tt.value)
	}

	for i, tt := range tests {
		if got, want := ddb.Get(tt.key), tt.value; !bytes.Equal(got, want) {
			t.Errorf("#%d: Get mismatch\ngot: %x\nwant:%x", i, got, want)
		}
	}

	// Purge them all
	for _, tt := range tests {
		ddb.DeleteSync(tt.key)
	}

	for i, tt := range tests {
		if got := ddb.Get(tt.key); !bytes.Equal(got, nil) {
			t.Errorf("#%d: Get mismatch\ngot: %x\nwant:nil", i, got)
		}
	}
}

func TestBadgerDBFprint(t *testing.T) {
	t.Parallel()

	adb, tearDown := setupBadgerDB(t)
	defer tearDown()

	ddb := adb.(*db.BadgerDB)

	// Initially no data
	buf := new(bytes.Buffer)
	ddb.Fprint(buf)
	require.Equal(t, 0, len(buf.Bytes()), "unexpected data here: %s", buf.Bytes())

	tests := []struct {
		key, value []byte
	}{
		{key: []byte("foo"), value: []byte("bar")},
		{key: []byte("                                          "), value: []byte("")},
		{key: []byte(" "), value: []byte("a                                         2")},
		{key: common.RandBytes(1024), value: []byte("right here ")},
		{key: nil, value: common.RandBytes(1024)},
		{key: []byte("p"), value: []byte("                                          ")},
		{key: common.RandBytes(511), value: nil},
	}

	batch := ddb.NewBatch()
	for _, tt := range tests {
		batch.Set(tt.key, tt.value)
	}
	batch.Write()

	ddb.Fprint(buf)

	// To ensure that all the entries match, nothing less, nothing more
	// We'll iterate through the DB, printing out entries to a log
	// and afterwards go through the expected entries, trimming
	// out existent entries and at the end the remaining log should
	// be entirely empty

	rawLog := buf.Bytes()
	require.NotEqual(t, 0, len(rawLog), "expecting log entries")

	for i, tt := range tests {
		require.NotEqual(t, 0, len(rawLog), "#%d log unexpectedly is of length 0", i)

		wantKey, wantValue := tt.key, tt.value
		wantEntry := []byte(fmt.Sprintf("[%X]:\t[%X]\n", wantKey, wantValue))
		index := bytes.Index(rawLog, wantEntry)
		if index < 0 {
			t.Errorf("#%d: could not find entry %q. current log: %q", i, wantEntry, rawLog)
			continue
		}

		// Otherwise now prune out that entry to trim the log
		rawLog = bytes.Join([][]byte{rawLog[:index], rawLog[index+len(wantEntry):]}, []byte(""))
	}
}

func TestBadgerDBBatchSet(t *testing.T) {
	t.Parallel()

	ddb, tearDown := setupBadgerDB(t)
	defer tearDown()

	tests := []struct {
		key, value []byte
	}{
		{key: []byte("foo"), value: []byte("bar")},
		{key: []byte("                                          "), value: []byte("")},
		{key: []byte(" "), value: []byte("a                                         2")},
		{key: common.RandBytes(1024), value: []byte("right here ")},
		{key: nil, value: common.RandBytes(1024)},
		{key: []byte("p"), value: []byte("                                          ")},
		{key: common.RandBytes(511), value: nil},
	}

	batch := ddb.NewBatch()

	// Test concurrent batch sets and
	// ensure that we don't have an races.
	var wg sync.WaitGroup
	for _, tt := range tests {
		wg.Add(1)
		go func(key, value []byte) {
			defer wg.Done()
			batch.Set(key, value)
		}(tt.key, tt.value)
	}
	wg.Wait()

	// Unfortunately the maintained BadgerDB v1.0 as of Sun  5 Nov 2017
	// doesn't have a "Batch" hence we can't test that writes
	// haven't yet been committed to disk
	batch.Write()

	// Now ensure that we wrote the data
	for i, tt := range tests {
		if got, want := ddb.Get(tt.key), tt.value; !bytes.Equal(got, want) {
			t.Errorf("#%d: Get mismatch\ngot: %x\nwant:%x", i, got, want)
		}
	}
}

func TestBadgerDBIterator(t *testing.T) {
	t.Parallel()

	ddb, tearDown := setupBadgerDB(t)
	defer tearDown()

	tests := []struct {
		key, value []byte
	}{
		{key: []byte("foo"), value: []byte("bar")},
		{key: []byte("                                          "), value: []byte("")},
		{key: []byte(" "), value: []byte("a                                         2")},
		{key: common.RandBytes(1024), value: []byte("right here ")},
		{key: nil, value: common.RandBytes(1024)},
		{key: []byte("p"), value: []byte("                                          ")},
		{key: common.RandBytes(511), value: nil},
	}

	batch := ddb.NewBatch()
	for _, tt := range tests {
		batch.Set(tt.key, tt.value)
	}
	batch.Write()

	// To ensure that all the entries match, nothing less, nothing more
	// We'll iterate through the DB, printing out entries to a log
	// and afterwards go through the expected entries, trimming
	// out existent entries and at the end the remaining log should
	// be entirely empty
	iter := ddb.Iterator()
	resultLog := new(bytes.Buffer)
	for {
		gotKey, gotValue := iter.Key(), iter.Value()
		fmt.Fprintf(resultLog, "%x:%x\n", gotKey, gotValue)
		if !iter.Next() {
			break
		}
	}

	rawLog := resultLog.Bytes()
	require.NotEqual(t, 0, len(rawLog), "expecting log entries")

	for i, tt := range tests {
		require.NotEqual(t, 0, len(rawLog), "#%d log unexpectedly is of length 0", i)

		wantKey, wantValue := tt.key, tt.value
		wantEntry := []byte(fmt.Sprintf("%x:%x\n", wantKey, wantValue))
		index := bytes.Index(rawLog, wantEntry)
		if index < 0 {
			t.Errorf("#%d: could not find entry %q. current log: %q", i, wantEntry, rawLog)
			continue
		}

		// Otherwise now prune out that entry to trim the log
		rawLog = bytes.Join([][]byte{rawLog[:index], rawLog[index+len(wantEntry):]}, []byte(""))
	}
	require.Equal(t, 0, len(rawLog), "expected all log entries to have been matched and trimmed out")
}

func BenchmarkSetBadgerDB(b *testing.B) {
	ddb, tearDown := setupBadgerDB(nil)
	defer tearDown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tt := range benchmarkData {
			ddb.SetSync(tt.key, tt.value)
		}
	}
	b.ReportAllocs()
}

func BenchmarkSetSyncBadgerDB(b *testing.B) {
	ddb, tearDown := setupBadgerDB(nil)
	defer tearDown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tt := range benchmarkData {
			ddb.SetSync(tt.key, tt.value)
		}
	}
	b.ReportAllocs()
}

func BenchmarkBatchSetBadgerDB(b *testing.B) {
	ddb, tearDown := setupBadgerDB(nil)
	defer tearDown()

	benchmarkBatchSet(b, ddb)
}

func BenchmarkSetDeleteBadgerDB(b *testing.B) {
	ddb, tearDown := setupBadgerDB(nil)
	defer tearDown()

	benchmarkSetDelete(b, ddb)
}

func BenchmarkRandomReadsWritesBadgerDB(b *testing.B) {
	db, tearDown := setupBadgerDB(nil)
	defer tearDown()

	benchmarkRandomReadsWritesOnDB(b, db)
}

func BenchmarkBatchSetDeleteBadgerDB(b *testing.B) {
	db, tearDown := setupBadgerDB(nil)
	defer tearDown()

	benchmarkBatchSetDelete(b, db)
}
