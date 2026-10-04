// Source: src/util/hash.cpp:26-102.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2006-09-11.
package utilhash

import "github.com/glycerine/g3/golang/memory"

// RuntimeBytes reads actual little-endian raw words and source signed-char tails.
// The last eleven bytes are read in descending switch fallthrough order.
func RuntimeBytes(r *memory.Runtime, block memory.Block, offset uint64, n, seed uint32) (uint32, error) {
	return runtimeHashBytes(runtimeHashSource{runtime: r, block: block, offset: offset}, n, seed)
}
