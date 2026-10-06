// Source: src/smt/params/smt_params.h:75-318; src/smt/params/qi_params.h:31-109;
// src/qe/qe_sat_tactic.cpp:632; src/muz/pdr/pdr_farkas_learner.cpp:390.
// Source author: Leonardo de Moura (2008-02-20; 2008-06-15).
// Licensed under the MIT license; see LICENSE.txt.
package g3

import "reflect"

// Compiler-generated copies copy source members, never Go owner/receipt history.
// Borrowed global services need not be touched while copying inline values and
// already-owned string storage. Ended or source-stopped owners are a Go guard.
func (s *SMTParams) copySourceReady() error {
	if e := s.open(); e != nil {
		return e
	}
	if !s.constructorReady || s.helpers != nil || s.sourceStop() != nil {
		return ErrParamsPrecondition
	}
	return nil
}
func (s *SMTParams) initCopyOwner(o *smtModuleParamsOwner) {
	*o = smtModuleParamsOwner{context: s.context, globals: s.globals, budget: s.budget}
	o.self = o
}
func (s *SMTParams) Clone() (*SMTParams, error) {
	if e := s.copySourceReady(); e != nil {
		return nil, e
	}
	n := uint64(reflect.TypeOf(SMTParams{}).Size())
	if e := s.budget.Reserve(n); e != nil {
		return nil, e
	}
	d := &SMTParams{header: n, empty: ParamsRef{owner: s.context.owner}}
	d.self = d
	d.smtModuleParamsOwner = smtModuleParamsOwner{context: s.context, globals: s.globals, budget: s.budget}
	d.smtModuleParamsOwner.self = &d.smtModuleParamsOwner
	// Declaration-order bases. The first two contain only source scalar members.
	d.initCopyOwner(&d.preprocess.smtModuleParamsOwner)
	d.preprocess.self = &d.preprocess
	d.preprocess.empty = ParamsRef{owner: s.context.owner}
	d.initCopyOwner(&d.preprocess.pattern.smtModuleParamsOwner)
	d.preprocess.pattern.self = &d.preprocess.pattern
	d.preprocess.pattern.state = s.preprocess.pattern.state
	d.preprocess.pattern.constructorReady = true
	d.preprocess.bit = s.preprocess.bit
	d.preprocess.completedBases = 3
	d.preprocess.state = s.preprocess.state
	d.preprocess.constructorReady = true
	d.completedBases |= 1
	d.initCopyOwner(&d.dyn.smtModuleParamsOwner)
	d.dyn.self = &d.dyn
	d.dyn.state = s.dyn.state
	d.dyn.constructorReady = true
	d.completedBases |= 2
	d.initCopyOwner(&d.qi.smtModuleParamsOwner)
	d.qi.self = &d.qi
	// m_qi_ematching is declared before the two actual string constructors.
	d.qi.state.Ematching = s.qi.state.Ematching
	d.qi.state.EmatchingInitialized = s.qi.state.EmatchingInitialized
	if e := d.qi.cost.copyConstruct(d.budget, &s.qi.cost); e != nil {
		return d, d.copyConstructorFailure(e)
	}
	d.qi.completed = 1
	if e := d.qi.newGen.copyConstruct(d.budget, &s.qi.newGen); e != nil {
		d.qi.cost.sourceDestroy()
		return d, d.copyConstructorFailure(e)
	}
	d.qi.completed = 3
	d.qi.state = s.qi.state
	d.qi.constructorReady = true
	d.completedBases |= 4
	d.initCopyOwner(&d.arith.smtModuleParamsOwner)
	d.arith.self = &d.arith
	d.arith.state = s.arith.state
	d.arith.constructorReady = true
	d.completedBases |= 8
	d.initCopyOwner(&d.array.smtModuleParamsOwner)
	d.array.self = &d.array
	d.array.state = s.array.state
	d.completedBases |= 16
	d.initCopyOwner(&d.bv.smtModuleParamsOwner)
	d.bv.self = &d.bv
	d.bv.state = s.bv.state
	d.bv.constructorReady = true
	d.completedBases |= 32
	d.initCopyOwner(&d.str.smtModuleParamsOwner)
	d.str.self = &d.str
	d.str.state = s.str.state
	d.str.constructorReady = true
	d.completedBases |= 64
	d.initCopyOwner(&d.pb.smtModuleParamsOwner)
	d.pb.self = &d.pb
	d.pb.state = s.pb.state
	d.completedBases |= 128
	d.datatype = s.datatype
	d.completedBases |= 256
	d.state = s.state
	d.constructorReady = true
	return d, nil
}
func (s *SMTParams) copyConstructorFailure(e error) error {
	s.constructorCause = e
	s.qi.constructorCause = e
	s.endBasesSource() // Only the actually completed preprocessor/dyn bases.
	return e
}
func (s *SMTParams) Assign(src *SMTParams) error {
	if e := s.admitMutation(); e != nil {
		return e
	}
	if src == nil {
		return ErrParamsOwner
	}
	if e := src.copySourceReady(); e != nil {
		return e
	}
	if s.context != src.context || s.globals != src.globals || s.budget != src.budget {
		return ErrParamsOwner
	}
	if s == src {
		return nil
	}
	s.activeCall = true
	defer func() { s.activeCall = false }()
	s.preprocess.pattern.state = src.preprocess.pattern.state
	s.preprocess.bit = src.preprocess.bit
	s.preprocess.state = src.preprocess.state
	s.dyn.state = src.dyn.state
	s.qi.state.Ematching = src.qi.state.Ematching
	s.qi.state.EmatchingInitialized = src.qi.state.EmatchingInitialized
	if e := s.qi.cost.copyAssign(&src.qi.cost); e != nil {
		return s.copyAssignmentFailure(e)
	}
	if e := s.qi.newGen.copyAssign(&src.qi.newGen); e != nil {
		return s.copyAssignmentFailure(e)
	}
	s.qi.state = src.qi.state
	s.arith.state = src.arith.state
	s.array.state = src.array.state
	s.bv.state = src.bv.state
	s.str.state = src.str.state
	s.pb.state = src.pb.state
	s.datatype = src.datatype
	s.state = src.state
	return nil
}
func (s *SMTParams) copyAssignmentFailure(e error) error {
	if s.copyCause == nil {
		s.copyCause = e
	}
	return e
}
