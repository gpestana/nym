// +build gcc

// This file exists because some of the DBs e.g CLevelDB
// require gcc as the compiler before they can ran otherwise
// we'll encounter crashes such as in https://github.com/tendermint/merkleeyes/issues/39

package iavl

import (
	"testing"

	"github.com/tendermint/tm-cmn/db"
)

func BenchmarkImmutableAvlTreeCLevelDB(b *testing.B) {
	db := db.NewDB("test", db.CLevelDBBackendStr, "./")
	benchmarkImmutableAvlTreeWithDB(b, db)
}
