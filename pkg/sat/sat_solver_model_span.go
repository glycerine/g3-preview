// Source: src/sat/sat_types.h:138; src/util/lbool.h; src/util/vector.h:407–420; src/sat/sat_model_converter.cpp:35–114.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2011.
package sat

import (
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// Concrete model& borrow. Runtime reads resolve the current member backing at
// every operator[]; the preexisting caller-owned slice route is unchanged.
// The source lbool enum is signed32, unlike the solver's char assignments.
type satSolverModelSpan struct {
	slice  Model
	column *RuntimeUintColumn
}

func (m satSolverModelSpan) read(v BoolVar) (util.LBool, error) {
	if m.column != nil {
		x, e := m.column.Read(uint32(v))
		return util.LBool(int32(x)), e
	}
	if uint64(v) >= uint64(len(m.slice)) {
		return util.LUndef, arena.ErrBounds
	}
	return m.slice[v], nil
}
func (m satSolverModelSpan) write(v BoolVar, x util.LBool) error {
	if m.column != nil {
		return m.column.Write(uint32(v), uint32(int32(x)))
	}
	if uint64(v) >= uint64(len(m.slice)) {
		return arena.ErrBounds
	}
	m.slice[v] = x
	return nil
}
func (m satSolverModelSpan) value(l Literal) (util.LBool, error) {
	v, e := m.read(l.variable())
	if e != nil {
		return util.LUndef, e
	}
	if l.sign() {
		v = v.Negated()
	}
	return v, nil
}
func satSolverResizeModel(c *RuntimeUintColumn, n uint32) error {
	old, e := c.Size()
	if e != nil {
		return e
	}
	if n <= old {
		return c.Shrink(n)
	}
	for {
		cap, e := c.Capacity()
		if e != nil {
			return e
		}
		if n <= cap {
			break
		}
		if e = c.Raw.ExpandStride(4); e != nil {
			return e
		}
	}
	// Source size is published before ascending placement construction, unlike
	// repeated push_back. Each enum is written into the sole Runtime allocation.
	if e = WriteRuntimeWord(c.Raw.Runtime, c.Raw.Block, 4, n); e != nil {
		return e
	}
	for i := old; i < n; i++ {
		if e = c.Write(i, uint32(util.LUndef)); e != nil {
			return e
		}
	}
	return nil
}
