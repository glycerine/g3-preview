// Source: src/util/mpf.cpp:1590-1606; src/util/mpf.h:292.
package util

import (

	// ToStringRaw constructs the returned source string in fresh caller storage.
	// On failure CleanupRawString joins its actual temporaries and return slot.
	umemory "github.com/glycerine/g3/pkg/util/memory"
	utext "github.com/glycerine/g3/pkg/util/text"
)

func (m *ArbitraryFloatManager) ToStringRaw(x *ArbitraryFloat, out *utext.OwnedString) (cause error) {
	if e := m.valueGuard(x); e != nil {
		return e
	}
	if m.rawString.active {
		return ErrNumericPending
	}
	if out == nil || out.Self != nil {
		return utext.ErrOwnedStringOwner
	}
	s := &m.rawString
	*s = arbitraryFloatRawStringScope{manager: m, out: out, active: true, executing: true}
	if e := m.integer.BindCaller(&s.integerBinding, s, 27); e != nil {
		s.executing = false
		return e
	}
	if e := m.integer.Natural.BindCaller(&s.naturalBinding, s, 27); e != nil {
		s.executing = false
		return JoinNumericErrors(e, s.integerBinding.Unbind())
	}
	defer func() {
		s.executing = false
		s.bodyCause = cause
		cause = JoinNumericErrors(cause, m.finishRawStringSource())
		cause = JoinNumericErrors(cause, s.naturalBinding.Unbind())
		cause = JoinNumericErrors(cause, s.integerBinding.Unbind())
		if cause == nil {
			s.out = nil
			s.active = false
		}
	}()
	if e := out.Construct(m.budget, ""); e != nil {
		return e
	}
	s.outLive = true
	if e := out.AppendCString("["); e != nil {
		return e
	}
	sign := "+"
	if x.Sign() {
		sign = "-"
	}
	if e := out.AppendCString(sign); e != nil {
		return e
	}
	if e := out.AppendCString(" "); e != nil {
		return e
	}
	// The nested mpz::to_string owns a separate local ostringstream.
	if e := s.numberStream.Construct(m.budget); e != nil {
		return e
	}
	s.numberStreamLive = true
	if e := m.integer.insertSourceIntegerTracked(&s.numberStream, &x.significand, nil, &s.insertion); e != nil {
		return e
	}
	if e := s.numberStream.Str(&s.number); e != nil {
		return e
	}
	s.numberLive = true
	s.numberStream.Buffer.String.SourceDestroy()
	s.numberStreamLive = false
	if e := out.Append(s.number.View()); e != nil {
		return e
	}
	s.number.SourceDestroy()
	s.numberLive = false
	if e := out.AppendCString(" "); e != nil {
		return e
	}
	if e := s.exponentStream.Construct(m.budget); e != nil {
		return e
	}
	s.exponentStreamLive = true
	if e := s.exponentStream.AppendInt64(x.exponent); e != nil {
		return e
	}
	if e := s.exponentStream.Str(&s.exponent); e != nil {
		return e
	}
	s.exponentLive = true
	if e := out.Append(s.exponent.View()); e != nil {
		return e
	}
	s.exponent.SourceDestroy()
	s.exponentLive = false
	normal, e := m.IsNormal(x)
	if e != nil {
		return e
	}
	kind := " D"
	if normal {
		kind = " N"
	}
	if e := out.AppendCString(kind); e != nil {
		return e
	}
	return out.AppendCString("]")
}

type arbitraryFloatRawStringScope struct {
	manager                                                                 *ArbitraryFloatManager
	out                                                                     *utext.OwnedString
	number, exponent                                                        utext.OwnedString
	numberStream, exponentStream                                            utext.OwnedStringStream
	integerBinding, naturalBinding                                          NumericCallerBinding
	insertion                                                               NumeralInsertionReceipt
	numberLive, exponentLive, numberStreamLive, exponentStreamLive, outLive bool
	active, executing, finishing                                            bool
	bodyCause, sourceCause                                                  error
}

func (s *arbitraryFloatRawStringScope) valid() bool {
	return s != nil && s.manager != nil && s.manager.self == s.manager && s == &s.manager.rawString
}
func (s *arbitraryFloatRawStringScope) NumericExecuting() bool {
	return s.valid() && (s.executing || s.finishing)
}

func (m *ArbitraryFloatManager) finishRawStringSource() error {
	s := &m.rawString
	if s.executing || s.finishing {
		return ErrNumericPending
	}
	if !s.active {
		return nil
	}
	if s.sourceCause != nil {
		return s.sourceCause
	}
	s.finishing = true
	defer func() { s.finishing = false }()
	if e := s.insertion.SourceCause(); e != nil {
		s.sourceCause = e
		return e
	}
	for _, local := range [...]struct {
		str  *utext.OwnedString
		live *bool
	}{{&s.exponent, &s.exponentLive}, {&s.exponentStream.Buffer.String, &s.exponentStreamLive}, {&s.number, &s.numberLive}, {&s.numberStream.Buffer.String, &s.numberStreamLive}} {
		if *local.live {
			local.str.SourceDestroy()
			*local.live = false
		}
	}
	if s.bodyCause != nil && s.outLive {
		s.out.SourceDestroy()
		s.outLive = false
	}
	return nil
}

func (m *ArbitraryFloatManager) CleanupRawString() (cause error) {
	if m == nil || m.self != m {
		return umemory.ErrRuntimeOwner
	}
	s := &m.rawString
	if s.executing || s.finishing || m.valueExecuting || m.memory != nil && m.memory.FailureTransportActive() {
		return ErrNumericPending
	}
	if !s.active {
		return nil
	}
	if !s.valid() {
		return umemory.ErrRuntimeOwner
	}
	if e := m.integer.BindCaller(&s.integerBinding, s, 27); e != nil {
		return e
	}
	if e := m.integer.Natural.BindCaller(&s.naturalBinding, s, 27); e != nil {
		return JoinNumericErrors(e, s.integerBinding.Unbind())
	}
	s.finishing = true
	defer func() {
		s.finishing = false
		cause = JoinNumericErrors(cause, s.naturalBinding.Unbind())
		cause = JoinNumericErrors(cause, s.integerBinding.Unbind())
	}()
	if s.insertion.Active() {
		if e := s.cleanupInsertion(); e != nil {
			return e
		}
	}
	s.exponent.OwnAbort()
	s.exponentStream.OwnAbort()
	s.number.OwnAbort()
	s.numberStream.OwnAbort()
	if s.out != nil {
		s.out.OwnAbort()
	}
	s.exponentLive, s.exponentStreamLive, s.numberLive, s.numberStreamLive, s.outLive = false, false, false, false, false
	s.out = nil
	s.active = false
	return nil
}

// Owning abort joins the exact issued child without repeating a stopped
// source destructor or crediting an unreached source end.
func (s *arbitraryFloatRawStringScope) cleanupInsertion() error {
	return cleanupOwnedNumeralInsertion(&s.insertion, s.manager.integer.Natural)
}

func (s *arbitraryFloatRawStringScope) storageBytes() uint64 {
	var n uint64
	for _, str := range [...]*utext.OwnedString{&s.exponent, &s.exponentStream.Buffer.String, &s.number, &s.numberStream.Buffer.String, s.out} {
		if str != nil {
			n += str.StorageBytes()
		}
	}
	return n
}
