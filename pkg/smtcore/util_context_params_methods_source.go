// Source: src/cmd_context/context_params.cpp:42-135,158-196;
// src/cmd_context/context_params.h:26-80.
package smtcore

import (
	"errors"
	"math"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util"
)

type contextParamsFrame struct {
	self                                                                     *contextParamsFrame
	owner                                                                    *ContextParams
	next                                                                     *contextParamsFrame
	invocation, header                                                       uint64
	key                                                                      OwnedString
	descriptor                                                               util.ParamDescrs
	buffer                                                                   util.StringBuffer64
	value                                                                    util.ParamsRef
	sourceReceipt                                                            util.ParamsSourceReceipt
	setReceipt                                                               util.ParamsSetReceipt
	decrement                                                                util.ParamsDecrementReceipt
	bodyCause, destructorCause, ownCause                                     error
	setAcknowledged, sourceAcknowledged                                      bool
	keyComplete, descriptorAttempted, bufferComplete, valueOwned, sourceDone bool
}

func (s *ContextParams) beginSourceFrame() (*contextParamsFrame, error) {
	if s.destructorCause != nil {
		return nil, util.ErrParamsPrecondition
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
	if f.descriptorAttempted && f.descriptor.ConstructorReady() {
		if e := f.descriptor.SourceFinalizeMembers(); e != nil {
			f.destructorCause = e
			s.destructorCause = e
			return errors.Join(body, e)
		}
	}
	if f.keyComplete {
		f.key.SourceDestroy()
	}
	f.sourceDone = true
	// An incomplete descriptor's actual copied-text context, or an exact failed
	// Params callee frame, remains for separate owning abort; no auto own cleanup.
	if (!f.descriptorAttempted || f.descriptor.ConstructorReady() || !f.descriptor.Initialized()) && f.sourceReceipt.Invocation() == 0 && f.setReceipt.Invocation() == 0 {
		s.dropSourceFrame(f)
	}
	return body
}
func (s *ContextParams) closeSourceFrames() error {
	for s.frames != nil {
		f := s.frames
		if f.self != f || f.owner != s || f.invocation == 0 {
			return util.ErrParamsOwner
		}
		if f.setReceipt.Invocation() != 0 && !f.setAcknowledged {
			if e := s.context.CleanupSet(f.setReceipt); e != nil {
				if f.ownCause == nil {
					f.ownCause = e
				}
				return e
			}
			f.setAcknowledged = true
		}
		if f.sourceReceipt.Invocation() != 0 && !f.sourceAcknowledged {
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
			if f.decrement.Invocation() != 0 {
				e = s.context.CompleteDecrement(&f.value, f.decrement)
				if e == nil {
					f.value = util.ParamsRef{}
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
		if f.descriptorAttempted && f.descriptor.Initialized() && !f.descriptor.Closed() {
			var e error
			if f.descriptor.ConstructorReady() {
				e = f.descriptor.SourceFinalizeMembers()
			} else {
				e = f.descriptor.AbortConstructor()
			}
			if e != nil {
				return e
			}
		}
		f.key.OwnAbort()
		s.dropSourceFrame(f)
	}
	return nil
}
func (s *ContextParams) sourceCallReady() error {
	if e := s.open(); e != nil {
		return e
	}
	if !s.ready {
		return util.ErrParamsPrecondition
	}
	return nil
}
func (s *ContextParams) Set(param, value *string) error {
	var p, v util.ParamsStringView
	if param != nil {
		p = util.LiteralParamsStringView(*param)
	}
	if value != nil {
		v = util.LiteralParamsStringView(*value)
	}
	return s.SetViews(p, v)
}
func (s *ContextParams) SetViews(param, value util.ParamsStringView) (result error) {
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
	if e = f.key.Construct(s.budget, param); e != nil {
		return e
	}
	f.keyComplete = true
	n := uint32(f.key.Length)
	for i := uint32(0); i < n; i++ {
		ch := f.key.Inline[0]
		if f.key.External != nil {
			ch = f.key.External[i]
		} else {
			ch = f.key.Inline[i]
		}
		if ch >= 'A' && ch <= 'Z' {
			f.key.Put(uint64(i), ch-'A'+'a')
		} else if ch == '-' {
			f.key.Put(uint64(i), '_')
		}
	}
	switch {
	case f.key.Matches("timeout"):
		return s.setSourceUInt(f, &s.state.Timeout, param, value)
	case f.key.Matches("rlimit"):
		return s.setSourceUInt(f, &s.state.RLimit, param, value)
	case f.key.Matches("type_check"), f.key.Matches("well_sorted_check"):
		return s.setSourceBool(f, &s.state.WellSortedCheck, param, value)
	case f.key.Matches("auto_config"):
		return s.setSourceBool(f, &s.state.AutoConfig, param, value)
	case f.key.Matches("proof"):
		return s.setSourceBool(f, &s.state.Proof, param, value)
	case f.key.Matches("model"):
		return s.setSourceBool(f, &s.state.Model, param, value)
	case f.key.Matches("model_validate"):
		return s.setSourceBool(f, &s.state.ModelValidate, param, value)
	case f.key.Matches("dump_models"):
		return s.setSourceBool(f, &s.state.DumpModels, param, value)
	case f.key.Matches("trace"):
		return s.setSourceBool(f, &s.state.Trace, param, value)
	case f.key.Matches("trace_file_name"):
		return s.traceFile.Assign(value)
	case f.key.Matches("dot_proof_file"):
		return s.dotProof.Assign(value)
	case f.key.Matches("unsat_core"):
		return s.setSourceBool(f, &s.state.UnsatCore, param, value)
	case f.key.Matches("debug_ref_count"):
		return s.setSourceBool(f, &s.state.DebugRefCount, param, value)
	case f.key.Matches("smtlib2_compliant"):
		return s.setSourceBool(f, &s.state.SMTLib2Compliant, param, value)
	}
	f.descriptorAttempted = true
	if e = f.descriptor.InitEmbeddedSource(s.context.Symbols(), util.ParamDescrsConfig{Budget: s.budget}); e != nil {
		return e
	}
	if e = catalogCollectContextParams(&f.descriptor); e != nil {
		return e
	}
	f.initBuffer(s.budget)
	if e = f.buffer.AppendCString("unknown parameter '"); e != nil {
		return e
	}
	for i := uint64(0); i < f.key.Length; i++ {
		ch := f.key.Inline[0]
		if f.key.External != nil {
			ch = f.key.External[i]
		} else {
			ch = f.key.Inline[i]
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
	return util.NewDefaultException(text)
}

func (f *contextParamsFrame) initBuffer(b *arena.Budget) {
	f.buffer = util.StringBuffer64{State: util.StringBufferState{Budget: b, Capacity: 64, Initial: 64, LongBits: 64}}
	f.buffer.Self = &f.buffer
	f.bufferComplete = true
}
func contextStringEqual(v util.ParamsStringView, text string) (bool, error) {
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
func (s *ContextParams) setSourceBool(f *contextParamsFrame, opt *bool, param, value util.ParamsStringView) error {
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
func (s *ContextParams) setSourceUInt(f *contextParamsFrame, opt *uint32, param, value util.ParamsStringView) error {
	n, e := value.Length()
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
func appendContextView(out *util.StringBuffer64, v util.ParamsStringView) error {
	// The selected char-pointer insertion obtains strlen before stream emission.
	n, e := v.Length()
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
func (s *ContextParams) sourceInvalidValue(f *contextParamsFrame, kind string, param, value util.ParamsStringView) error {
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
	return util.NewDefaultException(text)
}

// The returned value is one actually acquired reference; callers own its Del.
// A source failure retains the concrete lexical owner and exact child receipt
// until source destruction/explicit owning abort, never a reference-count guess.
func (s *ContextParams) MergeDefaultParams(p *util.ParamsRef) (value util.ParamsRef, result error) {
	if e := s.sourceCallReady(); e != nil {
		return util.ParamsRef{}, e
	}
	s.executing = true
	defer func() { s.executing = false }()
	f, e := s.beginSourceFrame()
	if e != nil {
		return util.ParamsRef{}, e
	}
	defer func() { result = s.endSourceFrame(f, result) }()
	if e = s.mergeSourceFrame(f, p); e != nil {
		return util.ParamsRef{}, e
	}
	value = f.value
	f.valueOwned = false // Actual successful return/NRVO transfer.
	return value, nil
}
func (s *ContextParams) mergeSourceFrame(f *contextParamsFrame, p *util.ParamsRef) error {
	change := false
	if !s.state.AutoConfig {
		found, e := s.context.ContainsName(p, dynSourceName("auto_config"))
		if e != nil {
			return e
		}
		change = !found
	}
	f.value = s.context.EmptyValue()
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
		} else if !errors.Is(guard, util.ErrParamsPrecondition) {
			return errors.Join(e, guard)
		}
		// This actual mutation began with a fresh copied reference, so an active
		// callee frame on its returned pointer can only be this reached COW action.
		r, guard := s.context.StoppedSource(&f.value)
		if guard == nil {
			f.sourceReceipt = r
		} else if !errors.Is(guard, util.ErrParamsPrecondition) {
			return errors.Join(e, guard)
		}
		return e
	}
	return nil
}
func dynSourceName(s string) *string { return &s }
func (s *ContextParams) GetSolverParams(m *ast.ASTManager, p *util.ParamsRef, proofs, models, unsatCore *bool) (result error) {
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
		return util.ErrParamsOwner
	}
	*proofs = proof
	model, e := s.context.GetBoolName(p, dynSourceName("model"), s.state.Model)
	if e != nil {
		return e
	}
	if models == nil {
		return util.ErrParamsOwner
	}
	*models = model
	core, e := s.context.GetBoolName(p, dynSourceName("unsat_core"), s.state.UnsatCore)
	if e != nil {
		return e
	}
	if unsatCore == nil {
		return util.ErrParamsOwner
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
		} else if !errors.Is(guard, util.ErrParamsPrecondition) {
			return errors.Join(e, guard)
		}
		return e
	}
	return nil
}
