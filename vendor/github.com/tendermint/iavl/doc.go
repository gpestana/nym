// Package iavl implements a versioned, snapshottable (immutable) AVL+ tree
// for persisting key-value pairs.
//
//
// Basic usage of MutableTree.
//
//  import "github.com/tendermint/iavl"
//  import "github.com/tendermint/tm-cmn/db"
//  ...
//
//  tree := iavl.NewMutableTree(db.NewMemDB(), 128)
//
//  tree.IsEmpty() // true
//
//  tree.Set([]byte("alice"), []byte("abc"))
//  tree.SaveVersion(1)
//
//  tree.Set([]byte("alice"), []byte("xyz"))
//  tree.Set([]byte("bob"), []byte("xyz"))
//  tree.SaveVersion(2)
//
//  tree.LatestVersion() // 2
//
//  tree.GetVersioned([]byte("alice"), 1) // "abc"
//  tree.GetVersioned([]byte("alice"), 2) // "xyz"
//
// Proof of existence:
//
//  root := tree.Hash()
//  val, proof, err := tree.GetVersionedWithProof([]byte("bob"), 2) // "xyz", RangeProof, nil
//  proof.Verify([]byte("bob"), val, root) // nil
//
// Proof of absence:
//
//  _, proof, err = tree.GetVersionedWithProof([]byte("tom"), 2) // nil, RangeProof, nil
//  proof.Verify([]byte("tom"), nil, root) // nil
//
// Now we delete an old version:
//
//  tree.DeleteVersion(1)
//  tree.VersionExists(1) // false
//  tree.Get([]byte("alice")) // "xyz"
//  tree.GetVersioned([]byte("alice"), 1) // nil
//
// Can't create a proof of absence for a version we no longer have:
//
//  _, proof, err = tree.GetVersionedWithProof([]byte("tom"), 1) // nil, nil, error
//
package iavl
