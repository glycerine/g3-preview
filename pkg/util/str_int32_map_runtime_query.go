// Source: src/util/hashtable.h:375-435,465-522; src/util/str_hashtable.h:28-30.
// Licensed under the MIT license; see LICENSE.txt.
package util

func (h *StrInt32Map) FindCStringRuntime(r *Runtime, b Block, offset, extent uint64) (StrInt32MapEntryRef, bool, error) {
	return h.FindCStringByteSource(BorrowRuntimeCString(r, b, offset, extent))
}
func (h *StrInt32Map) PrepareFindCStringRuntime(r *Runtime, b Block, offset, extent uint64) (StrInt32MapEntryRef, bool, error) {
	return h.PrepareFindCStringByteSource(BorrowRuntimeCString(r, b, offset, extent))
}
