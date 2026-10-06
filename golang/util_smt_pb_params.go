// Source: src/smt/params/theory_pb_params.h:25-39;
// src/smt/params/theory_pb_params.cpp:22-37; src/smt/params/smt_params_helper.pyg:55-58.
// Source author: Nikolaj Bjorner (2014-01-01).
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

type SMTPBState struct {
	ConflictFrequency                                  uint32
	LearnComplements, EnableCompilation, EnableSimplex bool
}
type SMTPBParams struct {
	smtModuleParamsOwner
	self   *SMTPBParams
	state  SMTPBState
	header uint64
	closed bool
}

func NewSMTPBParams(b *arena.Budget) (*SMTPBParams, error) {
	if b == nil {
		return nil, arena.ErrConfig
	}
	n := uint64(reflect.TypeOf(SMTPBParams{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	s := &SMTPBParams{header: n}
	e := s.constructSource(b)
	return s, e
}

// Same source constructor engine: an actual containing header charges embedded fields once.
func (s *SMTPBParams) constructSource(b *arena.Budget) error {
	if s == nil || s.self != nil {
		return ErrParamsOwner
	}
	*s = SMTPBParams{smtModuleParamsOwner: smtModuleParamsOwner{budget: b}, state: SMTPBState{1000, true, true, false}, header: s.header}
	s.self = s
	s.smtModuleParamsOwner.self = &s.smtModuleParamsOwner
	// Source accepts an unused params_ref: it does not call updt_params.
	return nil
}
func (s *SMTPBParams) open() error {
	if s == nil || s.self != s || s.closed {
		return ErrParamsOwner
	}
	if s.running {
		return ErrParamsPrecondition
	}
	return nil
}
func (s *SMTPBParams) State() (SMTPBState, error) {
	if e := s.open(); e != nil {
		return SMTPBState{}, e
	}
	return s.state, nil
}
func (s *SMTPBParams) Update(c *ParamsContext, p *ParamsRef, g *GlobalParamsService) (result error) {
	if e := s.open(); e != nil {
		return e
	}
	if e := s.bind(c, g); e != nil {
		return e
	}
	s.running = true
	defer func() { s.running = false }()
	f, e := s.beginHelper(p, "smt")
	if f != nil {
		defer func() { result = s.finishHelper(f, result) }()
	}
	if e != nil {
		return e
	}
	frequency, e := s.frameUInt(f, "pb.conflict_frequency", 1000)
	if e != nil {
		return e
	}
	s.state.ConflictFrequency = frequency
	learn, e := s.frameBool(f, "pb.learn_complements", true)
	if e != nil {
		return e
	}
	s.state.LearnComplements = learn
	compile, e := s.frameBool(f, "pb.enable_compilation", true)
	if e != nil {
		return e
	}
	s.state.EnableCompilation = compile
	simplex, e := s.frameBool(f, "pb.enable_simplex", false)
	if e != nil {
		return e
	}
	s.state.EnableSimplex = simplex
	return nil
}
func (s *SMTPBParams) Close() error {
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
func (s *SMTPBParams) Display(out *NumericTextStream) error {
	if e := s.open(); e != nil {
		return e
	}
	if e := out.AppendCString("m_pb_conflict_frequency="); e != nil {
		return e
	}
	if e := out.AppendUint32(s.state.ConflictFrequency); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_pb_learn_complements="); e != nil {
		return e
	}
	bLearnComplements := int64(0)
	if s.state.LearnComplements {
		bLearnComplements = 1
	}
	if e := out.AppendInt64(bLearnComplements); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_pb_enable_compilation="); e != nil {
		return e
	}
	bEnableCompilation := int64(0)
	if s.state.EnableCompilation {
		bEnableCompilation = 1
	}
	if e := out.AppendInt64(bEnableCompilation); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_pb_enable_simplex="); e != nil {
		return e
	}
	bEnableSimplex := int64(0)
	if s.state.EnableSimplex {
		bEnableSimplex = 1
	}
	if e := out.AppendInt64(bEnableSimplex); e != nil {
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
