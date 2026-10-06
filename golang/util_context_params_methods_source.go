// Source: src/cmd_context/context_params.cpp:42-135,158-196;
// src/cmd_context/context_params.h:26-80.
// Source author: Leonardo de Moura (2012-12-01).
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"math"
	"reflect"
)

type contextParamsFrame struct {
	self                                                                     *contextParamsFrame
	owner                                                                    *ContextParams
	next                                                                     *contextParamsFrame
	invocation, header                                                       uint64
	key                                                                      qiOwnedString
	descriptor                                                               ParamDescrs
	buffer                                                                   StringBuffer64
	value                                                                    ParamsRef
	sourceReceipt                                                            ParamsSourceReceipt
	setReceipt                                                               ParamsSetReceipt
	decrement                                                                ParamsDecrementReceipt
	bodyCause, destructorCause, ownCause                                     error
	setAcknowledged, sourceAcknowledged                                      bool
	keyComplete, descriptorAttempted, bufferComplete, valueOwned, sourceDone bool
}

func (s *ContextParams) beginSourceFrame() (*contextParamsFrame, error) {
	if s.destructorCause != nil {
		return nil, ErrParamsPrecondition
	}
	if s.nextFrame == math.MaxUint64 {
		return nil, arena.ErrBounds
	}
	h := uint64(reflect.TypeOf(contextParamsFrame{}).Size())
	if e := s.budget.Reserve(h); e != nil {
		return nil, e
	}
	s.nextFrame++
	f := &contextParamsFrame{owner: s, next: s.frames, invocation: s.nextFrame, header: h}
	f.self = f
	s.frames = f
	return f, nil
}
func (s *ContextParams) dropSourceFrame(f *contextParamsFrame) {
	p := &s.frames
	for *p != nil {
		if *p == f {
			*p = f.next
			s.budget.Release(f.header)
			f.header = 0
			return
		}
		p = &(*p).next
	}
}
func (s *ContextParams) endSourceFrame(f *contextParamsFrame, body error) error {
	if f.bodyCause == nil {
		f.bodyCause = body
	}
	if body != nil && s.bodyCause == nil {
		s.bodyCause = body
	}
	if s.destructorCause != nil {
		return errors.Join(body, s.destructorCause)
	}
	if f.valueOwned {
		if e := s.context.DelTracked(&f.value, &f.decrement); e != nil {
			f.destructorCause = e
			s.destructorCause = e
			return errors.Join(body, e)
		}
		f.valueOwned = false
	}
	if f.bufferComplete {
		if e := f.buffer.Close(); e != nil {
			return errors.Join(body, e)
		}
	}
	if f.descriptorAttempted && f.descriptor.constructorReady {
		if e := f.descriptor.sourceFinalizeMembers(); e != nil {
			f.destructorCause = e
			s.destructorCause = e
			return errors.Join(body, e)
		}
	}
	if f.keyComplete {
		f.key.sourceDestroy()
	}
	f.sourceDone = true
	// An incomplete descriptor's actual copied-text context, or an exact failed
	// Params callee frame, remains for separate owning abort; no auto own cleanup.
	if (!f.descriptorAttempted || f.descriptor.constructorReady || f.descriptor.self == nil) && f.sourceReceipt.invocation == 0 && f.setReceipt.invocation == 0 {
		s.dropSourceFrame(f)
	}
	return body
}
func (s *ContextParams) closeSourceFrames() error {
	for s.frames != nil {
		f := s.frames
		if f.self != f || f.owner != s || f.invocation == 0 {
			return ErrParamsOwner
		}
		if f.setReceipt.invocation != 0 && !f.setAcknowledged {
			if e := s.context.CleanupSet(f.setReceipt); e != nil {
				if f.ownCause == nil {
					f.ownCause = e
				}
				return e
			}
			f.setAcknowledged = true
		}
		if f.sourceReceipt.invocation != 0 && !f.sourceAcknowledged {
			if e := s.context.CleanupSource(f.sourceReceipt); e != nil {
				if f.ownCause == nil {
					f.ownCause = e
				}
				return e
			}
			f.sourceAcknowledged = true
		}
		if f.valueOwned {
			var e error
			if f.decrement.invocation != 0 {
				e = s.context.CompleteDecrement(&f.value, f.decrement)
				if e == nil {
					f.value = ParamsRef{}
				}
			} else {
				e = s.context.DelTracked(&f.value, &f.decrement)
			}
			if e != nil {
				if f.ownCause == nil {
					f.ownCause = e
				}
				return e
			}
			f.valueOwned = false
		}
		if f.bufferComplete {
			if e := f.buffer.Close(); e != nil {
				return e
			}
		}
		if f.descriptorAttempted && f.descriptor.self != nil && !f.descriptor.closed {
			var e error
			if f.descriptor.constructorReady {
				e = f.descriptor.sourceFinalizeMembers()
			} else {
				e = f.descriptor.abortConstructor()
			}
			if e != nil {
				return e
			}
		}
		f.key.ownAbort()
		s.dropSourceFrame(f)
	}
	return nil
}
func (s *ContextParams) sourceCallReady() error {
	if e := s.open(); e != nil {
		return e
	}
	if !s.ready {
		return ErrParamsPrecondition
	}
	return nil
}
func (s *ContextParams) Set(param, value *string) error {
	var p, v ParamsStringView
	if param != nil {
		p = literalParamsStringView(*param)
	}
	if value != nil {
		v = literalParamsStringView(*value)
	}
	return s.SetViews(p, v)
}
func (s *ContextParams) SetViews(param, value ParamsStringView) (result error) {
	if e := s.sourceCallReady(); e != nil {
		return e
	}
	s.executing = true
	defer func() { s.executing = false }()
	f, e := s.beginSourceFrame()
	if e != nil {
		return e
	}
	defer func() { result = s.endSourceFrame(f, result) }()
	if e = f.key.construct(s.budget, param); e != nil {
		return e
	}
	f.keyComplete = true
	n := uint32(f.key.length)
	for i := uint32(0); i < n; i++ {
		ch := f.key.inline[0]
		if f.key.external != nil {
			ch = f.key.external[i]
		} else {
			ch = f.key.inline[i]
		}
		if ch >= 'A' && ch <= 'Z' {
			f.key.put(uint64(i), ch-'A'+'a')
		} else if ch == '-' {
			f.key.put(uint64(i), '_')
		}
	}
	switch {
	case f.key.matches("timeout"):
		return s.setSourceUInt(f, &s.state.Timeout, param, value)
	case f.key.matches("rlimit"):
		return s.setSourceUInt(f, &s.state.RLimit, param, value)
	case f.key.matches("type_check"), f.key.matches("well_sorted_check"):
		return s.setSourceBool(f, &s.state.WellSortedCheck, param, value)
	case f.key.matches("auto_config"):
		return s.setSourceBool(f, &s.state.AutoConfig, param, value)
	case f.key.matches("proof"):
		return s.setSourceBool(f, &s.state.Proof, param, value)
	case f.key.matches("model"):
		return s.setSourceBool(f, &s.state.Model, param, value)
	case f.key.matches("model_validate"):
		return s.setSourceBool(f, &s.state.ModelValidate, param, value)
	case f.key.matches("dump_models"):
		return s.setSourceBool(f, &s.state.DumpModels, param, value)
	case f.key.matches("trace"):
		return s.setSourceBool(f, &s.state.Trace, param, value)
	case f.key.matches("trace_file_name"):
		return s.traceFile.assign(value)
	case f.key.matches("dot_proof_file"):
		return s.dotProof.assign(value)
	case f.key.matches("unsat_core"):
		return s.setSourceBool(f, &s.state.UnsatCore, param, value)
	case f.key.matches("debug_ref_count"):
		return s.setSourceBool(f, &s.state.DebugRefCount, param, value)
	case f.key.matches("smtlib2_compliant"):
		return s.setSourceBool(f, &s.state.SMTLib2Compliant, param, value)
	}
	cfg, e := prepareSourceParamDescrs(s.context.symbols, ParamDescrsConfig{Budget: s.budget})
	if e != nil {
		return e
	}
	f.descriptor = ParamDescrs{manager: s.context.symbols, budget: s.budget}
	f.descriptor.self = &f.descriptor
	f.descriptorAttempted = true
	if e = f.descriptor.constructSourceMembers(cfg); e != nil {
		return e
	}
	if e = catalogCollectContextParams(&f.descriptor); e != nil {
		return e
	}
	f.initBuffer(s.budget)
	if e = f.buffer.AppendCString("unknown parameter '"); e != nil {
		return e
	}
	for i := uint64(0); i < f.key.length; i++ {
		ch := f.key.inline[0]
		if f.key.external != nil {
			ch = f.key.external[i]
		} else {
			ch = f.key.inline[i]
		}
		if e = f.buffer.AppendByte(ch); e != nil {
			return e
		}
	}
	if e = f.buffer.AppendCString("'\n"); e != nil {
		return e
	}
	if e = f.buffer.AppendCString("Legal parameters are:\n"); e != nil {
		return e
	}
	if e = f.descriptor.DisplayInto(&f.buffer, 2, false, false); e != nil {
		return e
	}
	text, e := f.buffer.CStr()
	if e != nil {
		return e
	}
	return NewDefaultException(text)
}
func (k *qiOwnedString) matches(v string) bool {
	if k.length != uint64(len(v)) {
		return false
	}
	for i := 0; i < len(v); i++ {
		b := k.inline[0]
		if k.external != nil {
			b = k.external[i]
		} else {
			b = k.inline[i]
		}
		if b != v[i] {
			return false
		}
	}
	return true
}
func (f *contextParamsFrame) initBuffer(b *arena.Budget) {
	f.buffer = StringBuffer64{state: stringBufferState{budget: b, capacity: 64, initial: 64, longBits: 64}}
	f.buffer.self = &f.buffer
	f.bufferComplete = true
}
func contextStringEqual(v ParamsStringView, text string) (bool, error) {
	for i := 0; i <= len(text); i++ {
		b, e := v.Byte(uint64(i))
		if e != nil {
			return false, e
		}
		want := byte(0)
		if i < len(text) {
			want = text[i]
		}
		if b != want {
			return false, nil
		}
	}
	return true, nil
}
func (s *ContextParams) setSourceBool(f *contextParamsFrame, opt *bool, param, value ParamsStringView) error {
	equal, e := contextStringEqual(value, "true")
	if e != nil {
		return e
	}
	if equal {
		*opt = true
		return nil
	}
	equal, e = contextStringEqual(value, "false")
	if e != nil {
		return e
	}
	if equal {
		*opt = false
		return nil
	}
	return s.sourceInvalidValue(f, "Boolean", param, value)
}
func (s *ContextParams) setSourceUInt(f *contextParamsFrame, opt *uint32, param, value ParamsStringView) error {
	n, e := value.length()
	if e != nil {
		return e
	}
	if n > math.MaxUint32 {
		return arena.ErrBounds
	}
	valid := true
	for i := uint32(0); uint64(i) < n; i++ {
		ch, e := value.Byte(uint64(i))
		if e != nil {
			return e
		}
		if ch < '0' || ch > '9' {
			valid = false
		}
	}
	if !valid {
		return s.sourceInvalidValue(f, "unsigned int", param, value)
	}
	// Selected LP64 strtol saturates positive overflow at LONG_MAX, errno ERANGE;
	// the source ignores errno and casts that actual signed result to unsigned32.
	number := uint64(0)
	for i := uint64(0); i < n; i++ {
		ch, e := value.Byte(i)
		if e != nil {
			return e
		}
		digit := uint64(ch - '0')
		if number > (math.MaxInt64-digit)/10 {
			number = math.MaxInt64
		} else {
			number = number*10 + digit
		}
	}
	*opt = uint32(number)
	return nil
}
func appendContextView(out *StringBuffer64, v ParamsStringView) error {
	// The selected char-pointer insertion obtains strlen before stream emission.
	n, e := v.length()
	if e != nil {
		return e
	}
	for i := uint64(0); i < n; i++ {
		ch, e := v.Byte(i)
		if e != nil {
			return e
		}
		if e = out.AppendByte(ch); e != nil {
			return e
		}
	}
	return nil
}
func (s *ContextParams) sourceInvalidValue(f *contextParamsFrame, kind string, param, value ParamsStringView) error {
	f.initBuffer(s.budget)
	if e := f.buffer.AppendCString("invalid value '"); e != nil {
		return e
	}
	if e := appendContextView(&f.buffer, value); e != nil {
		return e
	}
	if e := f.buffer.AppendCString("' for "); e != nil {
		return e
	}
	if e := f.buffer.AppendCString(kind); e != nil {
		return e
	}
	if e := f.buffer.AppendCString(" parameter '"); e != nil {
		return e
	}
	if e := appendContextView(&f.buffer, param); e != nil {
		return e
	}
	if e := f.buffer.AppendCString("'"); e != nil {
		return e
	}
	text, e := f.buffer.CStr()
	if e != nil {
		return e
	}
	return NewDefaultException(text)
}

