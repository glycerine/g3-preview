// Source: src/smt/params/theory_bv_params.h:24-50;
// src/smt/params/theory_bv_params.cpp:23-40; src/smt/params/smt_params_helper.pyg:38-39;
// src/ast/rewriter/bv_rewriter_params.pyg:1-9; scripts/mk_genfile_common.py:895-939.
// Source author: Leonardo de Moura (2008-06-06; 2012-12-02).
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

type SMTBVState struct {
	Mode                        uint32
	HiDiv0, Reflect, LazyLE, CC bool
	BlastMaxSize                uint32
	EnableInt2BV                bool
}
type SMTBVParams struct {
	smtModuleParamsOwner
	self                     *SMTBVParams
	state                    SMTBVState
	header                   uint64
	closed, constructorReady bool
	constructorCause         error
}

func NewSMTBVParams(c *ParamsContext, p *ParamsRef, g *GlobalParamsService, b *arena.Budget) (*SMTBVParams, error) {
	if c == nil || g == nil || b == nil || g.context != c || g.config.Budget != b {
		return nil, ErrParamsOwner
	}
	if e := c.check(p); e != nil {
		return nil, e
	}
	if e := g.open(); e != nil {
		return nil, e
	}
	n := uint64(reflect.TypeOf(SMTBVParams{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	s := &SMTBVParams{smtModuleParamsOwner: smtModuleParamsOwner{context: c, globals: g, budget: b}, header: n, state: SMTBVState{1, false, true, false, false, 2147483647, true}}
	s.self = s
	s.smtModuleParamsOwner.self = &s.smtModuleParamsOwner
	e := s.update(p)
	if e != nil {
		s.constructorCause = e
	} else {
		s.constructorReady = true
	}
	return s, e
}
func (s *SMTBVParams) open() error {
	if s == nil || s.self != s || s.closed {
		return ErrParamsOwner
	}
	if s.running {
		return ErrParamsPrecondition
	}
	return nil
}
func (s *SMTBVParams) State() (SMTBVState, error) {
	if e := s.open(); e != nil {
		return SMTBVState{}, e
	}
	return s.state, nil
}
func (s *SMTBVParams) Update(p *ParamsRef) error {
	if e := s.open(); e != nil {
		return e
	}
	if !s.constructorReady {
		return ErrParamsPrecondition
	}
	return s.update(p)
}
func (s *SMTBVParams) update(local *ParamsRef) (result error) {
	if e := s.open(); e != nil {
		return e
	}
	if s.helpers != nil || s.destructorCause != nil {
		return ErrParamsPrecondition
	}
	s.running = true
	defer func() { s.running = false }()
	p, e := s.beginHelper(local, "smt")
	if p != nil {
		defer func() { result = s.finishHelper(p, result) }()
	}
	if e != nil {
		return e
	}
	rp, e := s.beginHelper(local, "rewriter")
	if rp != nil {
		defer func() { result = s.finishHelper(rp, result) }()
	}
	if e != nil {
		return e
	}
	v, e := s.frameBool(rp, "hi_div0", true)
	if e != nil {
		return e
	}
	s.state.HiDiv0 = v
	v, e = s.frameBool(p, "bv.reflect", true)
	if e != nil {
		return e
	}
	s.state.Reflect = v
	v, e = s.frameBool(p, "bv.enable_int2bv", true)
	if e != nil {
		return e
	}
	s.state.EnableInt2BV = v
	return nil
}
func (s *SMTBVParams) Close() error {
	if s == nil || s.self != s {
		return ErrParamsOwner
	}
	if s.closed {
		return nil
	}
	if s.running {
		return ErrParamsPrecondition
	}
	if e := s.closeHelpers(); e != nil {
		return e
	}
	s.budget.Release(s.header)
	s.header = 0
	s.closed = true
	return nil
}
func (s *SMTBVParams) Display(out *NumericTextStream) error {
	if e := s.open(); e != nil {
		return e
	}
	if !s.constructorReady {
		return ErrParamsPrecondition
	}
	if e := out.AppendCString("m_bv_mode="); e != nil {
		return e
	}
	if e := out.AppendInt32(int32(s.state.Mode)); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_hi_div0="); e != nil {
		return e
	}
	v := int64(0)
	if s.state.HiDiv0 {
		v = 1
	}
	if e := out.AppendInt64(v); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_bv_reflect="); e != nil {
		return e
	}
	v = 0
	if s.state.Reflect {
		v = 1
	}
	if e := out.AppendInt64(v); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_bv_lazy_le="); e != nil {
		return e
	}
	v = 0
	if s.state.LazyLE {
		v = 1
	}
	if e := out.AppendInt64(v); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_bv_cc="); e != nil {
		return e
	}
	v = 0
	if s.state.CC {
		v = 1
	}
	if e := out.AppendInt64(v); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_bv_blast_max_size="); e != nil {
		return e
	}
	if e := out.AppendUint32(s.state.BlastMaxSize); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_bv_enable_int2bv2int="); e != nil {
		return e
	}
	v = 0
	if s.state.EnableInt2BV {
		v = 1
	}
	if e := out.AppendInt64(v); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	return out.Flush()
}
