// Source: src/smt/params/theory_str_params.h:25-143;
// src/smt/params/theory_str_params.cpp:21-40; src/smt/params/smt_params_helper.pyg:70-87.
// Source author: Murphy Berzish (2016-12-13).
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

type SMTStrState struct {
	StrongArrangements                           bool
	AggressiveLengthTesting                      bool
	AggressiveValueTesting                       bool
	AggressiveUnrollTesting                      bool
	UseFastLengthTesterCache                     bool
	UseFastValueTesterCache                      bool
	StringConstantCache                          bool
	FiniteOverlapModels                          bool
	UseBinarySearch                              bool
	BinarySearchInitialUpperBound                uint32
	OverlapTheoryAwarePriority                   float64
	RegexAutomata                                bool
	RegexAutomataDifficultyThreshold             uint32
	RegexAutomataIntersectionDifficultyThreshold uint32
	RegexAutomataFailedAutomatonThreshold        uint32
	RegexAutomataFailedIntersectionThreshold     uint32
	RegexAutomataLengthAttemptThreshold          uint32
}
type SMTStrParams struct {
	smtModuleParamsOwner
	self                     *SMTStrParams
	state                    SMTStrState
	header                   uint64
	closed, constructorReady bool
	constructorCause         error
}

func NewSMTStrParams(c *ParamsContext, p *ParamsRef, g *GlobalParamsService, b *arena.Budget) (*SMTStrParams, error) {
	if c == nil || g == nil || b == nil || g.context != c || g.config.Budget != b {
		return nil, ErrParamsOwner
	}
	if e := c.check(p); e != nil {
		return nil, e
	}
	if e := g.open(); e != nil {
		return nil, e
	}
	n := uint64(reflect.TypeOf(SMTStrParams{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	s := &SMTStrParams{header: n}
	e := s.constructSource(c, p, g, b)
	return s, e
}

// Same source constructor engine: an actual containing header charges embedded fields once.
func (s *SMTStrParams) constructSource(c *ParamsContext, p *ParamsRef, g *GlobalParamsService, b *arena.Budget) error {
	if s == nil || s.self != nil {
		return ErrParamsOwner
	}
	*s = SMTStrParams{smtModuleParamsOwner: smtModuleParamsOwner{context: c, globals: g, budget: b}, header: s.header, state: SMTStrState{true, false, false, true, false, true, true, false, false, 64, -0.1, true, 1000, 1000, 10, 10, 10}}
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
func (s *SMTStrParams) open() error {
	if s == nil || s.self != s || s.closed {
		return ErrParamsOwner
	}
	if s.running {
		return ErrParamsPrecondition
	}
	return nil
}
func (s *SMTStrParams) State() (SMTStrState, error) {
	if e := s.open(); e != nil {
		return SMTStrState{}, e
	}
	return s.state, nil
}
func (s *SMTStrParams) Update(p *ParamsRef) error {
	if e := s.open(); e != nil {
		return e
	}
	if !s.constructorReady {
		return ErrParamsPrecondition
	}
	return s.update(p)
}
func (s *SMTStrParams) update(local *ParamsRef) (result error) {
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
	vStrongArrangements, e := s.frameBool(p, "str.strong_arrangements", true)
	if e != nil {
		return e
	}
	s.state.StrongArrangements = vStrongArrangements
	vAggressiveLengthTesting, e := s.frameBool(p, "str.aggressive_length_testing", false)
	if e != nil {
		return e
	}
	s.state.AggressiveLengthTesting = vAggressiveLengthTesting
	vAggressiveValueTesting, e := s.frameBool(p, "str.aggressive_value_testing", false)
	if e != nil {
		return e
	}
	s.state.AggressiveValueTesting = vAggressiveValueTesting
	vAggressiveUnrollTesting, e := s.frameBool(p, "str.aggressive_unroll_testing", true)
	if e != nil {
		return e
	}
	s.state.AggressiveUnrollTesting = vAggressiveUnrollTesting
	vUseFastLengthTesterCache, e := s.frameBool(p, "str.fast_length_tester_cache", false)
	if e != nil {
		return e
	}
	s.state.UseFastLengthTesterCache = vUseFastLengthTesterCache
	vUseFastValueTesterCache, e := s.frameBool(p, "str.fast_value_tester_cache", true)
	if e != nil {
		return e
	}
	s.state.UseFastValueTesterCache = vUseFastValueTesterCache
	vStringConstantCache, e := s.frameBool(p, "str.string_constant_cache", true)
	if e != nil {
		return e
	}
	s.state.StringConstantCache = vStringConstantCache
	vFiniteOverlapModels, e := s.frameBool(p, "str.finite_overlap_models", false)
	if e != nil {
		return e
	}
	s.state.FiniteOverlapModels = vFiniteOverlapModels
	vUseBinarySearch, e := s.frameBool(p, "str.use_binary_search", false)
	if e != nil {
		return e
	}
	s.state.UseBinarySearch = vUseBinarySearch
	vBinarySearchInitialUpperBound, e := s.frameUInt(p, "str.binary_search_start", 64)
	if e != nil {
		return e
	}
	s.state.BinarySearchInitialUpperBound = vBinarySearchInitialUpperBound
	vOverlapTheoryAwarePriority, e := s.frameDouble(p, "str.overlap_priority", -0.1)
	if e != nil {
		return e
	}
	s.state.OverlapTheoryAwarePriority = vOverlapTheoryAwarePriority
	vRegexAutomata, e := s.frameBool(p, "str.regex_automata", true)
	if e != nil {
		return e
	}
	s.state.RegexAutomata = vRegexAutomata
	vRegexAutomataDifficultyThreshold, e := s.frameUInt(p, "str.regex_automata_difficulty_threshold", 1000)
	if e != nil {
		return e
	}
	s.state.RegexAutomataDifficultyThreshold = vRegexAutomataDifficultyThreshold
	vRegexAutomataIntersectionDifficultyThreshold, e := s.frameUInt(p, "str.regex_automata_intersection_difficulty_threshold", 1000)
	if e != nil {
		return e
	}
	s.state.RegexAutomataIntersectionDifficultyThreshold = vRegexAutomataIntersectionDifficultyThreshold
	vRegexAutomataFailedAutomatonThreshold, e := s.frameUInt(p, "str.regex_automata_failed_automaton_threshold", 10)
	if e != nil {
		return e
	}
	s.state.RegexAutomataFailedAutomatonThreshold = vRegexAutomataFailedAutomatonThreshold
	vRegexAutomataFailedIntersectionThreshold, e := s.frameUInt(p, "str.regex_automata_failed_intersection_threshold", 10)
	if e != nil {
		return e
	}
	s.state.RegexAutomataFailedIntersectionThreshold = vRegexAutomataFailedIntersectionThreshold
	vRegexAutomataLengthAttemptThreshold, e := s.frameUInt(p, "str.regex_automata_length_attempt_threshold", 10)
	if e != nil {
		return e
	}
	s.state.RegexAutomataLengthAttemptThreshold = vRegexAutomataLengthAttemptThreshold
	return nil
}
func (s *SMTStrParams) Close() error {
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
