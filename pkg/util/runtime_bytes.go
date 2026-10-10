// Source: src/util/hash.cpp:26-102.
package util

import

// RuntimeBytes reads actual little-endian raw words and source signed-char tails.
// The last eleven bytes are read in descending switch fallthrough order.
umemory "github.com/glycerine/g3/pkg/util/memory"

func RuntimeBytes(r *umemory.Runtime, block umemory.Block, offset uint64, n, seed uint32) (uint32, error) {
	return runtimeHashBytes(runtimeHashSource{runtime: r, block: block, offset: offset}, n, seed)
}