// The returned value is one actually acquired reference; callers own its Del.
// A source failure retains the concrete lexical owner and exact child receipt
// until source destruction/explicit owning abort, never a reference-count guess.
func (s *ContextParams) MergeDefaultParams(p *ParamsRef) (value ParamsRef, result error) {
	if e := s.sourceCallReady(); e != nil {
		return ParamsRef{}, e
	}
	s.executing = true
	defer func() { s.executing = false }()
	f, e := s.beginSourceFrame()
	if e != nil {
		return ParamsRef{}, e
	}
	defer func() { result = s.endSourceFrame(f, result) }()
	if e = s.mergeSourceFrame(f, p); e != nil {
		return ParamsRef{}, e
	}
	value = f.value
	f.valueOwned = false // Actual successful return/NRVO transfer.
	return value, nil
}
func (s *ContextParams) mergeSourceFrame(f *contextParamsFrame, p *ParamsRef) error {
	change := false
	if !s.state.AutoConfig {
		found, e := s.context.ContainsName(p, dynSourceName("auto_config"))
		if e != nil {
			return e
		}
		change = !found
	}
	f.value = ParamsRef{owner: s.context.owner}
	if e := s.context.Assign(&f.value, p); e != nil {
		return e
	}
	f.valueOwned = true
	if !change {
		return nil
	}
	if e := s.context.SetBoolName(&f.value, dynSourceName("auto_config"), false); e != nil {
		// The failed COW copy may retain its actual POD NUM acquisition. Capture
		// its exact receipt before the completed temporary's source destructor.
		set, guard := s.context.StoppedSet(&f.value)
		if guard == nil {
			f.setReceipt = set
		} else if !errors.Is(guard, ErrParamsPrecondition) {
			return errors.Join(e, guard)
		}
		// This actual mutation began with a fresh copied reference, so an active
		// callee frame on its returned pointer can only be this reached COW action.
		r, guard := s.context.StoppedSource(&f.value)
		if guard == nil {
			f.sourceReceipt = r
		} else if !errors.Is(guard, ErrParamsPrecondition) {
			return errors.Join(e, guard)
		}
		return e
	}
	return nil
}
func dynSourceName(s string) *string { return &s }
func (s *ContextParams) GetSolverParams(m *ASTManager, p *ParamsRef, proofs, models, unsatCore *bool) (result error) {
	if e := s.sourceCallReady(); e != nil {
		return e
	}
	s.executing = true
	defer func() { s.executing = false }()
	f, e := s.beginSourceFrame()
	if e != nil {
		return e
	}
	defer func() { result = s.endSourceFrame(f, result) }()
	enabled, e := m.ProofsEnabled()
	if e != nil {
		return e
	}
	proof := false
	if enabled {
		proof, e = s.context.GetBoolName(p, dynSourceName("proof"), s.state.Proof)
		if e != nil {
			return e
		}
	}
	if proofs == nil {
		return ErrParamsOwner
	}
	*proofs = proof
	model, e := s.context.GetBoolName(p, dynSourceName("model"), s.state.Model)
	if e != nil {
		return e
	}
	if models == nil {
		return ErrParamsOwner
	}
	*models = model
	core, e := s.context.GetBoolName(p, dynSourceName("unsat_core"), s.state.UnsatCore)
	if e != nil {
		return e
	}
	if unsatCore == nil {
		return ErrParamsOwner
	}
	*unsatCore = core
	if e = s.mergeSourceFrame(f, p); e != nil {
		return e
	}
	if e = s.context.Assign(p, &f.value); e != nil {
		// Caller destination remains caller-owned. This receipt acknowledges only
		// the callee's actual extra SRC acquisition; it never grants ownership of p
		// or adopts/continues that caller's stopped last-reference destructor.
		r, guard := s.context.StoppedSource(p)
		if guard == nil {
			f.sourceReceipt = r
		} else if !errors.Is(guard, ErrParamsPrecondition) {
			return errors.Join(e, guard)
		}
		return e
	}
	return nil
}
