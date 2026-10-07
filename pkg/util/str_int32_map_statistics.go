// Concrete source specialization: src/util/map.h:24-196; src/util/hashtable.h:44-63.
// Source: src/util/hashtable.h:33-36,138-140,254-256,267-269,281-283,359-373,409-424,481-491,533-543,649-653.
// Source author: Leonardo de Moura (2006-09-11).
// Licensed under the MIT license; see LICENSE.txt.
package util

func (h *StrInt32Map) runtimeObjectExtent() uint64 {
	if h.config.Statistics {
		return 32
	}
	return 24
}
func (h *StrInt32Map) readCollision() (uint64, error) {
	if h.config.Runtime == nil {
		if h.raw.initialized&0xC0 != 0xC0 {
			return 0, ErrRuntimeUninitialized
		}
		return uint64(h.raw.inlineStatistics[0]) | uint64(h.raw.inlineStatistics[1])<<32, nil
	}
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
func (h *StrInt32Map) writeCollision(v uint64) error {
	if h.config.Runtime == nil {
		h.raw.inlineStatistics[0] = uint32(v)
		h.raw.inlineStatistics[1] = uint32(v >> 32)
		h.raw.initialized |= 0xC0
		return nil
	}
	if e := h.writeRuntimeWord(24, uint32(v)); e != nil {
		return e
	}
	return h.writeRuntimeWord(28, uint32(v>>32))
}
func (h *StrInt32Map) incrementCollision() error {
	if !h.config.Statistics {
		return nil
	}
	v, e := h.readCollision()
	if e != nil {
		return e
	}
	return h.writeCollision(v + 1)
}
