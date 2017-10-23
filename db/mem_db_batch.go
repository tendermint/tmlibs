package db

type memDBBatch struct {
	db  *MemDB
	ops []operation
}

type opType int

const (
	opTypeSet    = 1
	opTypeDelete = 2
)

type operation struct {
	opType
	key   []byte
	value []byte
}

func (mBatch *memDBBatch) Set(key, value []byte) {
	mBatch.ops = append(mBatch.ops, operation{opTypeSet, key, value})
}

func (mBatch *memDBBatch) Delete(key []byte) {
	mBatch.ops = append(mBatch.ops, operation{opTypeDelete, key, nil})
}

func (mBatch *memDBBatch) Write() {
	mBatch.db.mtx.Lock()
	defer mBatch.db.mtx.Unlock()

	for _, op := range mBatch.ops {
		if op.opType == opTypeSet {
			mBatch.db.db[string(op.key)] = op.value
		} else if op.opType == opTypeDelete {
			delete(mBatch.db.db, string(op.key))
		}
	}

}
