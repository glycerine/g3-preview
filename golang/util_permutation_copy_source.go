// Source: src/util/permutation.h:25-40; src/util/vector.h:159-167,191-202,536-541.
// Source author: Leonardo de Moura.
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"errors"
	"reflect"
)

// Clone follows the implicit member copy constructor: forward then inverse.
// Only a completed vector member has a source destructor on constructor unwind.
func (p *RuntimePermutation) Clone() (*RuntimePermutation, error) {
	if e := p.open(); e != nil {
		return nil, e
	}
	h := uint64(reflect.TypeOf(RuntimePermutation{}).Size())
	if e := p.budget.Reserve(h); e != nil {
		return nil, e
	}
	out := &RuntimePermutation{budget: p.budget, header: h}
	out.self = out
	out.forward.bind(p.forward.column.raw.runtime, p.budget)
	out.inverse.bind(p.inverse.column.raw.runtime, p.budget)
	if e := out.forward.copyConstruct(&p.forward); e != nil {
		out.constructorCause = e
		if d := out.SourceFinalize(); d != nil {
			return out, errors.Join(e, d)
		}
		return out, e
	}
	out.completedMembers = 1
	if e := out.inverse.copyConstruct(&p.inverse); e != nil {
		out.constructorCause = e
		if d := out.SourceFinalize(); d != nil {
			return out, errors.Join(e, d)
		}
		return out, e
	}
	out.completedMembers = 3
	return out, nil
}
func (p *RuntimePermutation) Assign(src *RuntimePermutation) error {
	if e := p.open(); e != nil {
		return e
	}
	if e := src.open(); e != nil {
		return e
	}
	if e := p.forward.Assign(&src.forward); e != nil {
		return e
	}
	return p.inverse.Assign(&src.inverse)
}
