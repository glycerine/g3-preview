// Source: src/cmd_context/context_params.h:26-55;
// src/cmd_context/context_params.cpp:26-40,137-156.
// Source author: Leonardo de Moura (2012-12-01).
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

// InterpolantsInitialized remains false: the source never initializes m_interpolants.
// State of a failed constructor is diagnostic, not a callable source object.
type ContextParamsState struct {
	AutoConfig, Proof, Interpolants, InterpolantsInitialized, DebugRefCount, Trace, WellSortedCheck, Model, ModelValidate, DumpModels, UnsatCore, SMTLib2Compliant bool
	Timeout, RLimit                                                                                                                                                uint32
	DefaultsInitialized                                                                                                                                            bool
}
type ContextParams struct {
	self                       *ContextParams
	context                    *ParamsContext
	globals                    *GlobalParamsService
	budget                     *arena.Budget
	dotProof, traceFile        qiOwnedString
	state                      ContextParamsState
	header                     uint64
	ready, closed, executing   bool
	constructorCause           error
	bodyCause, destructorCause error
	frames                     *contextParamsFrame
	nextFrame                  uint64
}

func NewContextParams(c *ParamsContext, g *GlobalParamsService, b *arena.Budget) (*ContextParams, error) {
	if c == nil || g == nil || b == nil || g.context != c || g.config.Budget != b {
		return nil, ErrParamsOwner
	}
	if e := g.open(); e != nil {
		return nil, e
	}
	n := uint64(reflect.TypeOf(ContextParams{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	s := &ContextParams{context: c, globals: g, budget: b, header: n}
	s.self = s
	// Default std::string members construct in declaration order before the body.
	if e := s.dotProof.construct(b, literalParamsStringView("")); e != nil {
		s.constructorCause = e
		return s, e
	}
	if e := s.traceFile.construct(b, literalParamsStringView("")); e != nil {
		s.dotProof.sourceDestroy()
		s.constructorCause = e
		return s, e
	}
	s.state.UnsatCore = false
	s.state.Model = true
	s.state.ModelValidate = false
	s.state.DumpModels = false
	s.state.AutoConfig = true
	s.state.Proof = false
	s.state.Trace = false
	s.state.DebugRefCount = false
	s.state.SMTLib2Compliant = false
	s.state.WellSortedCheck = false
	s.state.Timeout = ^uint32(0)
	s.state.RLimit = 0
	s.state.DefaultsInitialized = true
	s.executing = true
	p, e := g.SourceGetRef()
	if e == nil {
		e = s.update(p)
	}
	s.executing = false
	if e != nil {
		s.constructorCause = e
		s.traceFile.sourceDestroy()
		s.dotProof.sourceDestroy()
	} else {
		s.ready = true
	}
	return s, e
}
func (s *ContextParams) open() error {
	if s == nil || s.self != s || s.closed {
		return ErrParamsOwner
	}
	if s.executing {
		return ErrParamsPrecondition
	}
	return nil
}
func (s *ContextParams) State() (ContextParamsState, error) {
	if e := s.open(); e != nil {
		return ContextParamsState{}, e
	}
	return s.state, nil
}
func (s *ContextParams) TraceFile() QITextView {
	if e := s.open(); e != nil {
		panic(e)
	}
	return s.traceFile.view()
}
func (s *ContextParams) DotProofFile() QITextView {
	if e := s.open(); e != nil {
		panic(e)
	}
	return s.dotProof.view()
}
func (s *ContextParams) Update(p *ParamsRef) error {
	if e := s.open(); e != nil {
		return e
	}
	if !s.ready {
		return ErrParamsPrecondition
	}
	s.executing = true
	defer func() { s.executing = false }()
	return s.update(p)
}
func (s *ContextParams) UpdateGlobal() error {
	if e := s.open(); e != nil {
		return e
	}
	if !s.ready {
		return ErrParamsPrecondition
	}
	s.executing = true
	defer func() { s.executing = false }()
	p, e := s.globals.SourceGetRef()
	if e != nil {
		return e
	}
	return s.update(p)
}
func (s *ContextParams) SetRLimit(n uint32) error {
	if e := s.open(); e != nil {
		return e
	}
	if !s.ready {
		return ErrParamsPrecondition
	}
	s.state.RLimit = n
	return nil
}
func (s *ContextParams) update(p *ParamsRef) error {
	kTimeout := "timeout"
	vTimeout, e := s.context.GetUIntName(p, &kTimeout, s.state.Timeout)
	if e != nil {
		return e
	}
	s.state.Timeout = vTimeout
	kRLimit := "rlimit"
	vRLimit, e := s.context.GetUIntName(p, &kRLimit, s.state.RLimit)
	if e != nil {
		return e
	}
	s.state.RLimit = vRLimit
	kInner := "well_sorted_check"
	inner, e := s.context.GetBoolName(p, &kInner, s.state.WellSortedCheck)
	if e != nil {
		return e
	}
	kOuter := "type_check"
	outer, e := s.context.GetBoolName(p, &kOuter, inner)
	if e != nil {
		return e
	}
	s.state.WellSortedCheck = outer
	kAutoConfig := "auto_config"
	vAutoConfig, e := s.context.GetBoolName(p, &kAutoConfig, s.state.AutoConfig)
	if e != nil {
		return e
	}
	s.state.AutoConfig = vAutoConfig
	kProof := "proof"
	vProof, e := s.context.GetBoolName(p, &kProof, s.state.Proof)
	if e != nil {
		return e
	}
	s.state.Proof = vProof
	kModel := "model"
	vModel, e := s.context.GetBoolName(p, &kModel, s.state.Model)
	if e != nil {
		return e
	}
	s.state.Model = vModel
	kModelValidate := "model_validate"
	vModelValidate, e := s.context.GetBoolName(p, &kModelValidate, s.state.ModelValidate)
	if e != nil {
		return e
	}
	s.state.ModelValidate = vModelValidate
	kDumpModels := "dump_models"
	vDumpModels, e := s.context.GetBoolName(p, &kDumpModels, s.state.DumpModels)
	if e != nil {
		return e
	}
	s.state.DumpModels = vDumpModels
	kTrace := "trace"
	vTrace, e := s.context.GetBoolName(p, &kTrace, s.state.Trace)
	if e != nil {
		return e
	}
	s.state.Trace = vTrace
	ktraceFile := "trace_file_name"
	dtraceFile := "z3.log"
	vtraceFile, e := s.context.GetStringViewFallbackName(p, &ktraceFile, nil, &dtraceFile)
	if e != nil {
		return e
	}
	if e = s.traceFile.assign(vtraceFile); e != nil {
		return e
	}
	kdotProof := "dot_proof_file"
	ddotProof := "proof.dot"
	vdotProof, e := s.context.GetStringViewFallbackName(p, &kdotProof, nil, &ddotProof)
	if e != nil {
		return e
	}
	if e = s.dotProof.assign(vdotProof); e != nil {
		return e
	}
	kUnsatCore := "unsat_core"
	vUnsatCore, e := s.context.GetBoolName(p, &kUnsatCore, s.state.UnsatCore)
	if e != nil {
		return e
	}
	s.state.UnsatCore = vUnsatCore
	kDebugRefCount := "debug_ref_count"
	vDebugRefCount, e := s.context.GetBoolName(p, &kDebugRefCount, s.state.DebugRefCount)
	if e != nil {
		return e
	}
	s.state.DebugRefCount = vDebugRefCount
	kSMTLib2Compliant := "smtlib2_compliant"
	vSMTLib2Compliant, e := s.context.GetBoolName(p, &kSMTLib2Compliant, s.state.SMTLib2Compliant)
	if e != nil {
		return e
	}
	s.state.SMTLib2Compliant = vSMTLib2Compliant
	return nil
}
func (s *ContextParams) Close() error {
	if s == nil || s.self != s {
		return ErrParamsOwner
	}
	if s.closed {
		return nil
	}
	if s.executing {
		return ErrParamsPrecondition
	}
	if e := s.closeSourceFrames(); e != nil {
		return e
	}
	// Reverse actual automatic string members, then separate own unjoined arrays.
	s.traceFile.sourceDestroy()
	s.dotProof.sourceDestroy()
	s.traceFile.ownAbort()
	s.dotProof.ownAbort()
	s.budget.Release(s.header)
	s.header = 0
	s.closed = true
	return nil
}
