// Source: src/smt/params/qi_params.h:31-109; src/smt/params/qi_params.cpp:23-64;
// src/smt/params/smt_params_helper.pyg:24-37; scripts/mk_genfile_common.py:895-939.
package smtcore

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// DefaultsInitialized records the actual scalar initializer frontier after
// both string constructors; State otherwise exposes diagnostic Go storage only.
type SMTQIState struct {
	DefaultsInitialized                                bool
	Ematching, EmatchingInitialized                    bool
	EagerThreshold, LazyThreshold                      float64
	MaxEagerMultipatterns, MaxLazyMultipatternMatching uint32
	Profile                                            bool
	ProfileFrequency, QuickChecker                     uint32
	LazyQuickChecker, PromoteUnsat                     bool
	MaxInstances                                       uint32
	LazyInstantiation, ConservativeFinalCheck, MBQI    bool
	MaxCexs, MaxCexsIncrement, MaxIterations           uint32
	Trace                                              bool
	ForceTemplate                                      uint32
	MBQIId                                             util.ParamsStringView
}
type SMTQIParams struct {
	util.ModuleParamsOwner
	self                     *SMTQIParams
	state                    SMTQIState
	cost, newGen             OwnedString
	header                   uint64
	completed                uint8
	constructorReady, closed bool
	constructorCause         error
}

