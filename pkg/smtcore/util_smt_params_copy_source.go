// Source: src/smt/params/smt_params.h:75-318; src/smt/params/qi_params.h:31-109;
// src/qe/qe_sat_tactic.cpp:632; src/muz/pdr/pdr_farkas_learner.cpp:390.
package smtcore

import (
	"reflect"

	"github.com/glycerine/g3/pkg/util"
)

// Compiler-generated copies copy source members, never Go owner/receipt history.
// Borrowed global services need not be touched while copying inline values and
// already-owned string storage. Ended or source-stopped owners are a Go guard.

func (s *SMTParams) copySourceReady() error {
	if e := s.open(); e != nil {
		return e
	}
	if !s.constructorReady || s.Helpers != nil || s.sourceStop() != nil {
		return util.ErrParamsPrecondition
	}
	return nil
}
func (s *SMTParams) initCopyOwner(o *util.ModuleParamsOwner) {
	*o = util.ModuleParamsOwner{Context: s.Context, Globals: s.Globals, Budget: s.Budget}
	o.Self = o
}
func (s *SMTParams) Clone() (*SMTParams, error) {
	if e := s.copySourceReady(); e != nil {
		return nil, e
	}
	n := uint64(reflect.TypeOf(SMTParams{}).Size())
	if e := s.Budget.Reserve(n); e != nil {
		return nil, e
	}
	d := &SMTParams{header: n, empty: s.Context.EmptyValue()}
	d.self = d
	d.ModuleParamsOwner = util.ModuleParamsOwner{Context: s.Context, Globals: s.Globals, Budget: s.Budget}
	d.ModuleParamsOwner.Self = &d.ModuleParamsOwner
	// Declaration-order bases. The first two contain only source scalar members.
	d.initCopyOwner(&d.preprocess.ModuleParamsOwner)
	d.preprocess.self = &d.preprocess
	d.preprocess.empty = s.Context.EmptyValue()
	d.initCopyOwner(&d.preprocess.pattern.ModuleParamsOwner)
	d.preprocess.pattern.self = &d.preprocess.pattern
	d.preprocess.pattern.state = s.preprocess.pattern.state
	d.preprocess.pattern.constructorReady = true
	d.preprocess.bit = s.preprocess.bit
	d.preprocess.completedBases = 3
	d.preprocess.state = s.preprocess.state
	d.preprocess.constructorReady = true
	d.completedBases |= 1
	d.initCopyOwner(&d.dyn.ModuleParamsOwner)
	d.dyn.self = &d.dyn
	d.dyn.state = s.dyn.state
	d.dyn.constructorReady = true
	d.completedBases |= 2
	d.initCopyOwner(&d.qi.ModuleParamsOwner)
	d.qi.self = &d.qi
	// m_qi_ematching is declared before the two actual string constructors.
	d.qi.state.Ematching = s.qi.state.Ematching
	d.qi.state.EmatchingInitialized = s.qi.state.EmatchingInitialized
	if e := d.qi.cost.CopyConstruct(d.Budget, &s.qi.cost); e != nil {
		return d, d.copyConstructorFailure(e)
	}
	d.qi.completed = 1
	if e := d.qi.newGen.CopyConstruct(d.Budget, &s.qi.newGen); e != nil {
		d.qi.cost.SourceDestroy()
		return d, d.copyConstructorFailure(e)
	}
	d.qi.completed = 3
	d.qi.state = s.qi.state
	d.qi.constructorReady = true
	d.completedBases |= 4
	d.initCopyOwner(&d.arith.ModuleParamsOwner)
	d.arith.self = &d.arith
	d.arith.state = s.arith.state
	d.arith.constructorReady = true
	d.completedBases |= 8
	d.initCopyOwner(&d.array.ModuleParamsOwner)
	d.array.self = &d.array
	d.array.state = s.array.state
	d.completedBases |= 16
	d.initCopyOwner(&d.bv.ModuleParamsOwner)
	d.bv.self = &d.bv
	d.bv.state = s.bv.state
	d.bv.constructorReady = true
	d.completedBases |= 32
	d.initCopyOwner(&d.str.ModuleParamsOwner)
	d.str.self = &d.str
	d.str.state = s.str.state
	d.str.constructorReady = true
	d.completedBases |= 64
	d.initCopyOwner(&d.pb.ModuleParamsOwner)
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
		return util.ErrParamsOwner
	}
	if e := src.copySourceReady(); e != nil {
		return e
	}
	if s.Context != src.Context || s.Globals != src.Globals || s.Budget != src.Budget {
		return util.ErrParamsOwner
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
	if e := s.qi.cost.CopyAssign(&src.qi.cost); e != nil {
		return s.copyAssignmentFailure(e)
	}
	if e := s.qi.newGen.CopyAssign(&src.qi.newGen); e != nil {
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
