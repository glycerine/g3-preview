// Source: src/math/polynomial/polynomial.h:62-69; src/math/polynomial/polynomial.cpp:66-82; src/util/vector.h:101-116,160-210,530-546.
// Licensed under the MIT license.
package mpnum

import "github.com/glycerine/g3/golang/memory"

// PolynomialVariableDegreesTextSink is a borrowed cold ostream binding.
// NumericTextStream supplies the actual integer formatting state. The map
// never owns the stream and introduces no interface in per-variable storage.
type PolynomialVariableDegreesTextSink interface {
	AppendCString(string) error
	AppendUint32(uint32) error
}

func (d *PolynomialVariableDegrees) DisplayInto(out PolynomialVariableDegreesTextSink) error {
	if e := d.open(); e != nil {
		return e
	}
	if out == nil {
		return ErrPolynomialOwner
	}
	first := true
	if e := out.AppendCString("["); e != nil {
		return e
	}
	for i := uint32(0); ; i++ {
		n, e := d.size()
		if e != nil {
			return e
		}
		if i >= n {
			break
		}
		n, e = d.size()
		if e != nil {
			return e
		}
		if n > 0 {
			if !first {
				if e = out.AppendCString(","); e != nil {
					return e
				}
			}
			if e = out.AppendCString("x"); e != nil {
				return e
			}
			if e = out.AppendUint32(i); e != nil {
				return e
			}
			if e = out.AppendCString("^"); e != nil {
				return e
			}
			if modularSourceDebug {
				n, e = d.size()
				if e != nil {
					return e
				}
				if i >= n {
					return ErrPolynomialAssertion
				}
			}
			v, e := d.runtime.ReadUint32(d.raw, uint64(8)+uint64(i)*4)
			if e != nil {
				return e
			}
			if e = out.AppendUint32(v); e != nil {
				return e
			}
			if first {
				first = false
			}
		}
	}
	return out.AppendCString("]")
}

// copyCore preserves source SIZE then CAP reads and size_t multiplication.
// The published new backing is the actual member root before element copies;
// an allocation returned with error remains a separate owning pending root.
func (d *PolynomialVariableDegrees) copyCore(source *PolynomialVariableDegrees) error {
	n, e := source.size()
	if e != nil {
		return e
	}
	cap, e := source.capacity()
	if e != nil {
		return e
	}
	bytes := uint64(cap)*4 + 8
	if d.runtime.SizeTBits() == 32 {
		bytes = uint64(uint32(bytes))
	}
	d.pending, e = d.runtime.Allocate(bytes)
	if e != nil {
		return e
	}
	if e = d.runtime.WriteUint32(d.pending, 0, cap); e != nil {
		return e
	}
	if e = d.runtime.WriteUint32(d.pending, 4, n); e != nil {
		return e
	}
	d.raw = d.pending
	d.pending = memory.Block{}
	d.rawDestroyed = false
	// source.end() is evaluated only after m_data publication.
	end, e := source.size()
	if e != nil {
		return e
	}
	for i := uint32(0); i < end; i++ {
		v, e := source.runtime.ReadUint32(source.raw, uint64(8)+uint64(i)*4)
		if e != nil {
			return e
		}
		if e = d.runtime.WriteUint32(d.raw, uint64(8)+uint64(i)*4, v); e != nil {
			return e
		}
	}
	return nil
}
func NewPolynomialVariableDegreesCopy(source *PolynomialVariableDegrees) (*PolynomialVariableDegrees, error) {
	if e := source.open(); e != nil {
		return nil, e
	}
	d, e := NewPolynomialVariableDegrees(source.runtime)
	if e != nil {
		return nil, e
	}
	d.constructed = false
	if source.raw != (memory.Block{}) {
		e = d.copyCore(source)
	}
	if e == nil && modularSourceDebug {
		var n, x uint32
		n, e = d.size()
		if e == nil {
			x, e = source.size()
			if e == nil && n != x {
				e = ErrPolynomialAssertion
			}
		}
	}
	if e != nil {
		d.bodyCause = e
		d.copyBroken = true
		if d.raw == (memory.Block{}) && d.pending == (memory.Block{}) {
			d.runtime.Budget().Release(d.header)
			d.header = 0
			d.closed = true
			return nil, e
		}
		return d, e
	}
	d.constructed = true
	return d, nil
}

// destroyForAssignment retains the source pointer history after a completed
// free. Runtime's physical token update is not a source m_data=null assignment.
func (d *PolynomialVariableDegrees) destroyForAssignment() error {
	if d.raw == (memory.Block{}) {
		return nil
	}
	// unsigned_vector is svector<unsigned>: CallDestructors=false, so no
	// SIZE read or scalar destruction precedes the actual free.
	old := d.raw
	released := d.raw
	e := d.runtime.Deallocate(&released)
	if released == (memory.Block{}) {
		d.rawDestroyed = true
	}
	d.raw = old
	return e
}
func (d *PolynomialVariableDegrees) CopyFrom(source *PolynomialVariableDegrees) error {
	if e := d.open(); e != nil {
		return e
	}
	if d == source {
		return nil
	}
	if e := source.open(); e != nil {
		return e
	}
	if d.copyBroken || d.pending != (memory.Block{}) || d.interrupted {
		return ErrPolynomialPending
	}
	if e := d.destroyForAssignment(); e != nil {
		d.copyBroken = true
		d.bodyCause = e
		return e
	}
	var e error
	if source.raw != (memory.Block{}) {
		e = d.copyCore(source)
	} else {
		d.raw = memory.Block{}
		d.rawDestroyed = false
	}
	if e != nil {
		// A previously empty target stays source-valid when allocation throws
		// without returning a root. Only actual dangling/partial/pending history
		// needs the Go stopped-object guard.
		d.copyBroken = d.rawDestroyed || d.raw != (memory.Block{}) || d.pending != (memory.Block{})
		d.bodyCause = e
	} else {
		d.bodyCause = nil
	}
	return e
}
func NewPolynomialVariableDegreesMove(source *PolynomialVariableDegrees) (*PolynomialVariableDegrees, error) {
	if e := source.open(); e != nil {
		return nil, e
	}
	d, e := NewPolynomialVariableDegrees(source.runtime)
	if e != nil {
		return nil, e
	}
	d.raw, source.raw = source.raw, d.raw
	d.rawDestroyed, source.rawDestroyed = source.rawDestroyed, false
	return d, nil
}

// MoveFrom follows var2degree's implicit move assignment. Its unsigned_vector
// member is svector<unsigned>, which declares only operator=(const&), hiding
// the base vector&& assignment. The actual source route is therefore copy
// assignment; the source pointer stays unchanged. Move construction above is
// genuinely stealing because svector explicitly declares that constructor.
func (d *PolynomialVariableDegrees) MoveFrom(source *PolynomialVariableDegrees) error {
	return d.CopyFrom(source)
}