func NewSMTQIParams(c *util.ParamsContext, p *util.ParamsRef, g *util.GlobalParamsService, b *arena.Budget) (*SMTQIParams, error) {
	if c == nil || g == nil || b == nil || g.CheckOwner(c, b) != nil {
		return nil, util.ErrParamsOwner
	}
	if e := c.Check(p); e != nil {
		return nil, e
	}
	if e := g.CheckOpen(); e != nil {
		return nil, e
	}
	n := uint64(reflect.TypeOf(SMTQIParams{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	s := &SMTQIParams{header: n}
	e := s.constructSource(c, p, g, b)
	return s, e
}

// Same source constructor engine: an actual containing header charges embedded fields once.
func (s *SMTQIParams) constructSource(c *util.ParamsContext, p *util.ParamsRef, g *util.GlobalParamsService, b *arena.Budget) error {
	if s == nil || s.self != nil {
		return util.ErrParamsOwner
	}
	*s = SMTQIParams{ModuleParamsOwner: util.ModuleParamsOwner{Context: c, Globals: g, Budget: b}, header: s.header}
	s.self = s
	s.ModuleParamsOwner.Self = &s.ModuleParamsOwner
	if e := s.cost.Construct(b, util.LiteralParamsStringView("(+ weight generation)")); e != nil {
		s.constructorCause = e
		return e
	}
	s.completed = 1
	if e := s.newGen.Construct(b, util.LiteralParamsStringView("cost")); e != nil {
		s.cost.SourceDestroy()
		s.constructorCause = e
		return e
	}
	s.completed = 2
	// Actual later scalar initializers run only after both earlier members.
	s.state = SMTQIState{EagerThreshold: 10, LazyThreshold: 20, MaxLazyMultipatternMatching: 2, ProfileFrequency: 4294967295, LazyQuickChecker: true, PromoteUnsat: true, MaxInstances: 4294967295, MBQI: true, MaxCexs: 1, MaxCexsIncrement: 1, MaxIterations: 1000, ForceTemplate: 10}
	s.state.DefaultsInitialized = true
	e := s.update(p)
	if e != nil {
		s.constructorCause = e
		// A younger stopped lexical DT protects older automatic members as well.
		if s.DestructorCause == nil {
			s.newGen.SourceDestroy()
			s.cost.SourceDestroy()
		}
	} else {
		s.constructorReady = true
	}
	return e
}
func (s *SMTQIParams) open() error {
	if s == nil || s.self != s || s.closed {
		return util.ErrParamsOwner
	}
	if s.Running {
		return util.ErrParamsPrecondition
	}
	return nil
}
func (s *SMTQIParams) State() (SMTQIState, error) {
	if e := s.open(); e != nil {
		return SMTQIState{}, e
	}
	return s.state, nil
}
func (s *SMTQIParams) CostText() (QITextView, error) {
	if e := s.open(); e != nil {
		return QITextView{}, e
	}
	if !s.constructorReady {
		return QITextView{}, util.ErrParamsPrecondition
	}
	return s.cost.View(), nil
}
func (s *SMTQIParams) SetEmatching(v bool) error {
	if e := s.open(); e != nil {
		return e
	}
	if !s.constructorReady {
		return util.ErrParamsPrecondition
	}
	s.state.Ematching = v
	s.state.EmatchingInitialized = true
	return nil
}
func (s *SMTQIParams) Update(p *util.ParamsRef) error {
	if e := s.open(); e != nil {
		return e
	}
	if !s.constructorReady {
		return util.ErrParamsPrecondition
	}
	return s.update(p)
}
func (s *SMTQIParams) update(local *util.ParamsRef) (result error) {
	if e := s.open(); e != nil {
		return e
	}
	if s.Helpers != nil || s.DestructorCause != nil || s.cost.Pending != nil {
		return util.ErrParamsPrecondition
	}
	s.Running = true
	defer func() { s.Running = false }()
	p, e := s.BeginHelper(local, "smt")
	if p != nil {
		defer func() { result = s.FinishHelper(p, result) }()
	}
	if e != nil {
		return e
	}
	b, e := s.FrameBool(p, "mbqi", true)
	if e != nil {
		return e
	}
	s.state.MBQI = b
	u, e := s.FrameUInt(p, "mbqi.max_cexs", 1)
	if e != nil {
		return e
	}
	s.state.MaxCexs = u
	u, e = s.FrameUInt(p, "mbqi.max_cexs_incr", 0)
	if e != nil {
		return e
	}
	s.state.MaxCexsIncrement = u
	u, e = s.FrameUInt(p, "mbqi.max_iterations", 1000)
	if e != nil {
		return e
	}
	s.state.MaxIterations = u
	b, e = s.FrameBool(p, "mbqi.trace", false)
	if e != nil {
		return e
	}
	s.state.Trace = b
	u, e = s.FrameUInt(p, "mbqi.force_template", 10)
	if e != nil {
		return e
	}
	s.state.ForceTemplate = u
	text, e := s.FrameStringView(p, "mbqi.id", "")
	if e != nil {
		return e
	}
	s.state.MBQIId = text
	b, e = s.FrameBool(p, "qi.profile", false)
	if e != nil {
		return e
	}
	s.state.Profile = b
	u, e = s.FrameUInt(p, "qi.profile_freq", 4294967295)
	if e != nil {
		return e
	}
	s.state.ProfileFrequency = u
	u, e = s.FrameUInt(p, "qi.max_instances", 4294967295)
	if e != nil {
		return e
	}
	s.state.MaxInstances = u
	d, e := s.FrameDouble(p, "qi.eager_threshold", 10)
	if e != nil {
		return e
	}
	s.state.EagerThreshold = d
	d, e = s.FrameDouble(p, "qi.lazy_threshold", 20)
	if e != nil {
		return e
	}
	s.state.LazyThreshold = d
	text, e = s.FrameStringView(p, "qi.cost", "(+ weight generation)")
	if e != nil {
		return e
	}
	if e = s.cost.Assign(text); e != nil {
		return e
	}
	u, e = s.FrameUInt(p, "qi.max_multi_patterns", 0)
	if e != nil {
		return e
	}
	s.state.MaxEagerMultipatterns = u
	return nil
}
func (s *SMTQIParams) Close() error {
	if s == nil || s.self != s {
		return util.ErrParamsOwner
	}
	if s.closed {
		return nil
	}
	if s.Running {
		return util.ErrParamsPrecondition
	}
	if e := s.CloseHelpers(); e != nil {
		return e
	}
	s.newGen.OwnAbort()
	s.cost.OwnAbort()
	s.Budget.Release(s.header)
	s.header = 0
	s.closed = true
	return nil
}

func (s *SMTQIParams) Display(out *util.NumericTextStream) error {
	if e := s.open(); e != nil {
		return e
	}
	if !s.constructorReady {
		return util.ErrParamsPrecondition
	}
	if e := out.AppendCString("m_qi_ematching="); e != nil {
		return e
	}
	if !s.state.EmatchingInitialized {
		return util.ErrRuntimeUninitialized
	}
	bEmatching := int64(0)
	if s.state.Ematching {
		bEmatching = 1
	}
	if e := out.AppendInt64(bEmatching); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_qi_cost="); e != nil {
		return e
	}
	if e := out.AppendOwnedString(&s.cost); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_qi_new_gen="); e != nil {
		return e
	}
	if e := out.AppendOwnedString(&s.newGen); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_qi_eager_threshold="); e != nil {
		return e
	}
	if e := out.AppendFloat64(s.state.EagerThreshold); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_qi_lazy_threshold="); e != nil {
		return e
	}
	if e := out.AppendFloat64(s.state.LazyThreshold); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_qi_max_eager_multipatterns="); e != nil {
		return e
	}
	if e := out.AppendUint32(s.state.MaxEagerMultipatterns); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_qi_max_lazy_multipattern_matching="); e != nil {
		return e
	}
	if e := out.AppendUint32(s.state.MaxLazyMultipatternMatching); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_qi_profile="); e != nil {
		return e
	}
	bProfile := int64(0)
	if s.state.Profile {
		bProfile = 1
	}
	if e := out.AppendInt64(bProfile); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_qi_profile_freq="); e != nil {
		return e
	}
	if e := out.AppendUint32(s.state.ProfileFrequency); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_qi_quick_checker="); e != nil {
		return e
	}
	if e := out.AppendInt32(int32(s.state.QuickChecker)); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_qi_lazy_quick_checker="); e != nil {
		return e
	}
	bLazyQuickChecker := int64(0)
	if s.state.LazyQuickChecker {
		bLazyQuickChecker = 1
	}
	if e := out.AppendInt64(bLazyQuickChecker); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_qi_promote_unsat="); e != nil {
		return e
	}
	bPromoteUnsat := int64(0)
	if s.state.PromoteUnsat {
		bPromoteUnsat = 1
	}
	if e := out.AppendInt64(bPromoteUnsat); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_qi_max_instances="); e != nil {
		return e
	}
	if e := out.AppendUint32(s.state.MaxInstances); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_qi_lazy_instantiation="); e != nil {
		return e
	}
	bLazyInstantiation := int64(0)
	if s.state.LazyInstantiation {
		bLazyInstantiation = 1
	}
	if e := out.AppendInt64(bLazyInstantiation); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_qi_conservative_final_check="); e != nil {
		return e
	}
	bConservativeFinalCheck := int64(0)
	if s.state.ConservativeFinalCheck {
		bConservativeFinalCheck = 1
	}
	if e := out.AppendInt64(bConservativeFinalCheck); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_mbqi="); e != nil {
		return e
	}
	bMBQI := int64(0)
	if s.state.MBQI {
		bMBQI = 1
	}
	if e := out.AppendInt64(bMBQI); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_mbqi_max_cexs="); e != nil {
		return e
	}
	if e := out.AppendUint32(s.state.MaxCexs); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_mbqi_max_cexs_incr="); e != nil {
		return e
	}
	if e := out.AppendUint32(s.state.MaxCexsIncrement); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_mbqi_max_iterations="); e != nil {
		return e
	}
	if e := out.AppendUint32(s.state.MaxIterations); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_mbqi_trace="); e != nil {
		return e
	}
	bTrace := int64(0)
	if s.state.Trace {
		bTrace = 1
	}
	if e := out.AppendInt64(bTrace); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_mbqi_force_template="); e != nil {
		return e
	}
	if e := out.AppendUint32(s.state.ForceTemplate); e != nil {
		return e
	}
	if e := out.AppendByte('\n'); e != nil {
		return e
	}
	if e := out.Flush(); e != nil {
		return e
	}
	if e := out.AppendCString("m_mbqi_id="); e != nil {
		return e
	}
	if e := out.AppendCStringView(s.state.MBQIId); e != nil {
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
