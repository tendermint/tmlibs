package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func bz(s string) []byte { return []byte(s) }

func TestCacheDB(t *testing.T) {
	mem := NewMemDB()
	cdb := NewCacheDB(mem)

	require.Empty(t, cdb.Get(bz("key1")), "Expected `key1` to be empty")

	mem.Set(bz("key1"), bz("value1"))
	require.Equal(t, cdb.Get(bz("key1")), bz("value1"), "Expected `key1` to be `value1`")

	cdb.Set(bz("key1"), bz("value2"))
	require.Equal(t, cdb.Get(bz("key1")), bz("value2"), "Expected `key1` to be `value2`")
	require.Equal(t, mem.Get(bz("key1")), bz("value1"), "Expected `key1` to be `value1`")

	cdb.Commit()
	require.Equal(t, mem.Get(bz("key1")), bz("value2"), "Expected `key1` to be `value2`")

	cdb.Delete(bz("key1"))
	require.Empty(t, cdb.Get(bz("key1")), "Expected `key1` to be empty")
	require.Equal(t, mem.Get(bz("key1")), bz("value2"), "Expected `key1` to be `value2`")

	cdb.Commit()
	require.Empty(t, cdb.Get(bz("key1")), "Expected `key1` to be empty")
	require.Empty(t, mem.Get(bz("key1")), "Expected `key1` to be empty")
}
