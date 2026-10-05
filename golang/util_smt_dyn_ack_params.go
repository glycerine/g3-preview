// Source: src/smt/params/dyn_ack_params.h:24-51; src/smt/params/dyn_ack_params.cpp:22-41;
// src/smt/params/smt_params_helper.pyg:61-66; scripts/mk_genfile_common.py:895-939.
// Source author: Leonardo de Moura (2007-05-18).
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

// These scalar members can be embedded once in the future actual smt_params.
// The standalone owner below is an explicit Go lifetime adapter, not native
// dynamic allocation for the source scalar base or its lexical helper.
type SMTDynAckState struct {
	Strategy       uint32
	Equalities     bool
	Factor         float64
	Threshold, GC  uint32
	GCInverseDecay float64
}
type smtDynAckHelper struct {
	owner                                    *SMTDynAckParams
	next                                     *smtDynAckHelper
	value                                    ParamsRef
	receipt                                  ParamsDecrementReceipt
	header                                   uint64
	acquired, destructorStarted, sourceDone  bool
	bodyCause, destructorCause, cleanupCause error
}
type SMTDynAckParams struct {
	self             *SMTDynAckParams
	context          *ParamsContext
	globals          *GlobalParamsService
	budget           *arena.Budget
	state            SMTDynAckState
	helpers          *smtDynAckHelper
	header           uint64
	running, closed  bool
	constructorCause error
	constructorReady bool
}

