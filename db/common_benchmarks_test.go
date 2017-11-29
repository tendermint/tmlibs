package db_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	cmn "github.com/tendermint/tmlibs/common"
	"github.com/tendermint/tmlibs/db"
)

var benchmarkData = []struct {
	key, value []byte
}{
	{cmn.RandBytes(100), nil},
	{cmn.RandBytes(1000), []byte("foo")},
	{[]byte("foo"), cmn.RandBytes(1000)},
	{[]byte(""), cmn.RandBytes(1000)},
	{nil, cmn.RandBytes(1000000)},
	{cmn.RandBytes(100000), nil},
	{cmn.RandBytes(1000000), nil},
}

func benchmarkRandomReadsWritesOnDB(b *testing.B, db db.DB) {
	numItems := int64(1000000)
	internal := map[int64]int64{}
	b.StopTimer()
	for i := 0; i < int(numItems); i++ {
		internal[int64(i)] = int64(0)
	}

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		// Write something
		{
			idx := (int64(cmn.RandInt()) % numItems)
			internal[idx] += 1
			val := internal[idx]
			idxBytes := int642Bytes(int64(idx))
			valBytes := int642Bytes(int64(val))
			db.SetSync(idxBytes, valBytes)
		}

		// Read something
		{
			idx := (int64(cmn.RandInt()) % numItems)
			val := internal[idx]
			idxBytes := int642Bytes(int64(idx))
			valBytes := db.Get(idxBytes)
			if val == 0 {
				if !bytes.Equal(valBytes, nil) {
					b.Errorf("Expected %v for %v, got %X",
						nil, idx, valBytes)
					break
				}
			} else {
				if len(valBytes) != 8 {
					b.Errorf("Expected length 8 for %v, got %X",
						idx, valBytes)
					break
				}
				valGot := bytes2Int64(valBytes)
				if val != valGot {
					b.Errorf("Expected %v for %v, got %v",
						val, idx, valGot)
					break
				}
			}
		}
	}
	b.ReportAllocs()
}

func benchmarkSetDelete(b *testing.B, db db.DB) {
	b.Skip("Until https://github.com/dgraph-io/badger/issues/308 is resolved, this benchmark is invalid")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tt := range benchmarkData {
			db.SetSync(tt.key, tt.value)
		}
		for _, tt := range benchmarkData {
			db.Delete(tt.key)
		}
	}
	b.ReportAllocs()
}

func benchmarkBatchSetDelete(b *testing.B, db db.DB) {
	b.ResetTimer()
	batch := db.NewBatch()
	for i := 0; i < b.N; i++ {
		for _, tt := range benchmarkData {
			batch.Set(tt.key, tt.value)
		}
		batch.Write()
		for _, tt := range benchmarkData {
			batch.Delete(tt.key)
		}
	}
	b.ReportAllocs()
}

func benchmarkBatchSet(b *testing.B, db db.DB) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		batch := db.NewBatch()
		for _, tt := range benchmarkData {
			batch.Set(tt.key, tt.value)
		}
		batch.Write()
	}
	b.ReportAllocs()
}

func int642Bytes(i int64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(i))
	return buf
}

func bytes2Int64(buf []byte) int64 {
	return int64(binary.BigEndian.Uint64(buf))
}
