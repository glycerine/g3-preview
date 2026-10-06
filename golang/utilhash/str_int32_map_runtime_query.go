// Source: src/util/hashtable.h:375-435,465-522; src/util/str_hashtable.h:28-30.
// Licensed under the MIT license; see LICENSE.txt.
package utilhash

import "github.com/glycerine/g3/golang/memory"

func (h *StrInt32Map) FindCStringRuntime(r *memory.Runtime, b memory.Block, offset, extent uint64) (StrInt32MapEntryRef, bool, error) {
	return h.FindCStringByteSource(BorrowRuntimeCString(r, b, offset, extent))
}
func (h *StrInt32Map) PrepareFindCStringRuntime(r *memory.Runtime, b memory.Block, offset, extent uint64) (StrInt32MapEntryRef, bool, error) {
	return h.PrepareFindCStringByteSource(BorrowRuntimeCString(r, b, offset, extent))
}
