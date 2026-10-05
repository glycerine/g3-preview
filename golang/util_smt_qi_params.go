// Source: src/smt/params/qi_params.h:31-109; src/smt/params/qi_params.cpp:23-64;
// src/smt/params/smt_params_helper.pyg:24-37; scripts/mk_genfile_common.py:895-939.
// Source author: Leonardo de Moura (2008-06-15; 2012-12-02).
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"reflect"
)

// Concrete selected char std::string storage policy: inline15 plus NUL,
// constructor exact capacity, replacement geometric capacity, then dispose /
// pointer-capacity-length publication. This is not a native STL ABI promise.
type qiOwnedString struct {
	self                         *qiOwnedString
	budget                       *arena.Budget
	inline                       [16]byte
	external, pending            []byte
	length, capacity, generation uint64
	ready, closed                bool
}
type QITextView struct {
	owner              *qiOwnedString
	generation, length uint64
}

func (v QITextView) Len() uint64 { return v.length }
func (v QITextView) Byte(i uint64) (byte, error) {
	if v.owner == nil || v.owner.self != v.owner || v.owner.closed || v.generation != v.owner.generation {
		return 0, arena.ErrInvalidSlot
	}
	if i >= v.length {
		return 0, arena.ErrBounds
	}
	if v.owner.external != nil {
		return v.owner.external[i], nil
	}
	return v.owner.inline[i], nil
}

// String is a diagnostic caller-owned Go copy; source getter/display paths
// never call it or acquire ownership through it.
func (v QITextView) String() (string, error) {
	if v.length > uint64(^uint(0)>>1) {
		return "", arena.ErrBounds
	}
	a := make([]byte, int(v.length))
	for i := range a {
		b, e := v.Byte(uint64(i))
		if e != nil {
			return "", e
		}
		a[i] = b
	}
	return string(a), nil
}
func (s *qiOwnedString) construct(b *arena.Budget, v ParamsStringView) error {
	s.self = s
	s.budget = b
	s.capacity = 15
	s.generation = 1
	n, e := v.length()
	if e != nil {
		return e
	}
	if n > 15 {
		if n >= uint64(^uint(0)>>1) {
			return arena.ErrBounds
		}
		if e = b.Reserve(n + 1); e != nil {
			return e
		}
		s.external = make([]byte, int(n+1))
		s.capacity = n
	}
	for i := uint64(0); i < n; i++ {
		ch, e := v.Byte(i)
		if e != nil {
			s.sourceDestroy()
			return e
		}
		s.put(i, ch)
	}
	s.put(n, 0)
	s.length = n
	s.ready = true
	return nil
}
func (s *qiOwnedString) put(i uint64, ch byte) {
	if s.external != nil {
		s.external[i] = ch
	} else {
		s.inline[i] = ch
	}
}
func (s *qiOwnedString) assign(v ParamsStringView) error {
	if s.self != s || !s.ready || s.closed || s.pending != nil {
		return ErrParamsPrecondition
	}
	n, e := v.length()
	if e != nil {
		return e
	}
	if n <= s.capacity {
		for i := uint64(0); i < n; i++ {
			ch, e := v.Byte(i)
			if e != nil {
				return e
			}
			s.put(i, ch)
		}
		s.put(n, 0)
		s.length = n
		return nil
	}
	if s.generation == math.MaxUint64 || n >= uint64(^uint(0)>>1) {
		return arena.ErrBounds
	}
	capacity := n
	if capacity < 2*s.capacity {
		capacity = 2 * s.capacity
	}
	if capacity >= uint64(^uint(0)>>1) {
		return arena.ErrBounds
	}
	if e = s.budget.Reserve(capacity + 1); e != nil {
		return e
	}
	s.pending = make([]byte, int(capacity+1))
	for i := uint64(0); i < n; i++ {
		ch, e := v.Byte(i)
		if e != nil {
			return e
		}
		s.pending[i] = ch
	}
	s.pending[n] = 0
	s.budget.Release(uint64(len(s.external)))
	s.external = s.pending
	s.pending = nil
	s.capacity = capacity
	s.generation++
	s.length = n
	return nil
}
func (s *qiOwnedString) sourceDestroy() {
	if s.self != s || s.closed {
		return
	}
	s.budget.Release(uint64(len(s.external)))
	s.external = nil
	s.closed = true
}
func (s *qiOwnedString) ownAbort() {
	s.sourceDestroy()
	if s.pending != nil {
		s.budget.Release(uint64(len(s.pending)))
		s.pending = nil
	}
}
func (s *qiOwnedString) view() QITextView { return QITextView{s, s.generation, s.length} }
func (out *NumericTextStream) appendQIString(s *qiOwnedString) error {
	if e := out.lock(); e != nil {
		return e
	}
	defer out.mu.Unlock()
	if s.self != s || s.closed || !s.ready {
		return ErrParamsOwner
	}
	view := s.view()
	n := s.length // actual data/size fields acquired before insertion.
	if out.fileBad {
		return nil
	}
	pad, e := out.fieldPadding(n)
	if e != nil {
		return e
	}
	if out.state.Alignment != NumericLeft {
		if e = out.padding(pad); e != nil {
			return e
		}
	}
	for i := uint64(0); i < n; i++ {
		ch, e := view.Byte(i)
		if e != nil {
			return e
		}
		if e = out.emitByte(ch); e != nil {
			return e
		}
	}
	if out.state.Alignment == NumericLeft {
		return out.padding(pad)
	}
	return nil
}

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
	MBQIId                                             ParamsStringView
}
type SMTQIParams struct {
	smtModuleParamsOwner
	self                     *SMTQIParams
	state                    SMTQIState
	cost, newGen             qiOwnedString
	header                   uint64
	completed                uint8
	constructorReady, closed bool
	constructorCause         error
}

