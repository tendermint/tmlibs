package db

import "testing"

func TestIteratePrefix(t *testing.T) {
	db := NewMemDB()
	// Under "key" prefix
	db.Set(bz("key"), bz("value"))
	db.Set(bz("key1"), bz("value1"))
	db.Set(bz("key2"), bz("value2"))
	db.Set(bz("key3"), bz("value3"))
	db.Set(bz("something"), bz("else"))
	db.Set(bz(""), bz(""))
	db.Set(bz("k"), bz("val"))
	db.Set(bz("ke"), bz("valu"))
	db.Set(bz("kee"), bz("valuu"))
	xitr := db.Iterator(nil, nil)
	xitr.Key()

	pdb := NewPrefixDB(db, bz("key"))
	checkValue(t, pdb.db, bz("key"), bz("value"))
	checkValue(t, pdb.db, bz("key1"), bz("value1"))
	checkValue(t, pdb.db, bz("key2"), bz("value2"))
	checkValue(t, pdb.db, bz("key3"), bz("value3"))
	checkValue(t, pdb.db, bz("something"), bz("else"))
	checkValue(t, pdb.db, bz(""), bz(""))
	checkValue(t, pdb.db, bz("k"), bz("val"))
	checkValue(t, pdb.db, bz("ke"), bz("valu"))
	checkValue(t, pdb.db, bz("kee"), bz("valuu"))

	itr := pdb.Iterator(nil, nil)
	itr.Key()
	checkItem(t, itr, bz(""), bz("value"))
	checkNext(t, itr, true)
	checkItem(t, itr, bz("1"), bz("value1"))
	checkNext(t, itr, true)
	checkItem(t, itr, bz("2"), bz("value2"))
	checkNext(t, itr, true)
	checkItem(t, itr, bz("3"), bz("value3"))
	itr.Close()
}
