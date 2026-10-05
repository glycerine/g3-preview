// Source: src/util/hashtable.h:33-36,138-140,254-256,267-269,281-283,359-373,409-424,481-491,533-543,649-653.
// Source author: Leonardo de Moura (2006-09-11).
// Licensed under the MIT license; see LICENSE.txt.
package utilhash

func (h *StrHashtable) runtimeObjectExtent() uint64 {
	if h.config.Statistics {
		return 32
	}
	return 24
}
func (h *StrHashtable) readCollision() (uint64, error) {
	lo, e := h.runtimeWord(24)
	if e != nil {
		return 0, e
	}
	hi, e := h.runtimeWord(28)
	if e != nil {
		return 0, e
	}
	return uint64(lo) | uint64(hi)<<32, nil
}
func (h *StrHashtable) writeCollision(v uint64) error {
	if e := h.writeRuntimeWord(24, uint32(v)); e != nil {
		return e
	}
	return h.writeRuntimeWord(28, uint32(v>>32))
}
func (h *StrHashtable) incrementCollision() error {
	if !h.config.Statistics {
		return nil
	}
	v, e := h.readCollision()
	if e != nil {
		return e
	}
	return h.writeCollision(v + 1)
}
