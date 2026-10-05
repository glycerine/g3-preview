// Source: src/ast/pattern/pattern_inference_params.h:30-50;
// src/ast/pattern/pattern_inference_params.cpp:22-47; src/ast/pattern/pattern_inference_params_helper.pyg:1-12.
// Source author: Leonardo de Moura (2008-03-24; 2012-12-02).
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

// Initialized is cold diagnostic metadata; unassigned scalar fields do not
// become source defaults merely because their Go representation contains zero.
type SMTPatternInferenceState struct {
	MaxMultiPatterns          uint32
	BlockLoopPatterns         bool
	Arithmetic                uint32
	UseDatabase               bool
	ArithmeticWeight          uint32
	NonNestedArithmeticWeight uint32
	PullQuantifiers           bool
	NoPatternWeight           int32
	AvoidSkolems              bool
	Warnings                  bool
	Initialized               uint16
}
type SMTPatternInferenceParams struct {
	smtModuleParamsOwner
	self                     *SMTPatternInferenceParams
	state                    SMTPatternInferenceState
	header                   uint64
	closed, constructorReady bool
	constructorCause         error
}

func NewSMTPatternInferenceParams(c *ParamsContext, p *ParamsRef, g *GlobalParamsService, b *arena.Budget) (*SMTPatternInferenceParams, error) {
	if c == nil || g == nil || b == nil || g.context != c || g.config.Budget != b {
		return nil, ErrParamsOwner
	}
	if e := c.check(p); e != nil {
		return nil, e
	}
	if e := g.open(); e != nil {
		return nil, e
	}
	n := uint64(reflect.TypeOf(SMTPatternInferenceParams{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	s := &SMTPatternInferenceParams{header: n}
	e := s.constructSource(c, p, g, b)
	return s, e
}

// Same actual constructor engine for standalone and embedded source bases. The
// containing object charges the embedded fields once; no extra wrapper/header.
func (s *SMTPatternInferenceParams) constructSource(c *ParamsContext, p *ParamsRef, g *GlobalParamsService, b *arena.Budget) error {
	if s == nil || s.self != nil {
		return ErrParamsOwner
	}
	s.smtModuleParamsOwner = smtModuleParamsOwner{context: c, globals: g, budget: b}
	s.state = SMTPatternInferenceState{NoPatternWeight: -1, AvoidSkolems: true, Initialized: 0x180}
	s.self = s
	s.smtModuleParamsOwner.self = &s.smtModuleParamsOwner
	e := s.update(p)
	if e != nil {
		s.constructorCause = e
	} else {
		s.constructorReady = true
	}
	return e
}

func (s *SMTPatternInferenceParams) open() error {
	if s == nil || s.self != s || s.closed {
		return ErrParamsOwner
	}
	if s.running {
		return ErrParamsPrecondition
	}
	return nil
}
func (s *SMTPatternInferenceParams) State() (SMTPatternInferenceState, error) {
	if e := s.open(); e != nil {
		return SMTPatternInferenceState{}, e
	}
	return s.state, nil
}
func (s *SMTPatternInferenceParams) Update(p *ParamsRef) error {
	if e := s.open(); e != nil {
		return e
	}
	if !s.constructorReady {
		return ErrParamsPrecondition
	}
	return s.update(p)
}
func (s *SMTPatternInferenceParams) update(local *ParamsRef) (result error) {
	if e := s.open(); e != nil {
		return e
	}
	if s.helpers != nil || s.destructorCause != nil {
		return ErrParamsPrecondition
	}
	s.running = true
	defer func() { s.running = false }()
	p, e := s.beginHelper(local, "pi")
	if p != nil {
		defer func() { result = s.finishHelper(p, result) }()
	}
	if e != nil {
		return e
	}
	vMaxMultiPatterns, e := s.frameUInt(p, "max_multi_patterns", 0)
	if e != nil {
		return e
	}
	s.state.MaxMultiPatterns = vMaxMultiPatterns
	s.state.Initialized |= 1 << 0
	vBlockLoopPatterns, e := s.frameBool(p, "block_loop_patterns", true)
	if e != nil {
		return e
	}
	s.state.BlockLoopPatterns = vBlockLoopPatterns
	s.state.Initialized |= 1 << 1
	vArithmetic, e := s.frameUInt(p, "arith", 1)
	if e != nil {
		return e
	}
	s.state.Arithmetic = vArithmetic
	s.state.Initialized |= 1 << 2
	vUseDatabase, e := s.frameBool(p, "use_database", false)
	if e != nil {
		return e
	}
	s.state.UseDatabase = vUseDatabase
	s.state.Initialized |= 1 << 3
	vArithmeticWeight, e := s.frameUInt(p, "arith_weight", 5)
	if e != nil {
		return e
	}
	s.state.ArithmeticWeight = vArithmeticWeight
	s.state.Initialized |= 1 << 4
	vNonNestedArithmeticWeight, e := s.frameUInt(p, "non_nested_arith_weight", 10)
	if e != nil {
		return e
	}
	s.state.NonNestedArithmeticWeight = vNonNestedArithmeticWeight
	s.state.Initialized |= 1 << 5
	vPullQuantifiers, e := s.frameBool(p, "pull_quantifiers", true)
	if e != nil {
		return e
	}
	s.state.PullQuantifiers = vPullQuantifiers
	s.state.Initialized |= 1 << 6
	vWarnings, e := s.frameBool(p, "warnings", false)
	if e != nil {
		return e
	}
	s.state.Warnings = vWarnings
	s.state.Initialized |= 1 << 9
	return nil
}
func (s *SMTPatternInferenceParams) Close() error {
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
func (s *SMTPatternInferenceParams) Display(out *NumericTextStream) error {
	if e := s.open(); e != nil {
		return e
	}
	if !s.constructorReady {
		return ErrParamsPrecondition
	}
	if e := out.AppendCString("m_pi_max_multi_patterns="); e != nil {
		return e
	}
	if e := out.AppendUint32(s.state.MaxMultiPatterns); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_pi_block_loop_patterns="); e != nil {
		return e
	}
	bBlockLoopPatterns := int64(0)
	if s.state.BlockLoopPatterns {
		bBlockLoopPatterns = 1
	}
	if e := out.AppendInt64(bBlockLoopPatterns); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_pi_arith="); e != nil {
		return e
	}
	if e := out.AppendInt32(int32(s.state.Arithmetic)); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_pi_use_database="); e != nil {
		return e
	}
	bUseDatabase := int64(0)
	if s.state.UseDatabase {
		bUseDatabase = 1
	}
	if e := out.AppendInt64(bUseDatabase); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_pi_arith_weight="); e != nil {
		return e
	}
	if e := out.AppendUint32(s.state.ArithmeticWeight); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_pi_non_nested_arith_weight="); e != nil {
		return e
	}
	if e := out.AppendUint32(s.state.NonNestedArithmeticWeight); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_pi_pull_quantifiers="); e != nil {
		return e
	}
	bPullQuantifiers := int64(0)
	if s.state.PullQuantifiers {
		bPullQuantifiers = 1
	}
	if e := out.AppendInt64(bPullQuantifiers); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_pi_nopat_weight="); e != nil {
		return e
	}
	if e := out.AppendInt32(s.state.NoPatternWeight); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_pi_avoid_skolems="); e != nil {
		return e
	}
	bAvoidSkolems := int64(0)
	if s.state.AvoidSkolems {
		bAvoidSkolems = 1
	}
	if e := out.AppendInt64(bAvoidSkolems); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_pi_warnings="); e != nil {
		return e
	}
	bWarnings := int64(0)
	if s.state.Warnings {
		bWarnings = 1
	}
	if e := out.AppendInt64(bWarnings); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	return nil
}
