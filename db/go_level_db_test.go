package db_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	cmn "github.com/tendermint/tmlibs/common"
	"github.com/tendermint/tmlibs/db"
)

func setupGoLevelDB() (db.DB, func() error, error) {
	dir := "golevel_db_test"
	dirPrefix := filepath.Join(dir, fmt.Sprintf("%x", cmn.RandStr(12)))
	db, err := db.NewGoLevelDB(dirPrefix, "")
	if err != nil {
		return nil, nil, err
	}
	tearDown := func() error { return os.RemoveAll(dir) }
	return db, tearDown, nil
}

func BenchmarkRandomReadsWritesGoLevelDB(b *testing.B) {
	db, tearDown, err := setupGoLevelDB()
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		db.Close()
		tearDown()
	}()

	benchmarkRandomReadsWritesOnDB(b, db)
}

func BenchmarkSetDeleteGoLevelDB(b *testing.B) {
	ddb, tearDown, err := setupGoLevelDB()
	if err != nil {
		b.Fatal(err)
	}
	defer tearDown()

	benchmarkSetDelete(b, ddb)
}

func BenchmarkBatchSetDeleteGoLevelDB(b *testing.B) {
	ddb, tearDown, err := setupGoLevelDB()
	if err != nil {
		b.Fatal(err)
	}
	defer tearDown()

	benchmarkBatchSetDelete(b, ddb)
}

func BenchmarkBatchSetGoLevelDB(b *testing.B) {
	ddb, tearDown, err := setupGoLevelDB()
	if err != nil {
		b.Fatal(err)
	}
	defer tearDown()

	benchmarkBatchSet(b, ddb)
}
