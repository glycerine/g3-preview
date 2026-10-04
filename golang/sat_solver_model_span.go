// Source: src/sat/sat_types.h:138; src/util/lbool.h; src/util/vector.h:407–420; src/sat/sat_model_converter.cpp:35–114.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2011.
package g3

import "github.com/glycerine/g3/golang/arena"

// Concrete model& borrow. Runtime reads resolve the current member backing at
// every operator[]; the preexisting caller-owned slice route is unchanged.
// The source lbool enum is signed32, unlike the solver's char assignments.
type satSolverModelSpan struct {
	slice  satModel
	column *satSolverRuntimeUintColumn
}

func (m satSolverModelSpan) read(v satBoolVar) (lbool, error) {
	if m.column != nil {
		x, e := m.column.read(uint32(v))
		return lbool(int32(x)), e
	}
	if uint64(v) >= uint64(len(m.slice)) {
		return lUndef, arena.ErrBounds
	}
	return m.slice[v], nil
}
func (m satSolverModelSpan) write(v satBoolVar, x lbool) error {
	if m.column != nil {
		return m.column.write(uint32(v), uint32(int32(x)))
	}
	if uint64(v) >= uint64(len(m.slice)) {
		return arena.ErrBounds
	}
	m.slice[v] = x
	return nil
}
func (m satSolverModelSpan) value(l satLiteral) (lbool, error) {
	v, e := m.read(l.variable())
	if e != nil {
		return lUndef, e
	}
	if l.sign() {
		v = v.negated()
	}
	return v, nil
}
func satSolverResizeModel(c *satSolverRuntimeUintColumn, n uint32) error {
	old, e := c.size()
	if e != nil {
		return e
	}
	if n <= old {
		return c.shrink(n)
	}
	for {
		cap, e := c.capacity()
		if e != nil {
			return e
		}
		if n <= cap {
			break
		}
		if e = c.raw.expandStride(4); e != nil {
			return e
		}
	}
	// Source size is published before ascending placement construction, unlike
	// repeated push_back. Each enum is written into the sole Runtime allocation.
	if e = satAssignmentWriteWord(c.raw.runtime, c.raw.block, 4, n); e != nil {
		return e
	}
	for i := old; i < n; i++ {
		if e = c.write(i, uint32(lUndef)); e != nil {
			return e
		}
	}
	return nil
}