func NewSMTDynAckParams(c *ParamsContext, p *ParamsRef, g *GlobalParamsService, b *arena.Budget) (*SMTDynAckParams, error) {
	if c == nil || g == nil || g.context != c || g.config.Budget != b || b == nil {
		return nil, ErrParamsOwner
	}
	if e := c.check(p); e != nil {
		return nil, e
	}
	if e := g.open(); e != nil {
		return nil, e
	}
	n := uint64(reflect.TypeOf(SMTDynAckParams{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	s := &SMTDynAckParams{context: c, globals: g, budget: b, header: n, state: SMTDynAckState{1, false, 0.1, 10, 2000, 0.8}}
	s.self = s
	e := s.update(p)
	if e != nil {
		s.constructorCause = e
	} else {
		s.constructorReady = true
	}
	// Failed source construction retains any actual helper acquisition. Owning
	// caller Close is separate, even if a lexical source destructor stopped.
	return s, e
}
func (s *SMTDynAckParams) open() error {
	if s == nil || s.self != s || s.closed {
		return ErrParamsOwner
	}
	if s.running {
		return ErrParamsPrecondition
	}
	return nil
}
func (s *SMTDynAckParams) State() (SMTDynAckState, error) {
	if e := s.open(); e != nil {
		return SMTDynAckState{}, e
	}
	return s.state, nil
}
func (s *SMTDynAckParams) finishHelper(f *smtDynAckHelper, body error) error {
	if f.bodyCause == nil {
		f.bodyCause = body
	}
	if !f.acquired {
		s.helpers = f.next
		s.budget.Release(f.header)
		f.header = 0
		return body
	}
	f.destructorStarted = true
	e := s.context.DelTracked(&f.value, &f.receipt)
	if e != nil {
		if f.destructorCause == nil {
			f.destructorCause = e
		}
		return errors.Join(body, e)
	}
	f.sourceDone = true
	s.helpers = f.next
	s.budget.Release(f.header)
	f.header = 0
	return body
}
func (s *SMTDynAckParams) Update(p *ParamsRef) error {
	if e := s.open(); e != nil {
		return e
	}
	if !s.constructorReady {
		return ErrParamsPrecondition
	}
	return s.update(p)
}
func (s *SMTDynAckParams) update(p *ParamsRef) (result error) {
	if e := s.open(); e != nil {
		return e
	}
	// Retained stopped helper frames cannot be replaced by an unrelated call.
	// This Go owning protocol guard has no source successful update credit.
	if s.helpers != nil {
		return ErrParamsPrecondition
	}
	s.running = true
	defer func() { s.running = false }()
	n := uint64(reflect.TypeOf(smtDynAckHelper{}).Size())
	if e := s.budget.Reserve(n); e != nil {
		return e
	}
	f := &smtDynAckHelper{owner: s, header: n, next: s.helpers}
	s.helpers = f
	defer func() { result = s.finishHelper(f, result) }()
	var e error
	f.value, e = s.globals.SourceGetModuleName("smt")
	if e != nil {
		return e
	}
	f.acquired = true // authority is this actual returned module reference.
	key := "dack"
	u, e := s.context.GetUIntFallbackName(p, &key, &f.value, 1)
	if e != nil {
		return e
	}
	s.state.Strategy = u
	key = "dack.eq"
	v, e := s.context.GetBoolFallbackName(p, &key, &f.value, false)
	if e != nil {
		return e
	}
	s.state.Equalities = v
	key = "dack.factor"
	d, e := s.context.GetDoubleFallbackName(p, &key, &f.value, 0.1)
	if e != nil {
		return e
	}
	s.state.Factor = d
	key = "dack.threshold"
	u, e = s.context.GetUIntFallbackName(p, &key, &f.value, 10)
	if e != nil {
		return e
	}
	s.state.Threshold = u
	key = "dack.gc"
	u, e = s.context.GetUIntFallbackName(p, &key, &f.value, 2000)
	if e != nil {
		return e
	}
	s.state.GC = u
	key = "dack.gc_inv_decay"
	d, e = s.context.GetDoubleFallbackName(p, &key, &f.value, 0.8)
	if e != nil {
		return e
	}
	s.state.GCInverseDecay = d
	return nil
}
func (s *SMTDynAckParams) Close() error {
	if s == nil || s.self != s {
		return ErrParamsOwner
	}
	if s.closed {
		return nil
	}
	if s.running {
		return ErrParamsPrecondition
	}
	s.running = true
	defer func() { s.running = false }()
	for s.helpers != nil {
		f := s.helpers
		if f.owner != s {
			return ErrParamsOwner
		}
		if f.acquired && !f.sourceDone {
			var e error
			if f.destructorStarted {
				if f.receipt.invocation == 0 {
					return f.destructorCause
				}
				e = s.context.CompleteDecrement(&f.value, f.receipt)
			} else {
				f.destructorStarted = true
				e = s.context.DelTracked(&f.value, &f.receipt)
			}
			if e != nil {
				if f.cleanupCause == nil {
					f.cleanupCause = e
				}
				return e
			}
			f.sourceDone = true
		}
		s.helpers = f.next
		s.budget.Release(f.header)
		f.header = 0
	}
	s.budget.Release(s.header)
	s.header = 0
	s.closed = true
	return nil
}

// Display preserves each source insertion and endl flush, using the selected
// actual ostream profile. Arbitrary native locale/boolalpha remain explicit
// stream-profile obligations rather than a preformatted whole-message path.
func (s *SMTDynAckParams) Display(out *NumericTextStream) error {
	if e := s.open(); e != nil {
		return e
	}
	if !s.constructorReady {
		return ErrParamsPrecondition
	}
	if e := out.AppendCString("m_dack="); e != nil {
		return e
	}
	if e := out.AppendInt32(int32(s.state.Strategy)); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_dack_eq="); e != nil {
		return e
	}
	b := int64(0)
	if s.state.Equalities {
		b = 1
	}
	if e := out.AppendInt64(b); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_dack_factor="); e != nil {
		return e
	}
	if e := out.AppendFloat64(s.state.Factor); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_dack_threshold="); e != nil {
		return e
	}
	if e := out.AppendUint32(s.state.Threshold); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_dack_gc="); e != nil {
		return e
	}
	if e := out.AppendUint32(s.state.GC); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_dack_gc_inv_decay="); e != nil {
		return e
	}
	if e := out.AppendFloat64(s.state.GCInverseDecay); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	return out.Flush()
}