func NewSMTQIParams(c *ParamsContext, p *ParamsRef, g *GlobalParamsService, b *arena.Budget) (*SMTQIParams, error) {
	if c == nil || g == nil || b == nil || g.context != c || g.config.Budget != b {
		return nil, ErrParamsOwner
	}
	if e := c.check(p); e != nil {
		return nil, e
	}
	if e := g.open(); e != nil {
		return nil, e
	}
	n := uint64(reflect.TypeOf(SMTQIParams{}).Size())
	if e := b.Reserve(n); e != nil {
		return nil, e
	}
	s := &SMTQIParams{smtModuleParamsOwner: smtModuleParamsOwner{context: c, globals: g, budget: b}, header: n}
	s.self = s
	s.smtModuleParamsOwner.self = &s.smtModuleParamsOwner
	if e := s.cost.construct(b, literalParamsStringView("(+ weight generation)")); e != nil {
		s.constructorCause = e
		return s, e
	}
	s.completed = 1
	if e := s.newGen.construct(b, literalParamsStringView("cost")); e != nil {
		s.cost.sourceDestroy()
		s.constructorCause = e
		return s, e
	}
	s.completed = 2
	// Actual later scalar initializers run only after both earlier members.
	s.state = SMTQIState{EagerThreshold: 10, LazyThreshold: 20, MaxLazyMultipatternMatching: 2, ProfileFrequency: 4294967295, LazyQuickChecker: true, PromoteUnsat: true, MaxInstances: 4294967295, MBQI: true, MaxCexs: 1, MaxCexsIncrement: 1, MaxIterations: 1000, ForceTemplate: 10}
	s.state.DefaultsInitialized = true
	e := s.update(p)
	if e != nil {
		s.constructorCause = e
		// A younger stopped lexical DT protects older automatic members as well.
		if s.destructorCause == nil {
			s.newGen.sourceDestroy()
			s.cost.sourceDestroy()
		}
	} else {
		s.constructorReady = true
	}
	return s, e
}
func (s *SMTQIParams) open() error {
	if s == nil || s.self != s || s.closed {
		return ErrParamsOwner
	}
	if s.running {
		return ErrParamsPrecondition
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
		return QITextView{}, ErrParamsPrecondition
	}
	return s.cost.view(), nil
}
func (s *SMTQIParams) SetEmatching(v bool) error {
	if e := s.open(); e != nil {
		return e
	}
	if !s.constructorReady {
		return ErrParamsPrecondition
	}
	s.state.Ematching = v
	s.state.EmatchingInitialized = true
	return nil
}
func (s *SMTQIParams) Update(p *ParamsRef) error {
	if e := s.open(); e != nil {
		return e
	}
	if !s.constructorReady {
		return ErrParamsPrecondition
	}
	return s.update(p)
}
func (s *SMTQIParams) update(local *ParamsRef) (result error) {
	if e := s.open(); e != nil {
		return e
	}
	if s.helpers != nil || s.destructorCause != nil || s.cost.pending != nil {
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
	b, e := s.frameBool(p, "mbqi", true)
	if e != nil {
		return e
	}
	s.state.MBQI = b
	u, e := s.frameUInt(p, "mbqi.max_cexs", 1)
	if e != nil {
		return e
	}
	s.state.MaxCexs = u
	u, e = s.frameUInt(p, "mbqi.max_cexs_incr", 0)
	if e != nil {
		return e
	}
	s.state.MaxCexsIncrement = u
	u, e = s.frameUInt(p, "mbqi.max_iterations", 1000)
	if e != nil {
		return e
	}
	s.state.MaxIterations = u
	b, e = s.frameBool(p, "mbqi.trace", false)
	if e != nil {
		return e
	}
	s.state.Trace = b
	u, e = s.frameUInt(p, "mbqi.force_template", 10)
	if e != nil {
		return e
	}
	s.state.ForceTemplate = u
	text, e := s.frameStringView(p, "mbqi.id", "")
	if e != nil {
		return e
	}
	s.state.MBQIId = text
	b, e = s.frameBool(p, "qi.profile", false)
	if e != nil {
		return e
	}
	s.state.Profile = b
	u, e = s.frameUInt(p, "qi.profile_freq", 4294967295)
	if e != nil {
		return e
	}
	s.state.ProfileFrequency = u
	u, e = s.frameUInt(p, "qi.max_instances", 4294967295)
	if e != nil {
		return e
	}
	s.state.MaxInstances = u
	d, e := s.frameDouble(p, "qi.eager_threshold", 10)
	if e != nil {
		return e
	}
	s.state.EagerThreshold = d
	d, e = s.frameDouble(p, "qi.lazy_threshold", 20)
	if e != nil {
		return e
	}
	s.state.LazyThreshold = d
	text, e = s.frameStringView(p, "qi.cost", "(+ weight generation)")
	if e != nil {
		return e
	}
	if e = s.cost.assign(text); e != nil {
		return e
	}
	u, e = s.frameUInt(p, "qi.max_multi_patterns", 0)
	if e != nil {
		return e
	}
	s.state.MaxEagerMultipatterns = u
	return nil
}
func (s *SMTQIParams) Close() error {
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
	s.newGen.ownAbort()
	s.cost.ownAbort()
	s.budget.Release(s.header)
	s.header = 0
	s.closed = true
	return nil
}

func (s *SMTQIParams) Display(out *NumericTextStream) error {
	if e := s.open(); e != nil {
		return e
	}
	if !s.constructorReady {
		return ErrParamsPrecondition
	}
	if e := out.AppendCString("m_qi_ematching="); e != nil {
		return e
	}
	if !s.state.EmatchingInitialized {
		return memory.ErrRuntimeUninitialized
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
	if e := out.appendQIString(&s.cost); e != nil {
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
	if e := out.appendQIString(&s.newGen); e != nil {
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
